package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"edi/defaults"
	"edi/internal/buffer"
	"edi/internal/config"
	"edi/internal/git"
	"edi/internal/plugin"
	"edi/internal/syntax"
	"edi/internal/theme"
	"edi/internal/tree"
	"edi/internal/updater"
	"edi/internal/version"

	"github.com/gdamore/tcell/v2"
)

// PromptType define el tipo de entrada modal inferior activa.
type PromptType int

const (
	PromptNone PromptType = iota
	PromptFind
	PromptGoto
	PromptSaveAs
	PromptPalette
	PromptFileFinder
)

// PaletteItem representa un comando en la paleta de acciones rápidas.
type PaletteItem struct {
	Title       string
	Action      string
	Description string
}

// Editor representa la sesión activa del editor de texto con soporte multibuffer, modales y explorador.
type Editor struct {
	Screen          tcell.Screen
	Buffers         []*buffer.Buffer
	ActiveBufferIdx int
	Buffer          *buffer.Buffer // Puntero al buffer activo actual
	Config          *config.EditorConfig
	Keybindings     map[string]string
	Theme           *theme.CompiledTheme
	Syntaxes        []*syntax.CompiledSyntax
	CurrentSyntax   *syntax.CompiledSyntax
	Plugins         []*plugin.Plugin

	Running       bool
	StatusMessage string
	StatusExpires time.Time
	QuitWarned    bool

	// Explorador de archivos en panel lateral (Sidebar Tree)
	SidebarOpen  bool
	SidebarWidth int
	SidebarFocus bool
	FileTree     *tree.FileTree
	GitManager   *git.Manager

	// Selección y portapapeles
	Selecting       bool
	SelectAnchorRow int
	SelectAnchorCol int
	Clipboard       string

	// Modales y prompts
	PromptMode     PromptType
	PromptText     string
	PromptCursor   int
	PaletteItems   []PaletteItem
	PaletteIndex   int
	PaletteScroll  int
	FilteredItems  []PaletteItem
	LastSearchTerm string

	// Buscador de archivos
	WorkspaceFiles   []string
	FilteredFiles    []string
	FileFinderIdx    int
	FileFinderScroll int
}

// NewEditor crea e inicializa una nueva instancia del editor.
func NewEditor(screen tcell.Screen, buf *buffer.Buffer, cfg *config.EditorConfig) (*Editor, error) {
	keybindings, err := config.LoadKeybindings()
	if err != nil {
		keybindings = make(map[string]string)
	}

	th, err := theme.LoadTheme(cfg.Theme)
	if err != nil {
		return nil, fmt.Errorf("error cargando tema %s: %w", cfg.Theme, err)
	}

	syntaxes := syntax.LoadAllSyntaxes()
	plugins := plugin.LoadAllPlugins()

	initialDir := ""
	if buf != nil && buf.FilePath != "" {
		initialDir = filepath.Dir(buf.FilePath)
	} else {
		initialDir, _ = os.Getwd()
	}
	ft, _ := tree.NewFileTree(initialDir)

	var gitMgr *git.Manager
	if cfg.GitIntegration {
		gitMgr = git.NewManager(initialDir)
	}

	ed := &Editor{
		Screen:          screen,
		Buffers:         []*buffer.Buffer{buf},
		ActiveBufferIdx: 0,
		Buffer:          buf,
		Config:          cfg,
		Keybindings:     keybindings,
		Theme:           th,
		Syntaxes:        syntaxes,
		Plugins:         plugins,
		Running:         true,
		PromptMode:      PromptNone,
		PaletteItems:    buildDefaultPalette(),
		SidebarOpen:     false,
		SidebarWidth:    26,
		SidebarFocus:    false,
		FileTree:        ft,
		GitManager:      gitMgr,
	}

	// Incorporar comandos expuestos por plugins
	for _, p := range plugins {
		for _, cmd := range p.Commands {
			ed.PaletteItems = append(ed.PaletteItems, PaletteItem{
				Title:       cmd.Title,
				Action:      "plugin:" + cmd.ID,
				Description: fmt.Sprintf("[%s] %s", p.Name, cmd.Command),
			})
		}
	}

	ed.detectCurrentSyntax()
	return ed, nil
}

func buildDefaultPalette() []PaletteItem {
	return []PaletteItem{
		{Title: "Documentación y Ayuda", Action: "help:docs", Description: "F1"},
		{Title: "Alternar explorador de archivos", Action: "view:toggle_tree", Description: "Ctrl+B"},
		{Title: "Alternar posición del explorador (Izquierda/Derecha)", Action: "view:toggle_sidebar_pos", Description: "Configurar posición"},
		{Title: "Enfocar explorador / editor", Action: "tree:focus", Description: "Ctrl+E"},
		{Title: "Buscar archivo por nombre en workspace...", Action: "file:find", Description: "Ctrl+O (Fuzzy Finder)"},
		{Title: "Nueva pestaña", Action: "buffer:new", Description: "Ctrl+N"},
		{Title: "Cerrar pestaña", Action: "buffer:close", Description: "Ctrl+W"},
		{Title: "Siguiente pestaña", Action: "buffer:next", Description: "Alt+Right / Ctrl+Tab"},
		{Title: "Pestaña anterior", Action: "buffer:prev", Description: "Alt+Left"},
		{Title: "Guardar archivo", Action: "buffer:save", Description: "Ctrl+S / F2"},
		{Title: "Guardar como...", Action: "buffer:save_as", Description: "Guardar con nuevo nombre"},
		{Title: "Ir a línea...", Action: "buffer:goto_line", Description: "Ctrl+G"},
		{Title: "Buscar texto...", Action: "buffer:find", Description: "Ctrl+F"},
		{Title: "Copiar selección", Action: "selection:copy", Description: "Ctrl+C"},
		{Title: "Cortar selección", Action: "selection:cut", Description: "Ctrl+X"},
		{Title: "Pegar portapapeles", Action: "selection:paste", Description: "Ctrl+V"},
		{Title: "Deshacer (Undo)", Action: "buffer:undo", Description: "Ctrl+Z"},
		{Title: "Rehacer (Redo)", Action: "buffer:redo", Description: "Ctrl+Y"},
		{Title: "Duplicar línea", Action: "line:duplicate", Description: "Ctrl+D"},
		{Title: "Eliminar línea", Action: "line:delete", Description: "Ctrl+K"},
		{Title: "Mover línea arriba", Action: "line:move_up", Description: "Alt+Up / Ctrl+Up"},
		{Title: "Mover línea abajo", Action: "line:move_down", Description: "Alt+Down / Ctrl+Down"},
		{Title: "Cambiar tema: Dracula", Action: "theme:dracula", Description: "Tema oscuro clásico"},
		{Title: "Cambiar tema: Monokai", Action: "theme:monokai", Description: "Tema oscuro retro"},
		{Title: "Alternar números de línea", Action: "toggle:line_numbers", Description: "Mostrar/Ocultar"},
		{Title: "Alternar resaltado línea actual", Action: "toggle:current_line", Description: "Mostrar/Ocultar"},
		{Title: "Git: Ver Diff del archivo actual", Action: "git:diff_file", Description: "Comparar con HEAD"},
		{Title: "Git: Ver Diff de todo el proyecto", Action: "git:diff_project", Description: "Todos los cambios"},
		{Title: "Git: Archivos modificados (Status)", Action: "git:status_files", Description: "Lista de archivos cambiados"},
		{Title: "Git: Refrescar estado", Action: "git:refresh", Description: "Actualizar cambios"},
		{Title: fmt.Sprintf("Acerca de edi (%s)", version.Version), Action: "app:about", Description: "Versión e información"},
		{Title: "Buscar actualizaciones de edi", Action: "app:check_update", Description: "Consultar última versión en GitHub"},
		{Title: "Salir de edi", Action: "app:quit", Description: "Ctrl+Q"},
	}
}

// OpenFolder carga una carpeta en el explorador de archivos y la abre en el panel lateral.
func (e *Editor) OpenFolder(dirPath string) {
	ft, err := tree.NewFileTree(dirPath)
	if err == nil {
		e.FileTree = ft
		e.SidebarOpen = true
		e.SidebarFocus = true
		if e.GitManager != nil {
			e.GitManager.RefreshAsync(dirPath, func() {
				e.Screen.PostEvent(tcell.NewEventInterrupt(nil))
			})
		}
		e.SetStatus(fmt.Sprintf("Carpeta abierta: %s (↑/↓ navegar, ↵ abrir, Tab alternar)", filepath.Base(dirPath)), 4*time.Second)
	}
}

// OpenHelp carga la documentación completa embebida en una nueva pestaña.
func (e *Editor) OpenHelp() {
	helpData, err := defaults.FS.ReadFile("help.md")
	if err != nil {
		e.SetStatus("Error cargando documentación", 2*time.Second)
		return
	}

	for i, b := range e.Buffers {
		if b.FilePath == "ayuda.md" {
			e.SwitchToBuffer(i)
			return
		}
	}

	lines := strings.Split(string(helpData), "\n")
	helpBuf := &buffer.Buffer{
		FilePath: "ayuda.md",
		Lines:    lines,
		Modified: false,
	}

	e.Buffers = append(e.Buffers, helpBuf)
	e.SwitchToBuffer(len(e.Buffers) - 1)
	e.SetStatus("Ayuda abierta (Usa Ctrl+F para buscar, Ctrl+W para cerrar)", 3*time.Second)
}

func (e *Editor) detectCurrentSyntax() {
	if e.Buffer != nil && e.Buffer.FilePath != "" {
		e.CurrentSyntax = syntax.DetectSyntax(e.Buffer.FilePath, e.Syntaxes)
	} else {
		e.CurrentSyntax = nil
	}
}

// SwitchToBuffer cambia el buffer activo al índice indicado.
func (e *Editor) SwitchToBuffer(idx int) {
	if idx >= 0 && idx < len(e.Buffers) {
		e.ActiveBufferIdx = idx
		e.Buffer = e.Buffers[idx]
		e.detectCurrentSyntax()
		e.Selecting = false
	}
}

// CloseActiveBuffer cierra el buffer actual o reinicia si es el único.
func (e *Editor) CloseActiveBuffer() {
	if len(e.Buffers) <= 1 {
		if e.Buffer.Modified && !e.QuitWarned {
			e.QuitWarned = true
			e.SetStatus("¡Cambios sin guardar! Presiona Ctrl+W de nuevo para descartar", 3*time.Second)
			return
		}
		e.Buffers[0] = buffer.NewEmptyBuffer()
		e.Buffer = e.Buffers[0]
		e.detectCurrentSyntax()
		e.QuitWarned = false
		e.SetStatus("Pestaña reiniciada", 1*time.Second)
		return
	}

	if e.Buffer.Modified && !e.QuitWarned {
		e.QuitWarned = true
		e.SetStatus("¡Cambios sin guardar! Presiona Ctrl+W de nuevo para cerrar pestaña", 3*time.Second)
		return
	}

	e.Buffers = append(e.Buffers[:e.ActiveBufferIdx], e.Buffers[e.ActiveBufferIdx+1:]...)
	if e.ActiveBufferIdx >= len(e.Buffers) {
		e.ActiveBufferIdx = len(e.Buffers) - 1
	}
	e.Buffer = e.Buffers[e.ActiveBufferIdx]
	e.detectCurrentSyntax()
	e.QuitWarned = false
	e.Selecting = false
}

// OpenFile abre un archivo en una nueva pestaña o salta a ella si ya está abierto.
func (e *Editor) OpenFile(path string) {
	clean := filepath.Clean(path)
	cleanAbs, _ := filepath.Abs(clean)
	for i, b := range e.Buffers {
		bClean := filepath.Clean(b.FilePath)
		if bClean == clean {
			e.SwitchToBuffer(i)
			e.SetStatus(fmt.Sprintf("Cambiado a pestaña: %s", filepath.Base(clean)), 2*time.Second)
			return
		}
		if cleanAbs != "" {
			if bAbs, err := filepath.Abs(bClean); err == nil && bAbs == cleanAbs {
				e.SwitchToBuffer(i)
				e.SetStatus(fmt.Sprintf("Cambiado a pestaña: %s", filepath.Base(clean)), 2*time.Second)
				return
			}
		}
	}

	newBuf, err := buffer.NewBufferFromFile(clean)
	if err != nil {
		e.SetStatus(fmt.Sprintf("Error abriendo archivo: %v", err), 3*time.Second)
		return
	}

	// Reemplazar buffer único si está vacío y sin título
	if len(e.Buffers) == 1 && e.Buffer.FilePath == "" && !e.Buffer.Modified && len(e.Buffer.Lines) <= 1 && e.Buffer.Lines[0] == "" {
		e.Buffers[0] = newBuf
		e.Buffer = newBuf
		e.detectCurrentSyntax()
		e.SetStatus(fmt.Sprintf("Abierto '%s'", filepath.Base(clean)), 2*time.Second)
		return
	}

	e.Buffers = append(e.Buffers, newBuf)
	e.SwitchToBuffer(len(e.Buffers) - 1)
	if e.GitManager != nil && !e.GitManager.IsRepo() {
		e.GitManager.RefreshAsync(filepath.Dir(clean), func() {
			e.Screen.PostEvent(tcell.NewEventInterrupt(nil))
		})
	}
	e.SetStatus(fmt.Sprintf("Abierto en pestaña %d: '%s'", len(e.Buffers), filepath.Base(clean)), 2*time.Second)
}

func (e *Editor) getWorkspaceDir() string {
	if e.FileTree != nil && e.FileTree.RootPath != "" && e.FileTree.RootPath != "." {
		return e.FileTree.RootPath
	}
	if e.Buffer != nil && e.Buffer.FilePath != "" {
		dir := filepath.Dir(e.Buffer.FilePath)
		if dir != "" && dir != "." {
			return dir
		}
	}
	if e.FileTree != nil && e.FileTree.RootPath != "" {
		return e.FileTree.RootPath
	}
	cwd, err := os.Getwd()
	if err == nil {
		return cwd
	}
	return "."
}

func (e *Editor) openVirtualBuffer(title, content string) {
	cleanLines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	for i, b := range e.Buffers {
		if b.FilePath == title {
			b.Lines = cleanLines
			b.Modified = false
			e.SwitchToBuffer(i)
			return
		}
	}

	newBuf := &buffer.Buffer{
		FilePath: title,
		Lines:    cleanLines,
		Modified: false,
	}

	e.Buffers = append(e.Buffers, newBuf)
	e.SwitchToBuffer(len(e.Buffers) - 1)
	e.detectCurrentSyntax()
	e.SetStatus(fmt.Sprintf("Abierto %s (Usa Ctrl+W para cerrar)", title), 2*time.Second)
}

// SetStatus muestra un mensaje temporal en la barra inferior.
func (e *Editor) SetStatus(msg string, duration time.Duration) {
	e.StatusMessage = msg
	e.StatusExpires = time.Now().Add(duration)
}

// Run ejecuta el bucle principal de renderizado y captura de eventos.
func (e *Editor) Run() error {
	defer e.Screen.Fini()

	for e.Running {
		e.Draw()
		ev := e.Screen.PollEvent()
		e.HandleEvent(ev)
	}

	return nil
}

// Draw renderiza la interfaz completa en la terminal.
func (e *Editor) Draw() {
	e.Screen.Clear()
	width, height := e.Screen.Size()

	if height < 4 {
		e.Screen.Show()
		return
	}

	sidebarOnRight := strings.ToLower(e.Config.SidebarPosition) == "right"
	sidebarStartX := -1
	sidebarWidth := e.SidebarWidth

	var editorStartX int
	var editorWidth int
	var tabBarStartX int
	var tabBarEndX int

	if e.SidebarOpen {
		if sidebarOnRight {
			sidebarStartX = width - sidebarWidth
			editorStartX = 0
			editorWidth = width - (sidebarWidth + 1)
			tabBarStartX = 0
			tabBarEndX = width - (sidebarWidth + 1)
		} else {
			sidebarStartX = 0
			editorStartX = sidebarWidth + 1
			editorWidth = width - editorStartX
			tabBarStartX = editorStartX
			tabBarEndX = width
		}
		e.drawSidebar(sidebarStartX, sidebarWidth, height)
	} else {
		editorStartX = 0
		editorWidth = width
		tabBarStartX = 0
		tabBarEndX = width
	}

	// 1. Renderizar Barra Superior de Pestañas (Línea 0)
	e.drawTabBar(tabBarStartX, tabBarEndX)

	topOffset := 1
	editorHeight := height - 3 // Línea 0 pestañas, height-2 status, height-1 prompt

	// Ajustar scroll vertical
	if e.Buffer.CursorRow < e.Buffer.ScrollRow {
		e.Buffer.ScrollRow = e.Buffer.CursorRow
	}
	if e.Buffer.CursorRow >= e.Buffer.ScrollRow+editorHeight {
		e.Buffer.ScrollRow = e.Buffer.CursorRow - editorHeight + 1
	}

	gutterWidth := 0
	if e.Config.LineNumbers {
		totalLines := e.Buffer.LineCount()
		gutterWidth = len(strconv.Itoa(totalLines)) + 2
	}

	contentStartX := editorStartX + gutterWidth
	contentWidth := editorWidth - gutterWidth
	contentEndX := editorStartX + editorWidth

	if contentWidth > 0 {
		if e.Buffer.CursorCol < e.Buffer.ScrollCol {
			e.Buffer.ScrollCol = e.Buffer.CursorCol
		}
		if e.Buffer.CursorCol >= e.Buffer.ScrollCol+contentWidth {
			e.Buffer.ScrollCol = e.Buffer.CursorCol - contentWidth + 1
		}
	}

	// Renderizar área del buffer de texto
	for y := 0; y < editorHeight; y++ {
		screenY := topOffset + y
		lineIdx := e.Buffer.ScrollRow + y
		isCursorLine := (lineIdx == e.Buffer.CursorRow)

		lineBgStyle := e.Theme.DefaultStyle
		if isCursorLine && e.Config.HighlightCurrentLine && (!e.SidebarOpen || !e.SidebarFocus) {
			lineBgStyle = e.Theme.CursorLineStyle
		}

		// Rellenar fondo de la línea en el área del editor
		for x := editorStartX; x < contentEndX; x++ {
			e.Screen.SetContent(x, screenY, ' ', nil, lineBgStyle)
		}

		if lineIdx >= e.Buffer.LineCount() {
			if !e.Config.LineNumbers && editorStartX < contentEndX {
				e.Screen.SetContent(editorStartX, screenY, '~', nil, e.Theme.LineNumberStyle)
			}
			continue
		}

		// Dibujar número de línea
		if e.Config.LineNumbers {
			numStr := fmt.Sprintf("%*d ", gutterWidth-1, lineIdx+1)
			numStyle := e.Theme.LineNumberStyle
			if isCursorLine && (!e.SidebarOpen || !e.SidebarFocus) {
				numStyle = e.Theme.LineNumberActive
			}
			for i, ch := range numStr {
				if editorStartX+i < contentEndX {
					e.Screen.SetContent(editorStartX+i, screenY, ch, nil, numStyle)
				}
			}
		}

		// Dibujar contenido de la línea con resaltado de sintaxis y selección
		lineText := e.Buffer.GetLine(lineIdx)
		spans := syntax.HighlightLine(e.CurrentSyntax, lineText)

		var renderedRunes []rune
		var runeStyles []tcell.Style

		lineRunes := []rune(lineText)
		byteOffset := 0

		for runeIdx, r := range lineRunes {
			runeLen := len(string(r))

			st := lineBgStyle
			for _, span := range spans {
				if byteOffset >= span.Start && byteOffset < span.End {
					st = e.Theme.GetSyntaxStyle(span.Scope)
					if isCursorLine && e.Config.HighlightCurrentLine && e.Theme.CursorLineBg != tcell.ColorDefault && (!e.SidebarOpen || !e.SidebarFocus) {
						st = st.Background(e.Theme.CursorLineBg)
					}
					break
				}
			}

			if e.isCellSelected(lineIdx, runeIdx) {
				st = e.Theme.SelectionStyle
			}

			if r == '\t' {
				spaces := e.Config.TabSize
				for s := 0; s < spaces; s++ {
					renderedRunes = append(renderedRunes, ' ')
					runeStyles = append(runeStyles, st)
				}
			} else {
				renderedRunes = append(renderedRunes, r)
				runeStyles = append(runeStyles, st)
			}

			byteOffset += runeLen
		}

		// Dibujar en pantalla respetando scroll horizontal
		screenX := contentStartX
		for colIdx := e.Buffer.ScrollCol; colIdx < len(renderedRunes) && screenX < contentEndX; colIdx++ {
			r := renderedRunes[colIdx]
			st := runeStyles[colIdx]
			e.Screen.SetContent(screenX, screenY, r, nil, st)
			screenX++
		}
	}

	// Renderizar Barra de Estado (Línea height - 2)
	e.drawStatusBar(width, height-2)

	// Renderizar Barra Inferior / Prompt / Paleta (Línea height - 1)
	e.drawBottomBar(width, height-1)

	// Posicionamiento de cursor / modales
	if e.PromptMode == PromptPalette {
		e.drawPaletteModal(width, height)
	} else if e.PromptMode == PromptFileFinder {
		e.drawFileFinderModal(width, height)
	} else if e.PromptMode == PromptSaveAs {
		e.drawSaveAsModal(width, height)
	} else if e.PromptMode != PromptNone {
		e.Screen.ShowCursor(e.PromptCursor, height-1)
	} else if e.SidebarOpen && e.SidebarFocus {
		selY := 1 + (e.FileTree.SelectedIdx - e.FileTree.ScrollY)
		if selY >= 1 && selY < height-2 {
			cursorCol := 1
			if sidebarStartX > 0 {
				cursorCol = sidebarStartX + 1
			}
			e.Screen.ShowCursor(cursorCol, selY)
		} else {
			e.Screen.HideCursor()
		}
	} else {
		cursorX := contentStartX + (e.Buffer.CursorCol - e.Buffer.ScrollCol)
		cursorY := topOffset + (e.Buffer.CursorRow - e.Buffer.ScrollRow)
		if cursorX >= contentStartX && cursorX < contentEndX && cursorY >= topOffset && cursorY < topOffset+editorHeight {
			e.Screen.ShowCursor(cursorX, cursorY)
		} else {
			e.Screen.HideCursor()
		}
	}

	e.Screen.Show()
}

func (e *Editor) drawSidebar(sidebarStartX, treeWidth, height int) {
	bgStyle := e.Theme.StatusBarStyle
	lineFg, _, _ := e.Theme.LineNumberStyle.Decompose()
	sepStyle := e.Theme.StatusBarStyle.Foreground(lineFg)

	// Título en fila 0
	titleStyle := bgStyle.Bold(true)
	if e.SidebarFocus {
		titleStyle = e.Theme.SelectionStyle.Bold(true)
	}
	for x := 0; x < treeWidth; x++ {
		e.Screen.SetContent(sidebarStartX+x, 0, ' ', nil, titleStyle)
	}
	e.drawString(sidebarStartX+1, 0, "EXPLORADOR", titleStyle)

	// Separador vertical
	sepX := treeWidth
	if sidebarStartX > 0 {
		sepX = sidebarStartX - 1
	}
	for y := 0; y < height-1; y++ {
		e.Screen.SetContent(sepX, y, '│', nil, sepStyle)
	}

	maxVisible := height - 3
	visibleRows := e.FileTree.VisibleRows
	selectedIdx := e.FileTree.SelectedIdx

	if selectedIdx < e.FileTree.ScrollY {
		e.FileTree.ScrollY = selectedIdx
	}
	if selectedIdx >= e.FileTree.ScrollY+maxVisible {
		e.FileTree.ScrollY = selectedIdx - maxVisible + 1
	}

	for i := 0; i < maxVisible; i++ {
		rowY := 1 + i
		rowIdx := e.FileTree.ScrollY + i

		for x := 0; x < treeWidth; x++ {
			e.Screen.SetContent(sidebarStartX+x, rowY, ' ', nil, bgStyle)
		}

		if rowIdx >= len(visibleRows) {
			continue
		}

		node := visibleRows[rowIdx]
		isSelected := (rowIdx == selectedIdx)

		itemStyle := bgStyle
		if isSelected {
			if e.SidebarFocus {
				itemStyle = e.Theme.SelectionStyle
			} else {
				itemStyle = e.Theme.CursorLineStyle
			}
		}

		for x := 0; x < treeWidth; x++ {
			e.Screen.SetContent(sidebarStartX+x, rowY, ' ', nil, itemStyle)
		}

		indent := strings.Repeat("  ", node.Depth)
		icon := "  "
		if node.IsDir {
			if node.Expanded {
				icon = "▾ "
			} else {
				icon = "▸ "
			}
		}

		label := indent + icon + node.Name

		gitBadge := ""
		gitBadgeStyle := itemStyle

		textStyle := itemStyle
		if node.IsDir {
			kwFg, _, _ := e.Theme.GetSyntaxStyle("keyword").Decompose()
			textStyle = textStyle.Foreground(kwFg).Bold(true)
		}

		if e.GitManager != nil && e.GitManager.IsRepo() {
			if node.IsDir {
				if e.GitManager.DirHasChanges(node.Path) {
					dotFg, _, _ := e.Theme.GetSyntaxStyle("number").Decompose()
					gitBadgeStyle = itemStyle.Foreground(dotFg)
					gitBadge = "•"
				}
			} else {
				st := e.GitManager.GetFileStatus(node.Path)
				if st != git.StatusNone {
					gitBadge = st.Badge()
					switch st {
					case git.StatusModified:
						yellowFg, _, _ := e.Theme.GetSyntaxStyle("type").Decompose()
						gitBadgeStyle = itemStyle.Foreground(yellowFg).Bold(true)
						if !isSelected {
							textStyle = textStyle.Foreground(yellowFg)
						}
					case git.StatusUntracked, git.StatusAdded:
						greenFg, _, _ := e.Theme.GetSyntaxStyle("string").Decompose()
						gitBadgeStyle = itemStyle.Foreground(greenFg).Bold(true)
						if !isSelected {
							textStyle = textStyle.Foreground(greenFg)
						}
					case git.StatusDeleted:
						redFg, _, _ := e.Theme.GetSyntaxStyle("keyword").Decompose()
						gitBadgeStyle = itemStyle.Foreground(redFg).Bold(true)
						if !isSelected {
							textStyle = textStyle.Foreground(redFg)
						}
					case git.StatusRenamed:
						cyanFg, _, _ := e.Theme.GetSyntaxStyle("function").Decompose()
						gitBadgeStyle = itemStyle.Foreground(cyanFg).Bold(true)
						if !isSelected {
							textStyle = textStyle.Foreground(cyanFg)
						}
					}
				}
			}
		}

		maxLabelWidth := treeWidth - 1
		if gitBadge != "" {
			maxLabelWidth = treeWidth - 3
		}
		if len([]rune(label)) > maxLabelWidth {
			label = string([]rune(label)[:maxLabelWidth-1]) + "…"
		}

		e.drawString(sidebarStartX, rowY, label, textStyle)

		if gitBadge != "" && treeWidth >= 6 {
			badgeX := sidebarStartX + treeWidth - 2
			e.drawString(badgeX, rowY, gitBadge, gitBadgeStyle)
		}
	}
}

func (e *Editor) drawTabBar(startX, endX int) {
	tabBarStyle := e.Theme.StatusBarStyle
	for x := startX; x < endX; x++ {
		e.Screen.SetContent(x, 0, ' ', nil, tabBarStyle)
	}

	currX := startX
	for i, b := range e.Buffers {
		name := b.FilePath
		if name == "" {
			name = "[Nuevo]"
		} else {
			name = filepath.Base(name)
		}

		mod := ""
		if b.Modified {
			mod = "*"
		}

		title := fmt.Sprintf(" %d:%s%s ", i+1, name, mod)

		tabStyle := tabBarStyle
		if i == e.ActiveBufferIdx {
			tabStyle = e.Theme.SelectionStyle.Bold(true)
		}

		for _, ch := range title {
			if currX < endX {
				e.Screen.SetContent(currX, 0, ch, nil, tabStyle)
				currX++
			}
		}

		if currX < endX {
			e.Screen.SetContent(currX, 0, '│', nil, tabBarStyle)
			currX++
		}
	}
}

func (e *Editor) drawStatusBar(width, y int) {
	st := e.Theme.StatusBarStyle
	for x := 0; x < width; x++ {
		e.Screen.SetContent(x, y, ' ', nil, st)
	}

	fileName := e.Buffer.FilePath
	if fileName == "" {
		fileName = "[Sin título]"
	}

	modTag := ""
	if e.Buffer.Modified {
		modTag = " [+]"
	}

	langName := "Texto"
	if e.CurrentSyntax != nil {
		langName = e.CurrentSyntax.Name
	}

	treeStatus := ""
	if e.SidebarOpen {
		if e.SidebarFocus {
			treeStatus = " [Árbol: Foco] "
		} else {
			treeStatus = " [Árbol] "
		}
	}

	gitInfo := ""
	if e.GitManager != nil && e.GitManager.IsRepo() {
		summary := e.GitManager.GetSummary()
		if summary != "" {
			gitInfo = " | " + summary
		}
	}

	left := fmt.Sprintf(" edi | %s%s%s%s (tab %d/%d)", fileName, modTag, gitInfo, treeStatus, e.ActiveBufferIdx+1, len(e.Buffers))
	right := fmt.Sprintf("Ln %d, Col %d | %s | %s ", e.Buffer.CursorRow+1, e.Buffer.CursorCol+1, langName, e.Theme.Name)

	col := 0
	for _, ch := range left {
		if col < width {
			e.Screen.SetContent(col, y, ch, nil, st)
			col++
		}
	}

	rightRunes := []rune(right)
	startX := width - len(rightRunes)
	if startX > len([]rune(left)) {
		for i, ch := range rightRunes {
			e.Screen.SetContent(startX+i, y, ch, nil, st)
		}
	}
}

func (e *Editor) drawBottomBar(width, y int) {
	bgStyle := e.Theme.DefaultStyle

	for x := 0; x < width; x++ {
		e.Screen.SetContent(x, y, ' ', nil, bgStyle)
	}

	switch e.PromptMode {
	case PromptFind:
		prompt := fmt.Sprintf("Buscar: %s", e.PromptText)
		e.drawString(0, y, prompt, e.Theme.DefaultStyle)
		e.PromptCursor = len([]rune("Buscar: ")) + len([]rune(e.PromptText))

	case PromptGoto:
		prompt := fmt.Sprintf("Ir a línea: %s", e.PromptText)
		e.drawString(0, y, prompt, e.Theme.DefaultStyle)
		e.PromptCursor = len([]rune("Ir a línea: ")) + len([]rune(e.PromptText))

	case PromptPalette, PromptFileFinder, PromptSaveAs:
		e.drawString(1, y, "Modal activo (Presiona Esc para cancelar)", e.Theme.LineNumberStyle)

	default:
		if e.StatusMessage != "" && time.Now().Before(e.StatusExpires) {
			e.drawString(1, y, e.StatusMessage, e.Theme.GetSyntaxStyle("function"))
		} else {
			help := "^S/F2: Guardar | ^O: Archivos | ^B: Árbol | ^P: Comandos | ^F: Buscar | ^W: Cerrar | Tab: Pestañas"
			e.drawString(1, y, help, e.Theme.LineNumberStyle)
		}
	}
}

func (e *Editor) drawPaletteModal(width, height int) {
	modalWidth := 66
	if width < 70 {
		modalWidth = width - 4
	}
	if modalWidth < 30 {
		modalWidth = width
	}

	startX := (width - modalWidth) / 2
	startY := 2
	maxItems := 7
	if height < 15 {
		maxItems = 3
	}

	totalItems := len(e.FilteredItems)
	visibleItems := totalItems
	if visibleItems > maxItems {
		visibleItems = maxItems
	}
	if visibleItems == 0 {
		visibleItems = 1
	}

	modalHeight := 4 + visibleItems

	boxStyle := e.Theme.StatusBarStyle
	borderStyle := e.Theme.StatusBarStyle
	inputStyle := e.Theme.DefaultStyle

	for dy := 0; dy < modalHeight; dy++ {
		for dx := 0; dx < modalWidth; dx++ {
			e.Screen.SetContent(startX+dx, startY+dy, ' ', nil, boxStyle)
		}
	}

	e.Screen.SetContent(startX, startY, '╭', nil, borderStyle)
	title := " Paleta de Comandos "
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, startY, '─', nil, borderStyle)
	}
	e.drawString(startX+3, startY, title, e.Theme.GetSyntaxStyle("keyword").Bold(true))
	e.Screen.SetContent(startX+modalWidth-1, startY, '╮', nil, borderStyle)

	e.Screen.SetContent(startX, startY+1, '│', nil, borderStyle)
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, startY+1, ' ', nil, inputStyle)
	}
	e.drawString(startX+2, startY+1, "> ", e.Theme.GetSyntaxStyle("function").Bold(true))
	e.drawString(startX+4, startY+1, e.PromptText, inputStyle)
	e.Screen.SetContent(startX+modalWidth-1, startY+1, '│', nil, borderStyle)

	e.Screen.SetContent(startX, startY+2, '├', nil, borderStyle)
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, startY+2, '─', nil, borderStyle)
	}
	e.Screen.SetContent(startX+modalWidth-1, startY+2, '┤', nil, borderStyle)

	if e.PaletteIndex < e.PaletteScroll {
		e.PaletteScroll = e.PaletteIndex
	}
	if e.PaletteIndex >= e.PaletteScroll+maxItems {
		e.PaletteScroll = e.PaletteIndex - maxItems + 1
	}

	itemRow := startY + 3
	if totalItems == 0 {
		e.Screen.SetContent(startX, itemRow, '│', nil, borderStyle)
		e.drawString(startX+4, itemRow, "(No hay comandos coincidentes)", e.Theme.LineNumberStyle)
		e.Screen.SetContent(startX+modalWidth-1, itemRow, '│', nil, borderStyle)
	} else {
		for i := 0; i < visibleItems; i++ {
			actualIdx := e.PaletteScroll + i
			if actualIdx >= totalItems {
				break
			}
			item := e.FilteredItems[actualIdx]
			isSelected := (actualIdx == e.PaletteIndex)

			itemStyle := boxStyle
			if isSelected {
				itemStyle = e.Theme.SelectionStyle
			}

			for dx := 1; dx < modalWidth-1; dx++ {
				e.Screen.SetContent(startX+dx, itemRow+i, ' ', nil, itemStyle)
			}

			e.Screen.SetContent(startX, itemRow+i, '│', nil, borderStyle)

			prefix := "  "
			if isSelected {
				prefix = "▶ "
			}
			e.drawString(startX+2, itemRow+i, prefix+item.Title, itemStyle)

			if item.Description != "" {
				descLen := len([]rune(item.Description))
				descX := startX + modalWidth - 2 - descLen
				if descX > startX+len([]rune(item.Title))+6 {
					e.drawString(descX, itemRow+i, item.Description, itemStyle)
				}
			}

			e.Screen.SetContent(startX+modalWidth-1, itemRow+i, '│', nil, borderStyle)
		}
	}

	bottomY := startY + modalHeight - 1
	e.Screen.SetContent(startX, bottomY, '╰', nil, borderStyle)
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, bottomY, '─', nil, borderStyle)
	}
	footerText := " [↑/↓ navegar | ↵ ejecutar | Esc cancelar] "
	if modalWidth > len([]rune(footerText))+6 {
		footerX := startX + (modalWidth-len([]rune(footerText)))/2
		e.drawString(footerX, bottomY, footerText, e.Theme.LineNumberStyle)
	}
	e.Screen.SetContent(startX+modalWidth-1, bottomY, '╯', nil, borderStyle)

	e.PromptCursor = startX + 4 + len([]rune(e.PromptText))
	e.Screen.ShowCursor(e.PromptCursor, startY+1)
}

func (e *Editor) drawFileFinderModal(width, height int) {
	modalWidth := 72
	if width < 76 {
		modalWidth = width - 4
	}
	if modalWidth < 30 {
		modalWidth = width
	}

	startX := (width - modalWidth) / 2
	startY := 2
	maxItems := 8
	if height < 16 {
		maxItems = 4
	}

	totalItems := len(e.FilteredFiles)
	visibleItems := totalItems
	if visibleItems > maxItems {
		visibleItems = maxItems
	}
	if visibleItems == 0 {
		visibleItems = 1
	}

	modalHeight := 4 + visibleItems

	boxStyle := e.Theme.StatusBarStyle
	borderStyle := e.Theme.StatusBarStyle
	inputStyle := e.Theme.DefaultStyle

	for dy := 0; dy < modalHeight; dy++ {
		for dx := 0; dx < modalWidth; dx++ {
			e.Screen.SetContent(startX+dx, startY+dy, ' ', nil, boxStyle)
		}
	}

	e.Screen.SetContent(startX, startY, '╭', nil, borderStyle)
	title := fmt.Sprintf(" Abrir Archivo (%d en espacio de trabajo) ", len(e.WorkspaceFiles))
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, startY, '─', nil, borderStyle)
	}
	e.drawString(startX+3, startY, title, e.Theme.GetSyntaxStyle("function").Bold(true))
	e.Screen.SetContent(startX+modalWidth-1, startY, '╮', nil, borderStyle)

	e.Screen.SetContent(startX, startY+1, '│', nil, borderStyle)
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, startY+1, ' ', nil, inputStyle)
	}
	e.drawString(startX+2, startY+1, "> ", e.Theme.GetSyntaxStyle("keyword").Bold(true))
	e.drawString(startX+4, startY+1, e.PromptText, inputStyle)
	e.Screen.SetContent(startX+modalWidth-1, startY+1, '│', nil, borderStyle)

	e.Screen.SetContent(startX, startY+2, '├', nil, borderStyle)
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, startY+2, '─', nil, borderStyle)
	}
	e.Screen.SetContent(startX+modalWidth-1, startY+2, '┤', nil, borderStyle)

	if e.FileFinderIdx < e.FileFinderScroll {
		e.FileFinderScroll = e.FileFinderIdx
	}
	if e.FileFinderIdx >= e.FileFinderScroll+maxItems {
		e.FileFinderScroll = e.FileFinderIdx - maxItems + 1
	}

	itemRow := startY + 3
	if totalItems == 0 {
		e.Screen.SetContent(startX, itemRow, '│', nil, borderStyle)
		e.drawString(startX+4, itemRow, "(No se encontraron archivos coincidentes)", e.Theme.LineNumberStyle)
		e.Screen.SetContent(startX+modalWidth-1, itemRow, '│', nil, borderStyle)
	} else {
		for i := 0; i < visibleItems; i++ {
			actualIdx := e.FileFinderScroll + i
			if actualIdx >= totalItems {
				break
			}
			filePath := e.FilteredFiles[actualIdx]
			isSelected := (actualIdx == e.FileFinderIdx)

			itemStyle := boxStyle
			if isSelected {
				itemStyle = e.Theme.SelectionStyle
			}

			for dx := 1; dx < modalWidth-1; dx++ {
				e.Screen.SetContent(startX+dx, itemRow+i, ' ', nil, itemStyle)
			}

			e.Screen.SetContent(startX, itemRow+i, '│', nil, borderStyle)

			prefix := "  "
			if isSelected {
				prefix = "▶ "
			}

			// Comprobar si el archivo ya está abierto en alguna pestaña
			openTag := ""
			for bIdx, b := range e.Buffers {
				if b.FilePath != "" && (b.FilePath == filePath || filepath.Clean(b.FilePath) == filePath || strings.HasSuffix(b.FilePath, filePath) || filepath.Base(b.FilePath) == filePath) {
					openTag = fmt.Sprintf("[tab %d]", bIdx+1)
					break
				}
			}

			maxLabelLen := modalWidth - 6
			if openTag != "" {
				maxLabelLen -= len(openTag) + 1
			}
			dispPath := filePath
			if len([]rune(dispPath)) > maxLabelLen && maxLabelLen > 5 {
				dispPath = "…" + string([]rune(dispPath)[len([]rune(dispPath))-(maxLabelLen-1):])
			}

			e.drawString(startX+2, itemRow+i, prefix+dispPath, itemStyle)

			if openTag != "" && modalWidth > 20 {
				tagStyle := e.Theme.GetSyntaxStyle("comment")
				if isSelected {
					tagStyle = itemStyle.Bold(true)
				}
				e.drawString(startX+modalWidth-len(openTag)-2, itemRow+i, openTag, tagStyle)
			}

			e.Screen.SetContent(startX+modalWidth-1, itemRow+i, '│', nil, borderStyle)
		}
	}

	bottomY := startY + modalHeight - 1
	e.Screen.SetContent(startX, bottomY, '╰', nil, borderStyle)
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, bottomY, '─', nil, borderStyle)
	}
	footerText := " [↑/↓ navegar | ↵ abrir | Esc cancelar] "
	if modalWidth > len([]rune(footerText))+6 {
		footerX := startX + (modalWidth-len([]rune(footerText)))/2
		e.drawString(footerX, bottomY, footerText, e.Theme.LineNumberStyle)
	}
	e.Screen.SetContent(startX+modalWidth-1, bottomY, '╯', nil, borderStyle)

	e.PromptCursor = startX + 4 + len([]rune(e.PromptText))
	e.Screen.ShowCursor(e.PromptCursor, startY+1)
}

func (e *Editor) drawSaveAsModal(width, height int) {
	modalWidth := 62
	if width < 66 {
		modalWidth = width - 4
	}
	if modalWidth < 30 {
		modalWidth = width
	}

	startX := (width - modalWidth) / 2
	startY := height / 3
	if startY < 2 {
		startY = 2
	}

	modalHeight := 6

	boxStyle := e.Theme.StatusBarStyle
	borderStyle := e.Theme.StatusBarStyle
	inputStyle := e.Theme.DefaultStyle

	for dy := 0; dy < modalHeight; dy++ {
		for dx := 0; dx < modalWidth; dx++ {
			e.Screen.SetContent(startX+dx, startY+dy, ' ', nil, boxStyle)
		}
	}

	// Borde superior
	e.Screen.SetContent(startX, startY, '╭', nil, borderStyle)
	title := " Guardar Archivo "
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, startY, '─', nil, borderStyle)
	}
	e.drawString(startX+3, startY, title, e.Theme.GetSyntaxStyle("function").Bold(true))
	e.Screen.SetContent(startX+modalWidth-1, startY, '╮', nil, borderStyle)

	// Fila 1: Indicación
	e.Screen.SetContent(startX, startY+1, '│', nil, borderStyle)
	label := " Ingrese el nombre o ruta del archivo:"
	if len([]rune(label)) > modalWidth-2 {
		label = string([]rune(label)[:modalWidth-3])
	}
	e.drawString(startX+2, startY+1, label, e.Theme.LineNumberStyle)
	e.Screen.SetContent(startX+modalWidth-1, startY+1, '│', nil, borderStyle)

	// Fila 2: Campo de entrada interactivo
	e.Screen.SetContent(startX, startY+2, '│', nil, borderStyle)
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, startY+2, ' ', nil, inputStyle)
	}
	e.drawString(startX+2, startY+2, "> ", e.Theme.GetSyntaxStyle("keyword").Bold(true))
	e.drawString(startX+4, startY+2, e.PromptText, inputStyle)
	e.Screen.SetContent(startX+modalWidth-1, startY+2, '│', nil, borderStyle)

	// Fila 3: Separador
	e.Screen.SetContent(startX, startY+3, '├', nil, borderStyle)
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, startY+3, '─', nil, borderStyle)
	}
	e.Screen.SetContent(startX+modalWidth-1, startY+3, '┤', nil, borderStyle)

	// Fila 4: Directorio actual de guardado
	e.Screen.SetContent(startX, startY+4, '│', nil, borderStyle)
	cwd, _ := os.Getwd()
	folderInfo := fmt.Sprintf(" Guardar en: ./%s", filepath.Base(cwd))
	if len([]rune(folderInfo)) > modalWidth-2 {
		folderInfo = string([]rune(folderInfo)[:modalWidth-3])
	}
	e.drawString(startX+2, startY+4, folderInfo, e.Theme.LineNumberStyle)
	e.Screen.SetContent(startX+modalWidth-1, startY+4, '│', nil, borderStyle)

	// Fila 5: Borde inferior con atajos
	bottomY := startY + 5
	e.Screen.SetContent(startX, bottomY, '╰', nil, borderStyle)
	for dx := 1; dx < modalWidth-1; dx++ {
		e.Screen.SetContent(startX+dx, bottomY, '─', nil, borderStyle)
	}
	footerText := " [↵ Guardar automáticamente | Esc Cancelar] "
	if modalWidth > len([]rune(footerText))+6 {
		footerX := startX + (modalWidth-len([]rune(footerText)))/2
		e.drawString(footerX, bottomY, footerText, e.Theme.LineNumberStyle)
	}
	e.Screen.SetContent(startX+modalWidth-1, bottomY, '╯', nil, borderStyle)

	// Posicionar cursor en el cuadro de texto
	e.PromptCursor = startX + 4 + len([]rune(e.PromptText))
	e.Screen.ShowCursor(e.PromptCursor, startY+2)
}

func (e *Editor) drawString(x, y int, str string, st tcell.Style) {
	col := 0
	for _, ch := range str {
		e.Screen.SetContent(x+col, y, ch, nil, st)
		col++
	}
}

// HandleEvent procesa teclado, redimensionado y mouse.
func (e *Editor) HandleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		e.Screen.Sync()

	case *tcell.EventInterrupt:
		// Refresco asíncrono completado (Git, plugins)

	case *tcell.EventMouse:
		if ev.Buttons()&tcell.ButtonPrimary != 0 {
			mx, my := ev.Position()
			e.handleMouseClick(mx, my)
		}

	case *tcell.EventKey:
		if e.PromptMode != PromptNone {
			e.handlePromptKey(ev)
		} else if e.SidebarOpen && e.SidebarFocus {
			e.handleSidebarKey(ev)
		} else {
			e.handleNormalKey(ev)
		}
	}
}

func (e *Editor) handleMouseClick(mx, my int) {
	if e.PromptMode != PromptNone {
		return
	}

	width, height := e.Screen.Size()
	sidebarOnRight := strings.ToLower(e.Config.SidebarPosition) == "right"
	sidebarStartX := 0
	if sidebarOnRight {
		sidebarStartX = width - e.SidebarWidth
	}

	// 1. Clic en barra de pestañas (fila 0)
	if my == 0 {
		if e.SidebarOpen {
			if sidebarOnRight && mx >= sidebarStartX {
				e.SidebarFocus = true
				return
			} else if !sidebarOnRight && mx < e.SidebarWidth {
				e.SidebarFocus = true
				return
			}
		}

		tabBarStartX := 0
		tabBarEndX := width
		if e.SidebarOpen {
			if sidebarOnRight {
				tabBarEndX = width - (e.SidebarWidth + 1)
			} else {
				tabBarStartX = e.SidebarWidth + 1
			}
		}

		currX := tabBarStartX
		for i, b := range e.Buffers {
			name := b.FilePath
			if name == "" {
				name = "[Nuevo]"
			} else {
				name = filepath.Base(name)
			}
			mod := ""
			if b.Modified {
				mod = "*"
			}
			tabTitle := fmt.Sprintf(" %d:%s%s ", i+1, name, mod)
			tabLen := len([]rune(tabTitle))

			if mx >= currX && mx < currX+tabLen && currX+tabLen <= tabBarEndX {
				e.SwitchToBuffer(i)
				return
			}
			currX += tabLen + 1
		}
		return
	}

	// 2. Clic en el panel lateral del explorador
	if e.SidebarOpen {
		inSidebar := false
		if sidebarOnRight && mx >= sidebarStartX {
			inSidebar = true
		} else if !sidebarOnRight && mx < e.SidebarWidth {
			inSidebar = true
		}

		if inSidebar {
			e.SidebarFocus = true
			rowIdx := e.FileTree.ScrollY + (my - 1)
			if rowIdx >= 0 && rowIdx < len(e.FileTree.VisibleRows) {
				e.FileTree.SelectedIdx = rowIdx
				node := e.FileTree.VisibleRows[rowIdx]
				if node.IsDir {
					e.FileTree.ToggleCurrent()
				} else {
					e.OpenFile(node.Path)
					e.SidebarFocus = false
				}
			}
			return
		}
	}

	// 3. Clic en el área de edición del texto
	if my >= 1 && my < height-2 {
		e.SidebarFocus = false
		editorStartX := 0
		if e.SidebarOpen && !sidebarOnRight {
			editorStartX = e.SidebarWidth + 1
		}
		gutterWidth := 0
		if e.Config.LineNumbers {
			gutterWidth = len(strconv.Itoa(e.Buffer.LineCount())) + 2
		}
		contentStartX := editorStartX + gutterWidth

		targetRow := e.Buffer.ScrollRow + (my - 1)
		targetCol := e.Buffer.ScrollCol + (mx - contentStartX)

		if targetRow >= 0 && targetRow < e.Buffer.LineCount() {
			e.Buffer.CursorRow = targetRow
			lineLen := len([]rune(e.Buffer.Lines[targetRow]))
			if targetCol < 0 {
				e.Buffer.CursorCol = 0
			} else if targetCol > lineLen {
				e.Buffer.CursorCol = lineLen
			} else {
				e.Buffer.CursorCol = targetCol
			}
			e.Buffer.ClampCursor()
			e.Selecting = false
		}
	}
}

func (e *Editor) handleSidebarKey(ev *tcell.EventKey) {
	chord := keyToString(ev)
	if chord == "Ctrl+B" || chord == "Ctrl+E" {
		e.SidebarFocus = false
		return
	}
	if action, ok := e.Keybindings[chord]; ok {
		e.executeAction(action)
		return
	}

	switch ev.Key() {
	case tcell.KeyUp:
		e.FileTree.MoveUp()
	case tcell.KeyDown:
		e.FileTree.MoveDown()
	case tcell.KeyRight:
		e.FileTree.ExpandCurrent()
	case tcell.KeyLeft:
		e.FileTree.CollapseCurrent()
	case tcell.KeyEnter:
		node := e.FileTree.GetSelected()
		if node != nil {
			if node.IsDir {
				e.FileTree.ToggleCurrent()
			} else {
				e.OpenFile(node.Path)
				e.SidebarFocus = false
			}
		}
	case tcell.KeyTab, tcell.KeyEscape:
		e.SidebarFocus = false
	case tcell.KeyRune:
		if ev.Rune() == 'a' || ev.Rune() == 'n' {
			node := e.FileTree.GetSelected()
			dirPath := e.FileTree.RootPath
			if node != nil {
				if node.IsDir {
					dirPath = node.Path
				} else {
					dirPath = filepath.Dir(node.Path)
				}
			}
			relDir, err := filepath.Rel(e.FileTree.RootPath, dirPath)
			if err != nil || relDir == "." {
				relDir = ""
			} else {
				relDir = relDir + string(filepath.Separator)
			}
			newBuf := buffer.NewEmptyBuffer()
			e.Buffers = append(e.Buffers, newBuf)
			e.SwitchToBuffer(len(e.Buffers) - 1)
			e.SidebarFocus = false
			e.PromptMode = PromptSaveAs
			e.PromptText = relDir
			return
		}
	}
}

func (e *Editor) handleNormalKey(ev *tcell.EventKey) {
	chord := keyToString(ev)
	if action, ok := e.Keybindings[chord]; ok {
		e.executeAction(action)
		return
	}

	// Manejo de Selección con Shift + Teclas de navegación
	if ev.Modifiers()&tcell.ModShift != 0 {
		if !e.Selecting {
			e.Selecting = true
			e.SelectAnchorRow = e.Buffer.CursorRow
			e.SelectAnchorCol = e.Buffer.CursorCol
		}
	} else if ev.Key() == tcell.KeyUp || ev.Key() == tcell.KeyDown || ev.Key() == tcell.KeyLeft || ev.Key() == tcell.KeyRight || ev.Key() == tcell.KeyHome || ev.Key() == tcell.KeyEnd {
		e.Selecting = false
	}

	switch ev.Key() {
	case tcell.KeyUp:
		e.Buffer.CursorRow--
		e.Buffer.ClampCursor()
	case tcell.KeyDown:
		e.Buffer.CursorRow++
		e.Buffer.ClampCursor()
	case tcell.KeyLeft:
		if e.Buffer.CursorCol > 0 {
			e.Buffer.CursorCol--
		} else if e.Buffer.CursorRow > 0 {
			e.Buffer.CursorRow--
			e.Buffer.CursorCol = len([]rune(e.Buffer.Lines[e.Buffer.CursorRow]))
		}
		e.Buffer.ClampCursor()
	case tcell.KeyRight:
		lineRunes := []rune(e.Buffer.Lines[e.Buffer.CursorRow])
		if e.Buffer.CursorCol < len(lineRunes) {
			e.Buffer.CursorCol++
		} else if e.Buffer.CursorRow < e.Buffer.LineCount()-1 {
			e.Buffer.CursorRow++
			e.Buffer.CursorCol = 0
		}
		e.Buffer.ClampCursor()
	case tcell.KeyHome:
		e.Buffer.CursorCol = 0
	case tcell.KeyEnd:
		e.Buffer.CursorCol = len([]rune(e.Buffer.Lines[e.Buffer.CursorRow]))
	case tcell.KeyPgUp:
		_, height := e.Screen.Size()
		e.Buffer.CursorRow -= height - 3
		e.Buffer.ClampCursor()
	case tcell.KeyPgDn:
		_, height := e.Screen.Size()
		e.Buffer.CursorRow += height - 3
		e.Buffer.ClampCursor()
	case tcell.KeyEnter:
		if e.Selecting {
			e.deleteSelection()
		}
		e.Buffer.SplitLine(e.Config.AutoIndent)
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if e.Selecting {
			e.deleteSelection()
		} else {
			e.Buffer.DeleteRune()
		}
	case tcell.KeyDelete:
		if e.Selecting {
			e.deleteSelection()
		} else {
			e.Buffer.DeleteForwardRune()
		}
	case tcell.KeyTab:
		if e.Config.ExpandTabs {
			for i := 0; i < e.Config.TabSize; i++ {
				e.Buffer.InsertRune(' ')
			}
		} else {
			e.Buffer.InsertRune('\t')
		}
	case tcell.KeyRune:
		if e.Selecting {
			e.deleteSelection()
		}
		e.Buffer.InsertRune(ev.Rune())
		e.QuitWarned = false
	}
}

func (e *Editor) handlePromptKey(ev *tcell.EventKey) {
	if ev.Key() == tcell.KeyCtrlV || (ev.Modifiers()&tcell.ModCtrl != 0 && (ev.Rune() == 'v' || ev.Rune() == 'V')) {
		if e.Clipboard != "" {
			e.PromptText += e.Clipboard
			if e.PromptMode == PromptPalette {
				e.updatePaletteFilter()
			} else if e.PromptMode == PromptFileFinder {
				e.FilteredFiles = FilterFiles(e.WorkspaceFiles, e.PromptText)
				e.FileFinderIdx = 0
				e.FileFinderScroll = 0
			}
			return
		}
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		wasSaveAs := (e.PromptMode == PromptSaveAs)
		e.PromptMode = PromptNone
		e.PromptText = ""
		if wasSaveAs {
			e.SetStatus("Guardado cancelado", 1*time.Second)
		}

	case tcell.KeyEnter:
		e.submitPrompt()

	case tcell.KeyUp:
		if e.PromptMode == PromptPalette && len(e.FilteredItems) > 0 {
			e.PaletteIndex = (e.PaletteIndex - 1 + len(e.FilteredItems)) % len(e.FilteredItems)
		} else if e.PromptMode == PromptFileFinder && len(e.FilteredFiles) > 0 {
			e.FileFinderIdx = (e.FileFinderIdx - 1 + len(e.FilteredFiles)) % len(e.FilteredFiles)
		}

	case tcell.KeyDown:
		if e.PromptMode == PromptPalette && len(e.FilteredItems) > 0 {
			e.PaletteIndex = (e.PaletteIndex + 1) % len(e.FilteredItems)
		} else if e.PromptMode == PromptFileFinder && len(e.FilteredFiles) > 0 {
			e.FileFinderIdx = (e.FileFinderIdx + 1) % len(e.FilteredFiles)
		}

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		runes := []rune(e.PromptText)
		if len(runes) > 0 {
			e.PromptText = string(runes[:len(runes)-1])
			if e.PromptMode == PromptPalette {
				e.updatePaletteFilter()
			} else if e.PromptMode == PromptFileFinder {
				e.FilteredFiles = FilterFiles(e.WorkspaceFiles, e.PromptText)
				e.FileFinderIdx = 0
				e.FileFinderScroll = 0
			}
		}

	case tcell.KeyRune:
		e.PromptText += string(ev.Rune())
		if e.PromptMode == PromptPalette {
			e.updatePaletteFilter()
		} else if e.PromptMode == PromptFileFinder {
			e.FilteredFiles = FilterFiles(e.WorkspaceFiles, e.PromptText)
			e.FileFinderIdx = 0
			e.FileFinderScroll = 0
		}
	}
}

func (e *Editor) updatePaletteFilter() {
	query := strings.ToLower(e.PromptText)
	var filtered []PaletteItem
	for _, item := range e.PaletteItems {
		if query == "" || strings.Contains(strings.ToLower(item.Title), query) || strings.Contains(strings.ToLower(item.Description), query) {
			filtered = append(filtered, item)
		}
	}
	e.FilteredItems = filtered
	e.PaletteIndex = 0
	e.PaletteScroll = 0
}

func (e *Editor) submitPrompt() {
	switch e.PromptMode {
	case PromptFind:
		term := strings.TrimSpace(e.PromptText)
		e.LastSearchTerm = term
		e.PromptMode = PromptNone
		if term != "" {
			e.findNext(term)
		}

	case PromptGoto:
		lineNum, err := strconv.Atoi(strings.TrimSpace(e.PromptText))
		e.PromptMode = PromptNone
		if err == nil && lineNum > 0 {
			e.Buffer.CursorRow = lineNum - 1
			e.Buffer.CursorCol = 0
			e.Buffer.ClampCursor()
		} else {
			e.SetStatus("Línea inválida", 2*time.Second)
		}

	case PromptSaveAs:
		newPath := strings.TrimSpace(e.PromptText)
		e.PromptMode = PromptNone
		if newPath != "" {
			e.Buffer.FilePath = newPath
			e.detectCurrentSyntax()
			e.saveFile()
		} else {
			e.SetStatus("Guardado cancelado: nombre de archivo vacío", 2*time.Second)
		}

	case PromptPalette:
		e.PromptMode = PromptNone
		if len(e.FilteredItems) > 0 && e.PaletteIndex < len(e.FilteredItems) {
			selected := e.FilteredItems[e.PaletteIndex]
			e.executeAction(selected.Action)
		}

	case PromptFileFinder:
		e.PromptMode = PromptNone
		var target string
		if len(e.FilteredFiles) > 0 && e.FileFinderIdx < len(e.FilteredFiles) {
			target = e.FilteredFiles[e.FileFinderIdx]
		} else if strings.TrimSpace(e.PromptText) != "" {
			target = strings.TrimSpace(e.PromptText)
		}
		if target != "" {
			if !filepath.IsAbs(target) {
				root := e.getWorkspaceDir()
				if root != "" && root != "." {
					target = filepath.Join(root, target)
				}
			}
			e.OpenFile(target)
		}
	}
}

func (e *Editor) executeAction(action string) {
	switch action {
	case "help:docs":
		e.OpenHelp()

	case "view:toggle_tree":
		if !e.SidebarOpen {
			e.SidebarOpen = true
			e.SidebarFocus = true
		} else if !e.SidebarFocus {
			// Ya está abierto, conmutar foco al explorador
			e.SidebarFocus = true
		} else {
			// Ya está enfocado, cerrarlo
			e.SidebarOpen = false
			e.SidebarFocus = false
		}

	case "view:toggle_sidebar_pos":
		if strings.ToLower(e.Config.SidebarPosition) == "right" {
			e.Config.SidebarPosition = "left"
			e.SetStatus("Posición del explorador: Izquierda", 2*time.Second)
		} else {
			e.Config.SidebarPosition = "right"
			e.SetStatus("Posición del explorador: Derecha", 2*time.Second)
		}

	case "tree:focus":
		if e.SidebarOpen {
			e.SidebarFocus = !e.SidebarFocus
		} else {
			e.SidebarOpen = true
			e.SidebarFocus = true
		}

	case "file:find":
		root := e.getWorkspaceDir()
		e.WorkspaceFiles = ScanWorkspaceFilesDir(root, 3000)
		e.FilteredFiles = FilterFiles(e.WorkspaceFiles, "")
		e.FileFinderIdx = 0
		e.FileFinderScroll = 0
		e.PromptMode = PromptFileFinder
		e.PromptText = ""

	case "buffer:new":
		newBuf := buffer.NewEmptyBuffer()
		e.Buffers = append(e.Buffers, newBuf)
		e.SwitchToBuffer(len(e.Buffers) - 1)
		e.SetStatus("Nueva pestaña creada", 1*time.Second)

	case "buffer:close":
		e.CloseActiveBuffer()

	case "buffer:next":
		if len(e.Buffers) > 1 {
			e.SwitchToBuffer((e.ActiveBufferIdx + 1) % len(e.Buffers))
		}

	case "buffer:prev":
		if len(e.Buffers) > 1 {
			e.SwitchToBuffer((e.ActiveBufferIdx - 1 + len(e.Buffers)) % len(e.Buffers))
		}

	case "buffer:goto_1":
		e.SwitchToBuffer(0)
	case "buffer:goto_2":
		e.SwitchToBuffer(1)
	case "buffer:goto_3":
		e.SwitchToBuffer(2)
	case "buffer:goto_4":
		e.SwitchToBuffer(3)
	case "buffer:goto_5":
		e.SwitchToBuffer(4)
	case "buffer:goto_6":
		e.SwitchToBuffer(5)
	case "buffer:goto_7":
		e.SwitchToBuffer(6)
	case "buffer:goto_8":
		e.SwitchToBuffer(7)
	case "buffer:goto_9":
		e.SwitchToBuffer(8)

	case "selection:copy":
		text := e.getSelectedText()
		if text == "" && e.Buffer.CursorRow < e.Buffer.LineCount() {
			text = e.Buffer.Lines[e.Buffer.CursorRow]
		}
		e.Clipboard = text
		e.SetStatus("Copiado al portapapeles", 1*time.Second)

	case "selection:cut":
		text := e.getSelectedText()
		if text != "" {
			e.Clipboard = text
			e.deleteSelection()
			e.SetStatus("Cortado al portapapeles", 1*time.Second)
		} else if e.Buffer.CursorRow < e.Buffer.LineCount() {
			e.Clipboard = e.Buffer.Lines[e.Buffer.CursorRow]
			e.Buffer.DeleteLine()
			e.SetStatus("Línea cortada", 1*time.Second)
		}

	case "selection:paste":
		e.pasteText(e.Clipboard)
		e.SetStatus("Pegado desde portapapeles", 1*time.Second)

	case "buffer:save":
		e.saveFile()

	case "buffer:save_as":
		e.PromptMode = PromptSaveAs
		e.PromptText = e.Buffer.FilePath

	case "app:quit":
		hasModified := false
		for _, b := range e.Buffers {
			if b.Modified {
				hasModified = true
				break
			}
		}
		if hasModified && !e.QuitWarned {
			e.QuitWarned = true
			e.SetStatus("¡Pestañas con cambios sin guardar! Presiona Ctrl+Q de nuevo para salir", 3*time.Second)
			return
		}
		e.Running = false

	case "buffer:undo":
		if e.Buffer.Undo() {
			e.SetStatus("Deshecho", 1*time.Second)
		}
	case "buffer:redo":
		if e.Buffer.Redo() {
			e.SetStatus("Rehecho", 1*time.Second)
		}
	case "buffer:find":
		e.PromptMode = PromptFind
		e.PromptText = e.LastSearchTerm
	case "buffer:goto_line":
		e.PromptMode = PromptGoto
		e.PromptText = ""
	case "palette:open":
		e.PaletteItems = buildDefaultPalette()
		if len(e.Buffers) > 1 {
			var tabItems []PaletteItem
			for idx, b := range e.Buffers {
				name := filepath.Base(b.FilePath)
				if name == "" || name == "." {
					name = fmt.Sprintf("Sin título %d", idx+1)
				}
				desc := fmt.Sprintf("Ir a pestaña %d", idx+1)
				if idx == e.ActiveBufferIdx {
					desc += " (Activa)"
				}
				tabItems = append(tabItems, PaletteItem{
					Title:       fmt.Sprintf("Pestaña: %s", name),
					Action:      fmt.Sprintf("buffer:goto_idx:%d", idx),
					Description: desc,
				})
			}
			e.PaletteItems = append(tabItems, e.PaletteItems...)
		}
		e.PromptMode = PromptPalette
		e.PromptText = ""
		e.updatePaletteFilter()
	case "line:duplicate":
		e.Buffer.DuplicateLine()
	case "line:delete":
		e.Buffer.DeleteLine()
	case "line:move_up":
		if e.Selecting {
			startRow, endRow := e.SelectAnchorRow, e.Buffer.CursorRow
			if startRow > endRow {
				startRow, endRow = endRow, startRow
			}
			if e.Buffer.MoveLinesUp(startRow, endRow) {
				e.SelectAnchorRow--
				e.Buffer.CursorRow--
				e.Buffer.ClampCursor()
			}
		} else {
			e.Buffer.MoveLineUp()
		}
	case "line:move_down":
		if e.Selecting {
			startRow, endRow := e.SelectAnchorRow, e.Buffer.CursorRow
			if startRow > endRow {
				startRow, endRow = endRow, startRow
			}
			if e.Buffer.MoveLinesDown(startRow, endRow) {
				e.SelectAnchorRow++
				e.Buffer.CursorRow++
				e.Buffer.ClampCursor()
			}
		} else {
			e.Buffer.MoveLineDown()
		}
	case "theme:dracula":
		if th, err := theme.LoadTheme("dracula"); err == nil {
			e.Theme = th
			e.SetStatus("Tema cambiado a Dracula", 2*time.Second)
		}
	case "theme:monokai":
		if th, err := theme.LoadTheme("monokai"); err == nil {
			e.Theme = th
			e.SetStatus("Tema cambiado a Monokai", 2*time.Second)
		}
	case "toggle:line_numbers":
		e.Config.LineNumbers = !e.Config.LineNumbers
	case "toggle:current_line":
		e.Config.HighlightCurrentLine = !e.Config.HighlightCurrentLine
	case "git:diff_file":
		if e.GitManager != nil && e.GitManager.IsRepo() {
			if e.Buffer.FilePath == "" {
				e.SetStatus("El archivo actual no tiene ruta guardada", 2*time.Second)
				return
			}
			diff, err := e.GitManager.GetFileDiff(e.Buffer.FilePath)
			if err != nil || strings.TrimSpace(diff) == "" {
				e.SetStatus("Sin cambios con respecto a HEAD", 2*time.Second)
				return
			}
			e.openVirtualBuffer(fmt.Sprintf("%s.diff", filepath.Base(e.Buffer.FilePath)), diff)
		} else {
			e.SetStatus("No se detectó repositorio Git activo", 2*time.Second)
		}

	case "git:diff_project":
		if e.GitManager != nil && e.GitManager.IsRepo() {
			diff, err := e.GitManager.GetProjectDiff()
			if err != nil || strings.TrimSpace(diff) == "" {
				e.SetStatus("El repositorio no tiene cambios con respecto a HEAD", 2*time.Second)
				return
			}
			e.openVirtualBuffer("proyecto.diff", diff)
		} else {
			e.SetStatus("No se detectó repositorio Git activo", 2*time.Second)
		}

	case "git:status_files":
		if e.GitManager != nil && e.GitManager.IsRepo() {
			snap := e.GitManager.GetStatusSnapshot()
			if len(snap.Files) == 0 {
				e.SetStatus("Repositorio limpio (0 cambios)", 2*time.Second)
				return
			}
			var changed []string
			for fPath := range snap.Files {
				rel, err := filepath.Rel(snap.RootDir, fPath)
				if err == nil {
					changed = append(changed, rel)
				} else {
					changed = append(changed, fPath)
				}
			}
			e.WorkspaceFiles = changed
			e.FilteredFiles = changed
			e.FileFinderIdx = 0
			e.FileFinderScroll = 0
			e.PromptMode = PromptFileFinder
			e.PromptText = ""
		} else {
			e.SetStatus("No se detectó repositorio Git activo", 2*time.Second)
		}

	case "git:refresh":
		if e.GitManager != nil && e.GitManager.IsRepo() {
			e.GitManager.RefreshSync(e.getWorkspaceDir())
			e.SetStatus(fmt.Sprintf("Git actualizado: %s", e.GitManager.GetSummary()), 2*time.Second)
		} else {
			e.SetStatus("No se detectó repositorio Git activo", 2*time.Second)
		}

	case "app:about":
		e.SetStatus(fmt.Sprintf("%s | github.com/%s", version.FullVersion(), updater.DefaultRepo), 4*time.Second)

	case "app:check_update":
		e.SetStatus("Verificando actualizaciones en GitHub...", 3*time.Second)
		go func() {
			rel, err := updater.CheckLatestRelease(updater.DefaultRepo)
			if err != nil {
				e.SetStatus(fmt.Sprintf("Actualización: %v", err), 4*time.Second)
			} else if updater.IsNewer(rel.TagName, version.Version) {
				e.SetStatus(fmt.Sprintf("¡Nueva versión %s disponible! Ejecuta 'edi --update' para actualizar.", rel.TagName), 6*time.Second)
			} else {
				e.SetStatus(fmt.Sprintf("edi está en la última versión (%s).", version.Version), 3*time.Second)
			}
			e.Screen.PostEvent(tcell.NewEventInterrupt(nil))
		}()
	default:
		if strings.HasPrefix(action, "buffer:goto_idx:") {
			idxStr := strings.TrimPrefix(action, "buffer:goto_idx:")
			if idx, err := strconv.Atoi(idxStr); err == nil && idx >= 0 && idx < len(e.Buffers) {
				e.SwitchToBuffer(idx)
			}
			return
		}
		if strings.HasPrefix(action, "plugin:") {
			e.runPluginCommand(strings.TrimPrefix(action, "plugin:"))
		}
	}
}

func (e *Editor) isCellSelected(row, col int) bool {
	if !e.Selecting {
		return false
	}
	startRow, startCol := e.SelectAnchorRow, e.SelectAnchorCol
	endRow, endCol := e.Buffer.CursorRow, e.Buffer.CursorCol

	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, startCol, endRow, endCol = endRow, endCol, startRow, startCol
	}

	if row < startRow || row > endRow {
		return false
	}
	if row == startRow && row == endRow {
		return col >= startCol && col < endCol
	}
	if row == startRow {
		return col >= startCol
	}
	if row == endRow {
		return col < endCol
	}
	return true
}

func (e *Editor) getSelectedText() string {
	if !e.Selecting {
		return ""
	}
	startRow, startCol := e.SelectAnchorRow, e.SelectAnchorCol
	endRow, endCol := e.Buffer.CursorRow, e.Buffer.CursorCol

	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, startCol, endRow, endCol = endRow, endCol, startRow, startCol
	}

	if startRow == endRow {
		runes := []rune(e.Buffer.Lines[startRow])
		if startCol > len(runes) {
			startCol = len(runes)
		}
		if endCol > len(runes) {
			endCol = len(runes)
		}
		return string(runes[startCol:endCol])
	}

	var parts []string
	firstRunes := []rune(e.Buffer.Lines[startRow])
	if startCol < len(firstRunes) {
		parts = append(parts, string(firstRunes[startCol:]))
	} else {
		parts = append(parts, "")
	}

	for r := startRow + 1; r < endRow; r++ {
		parts = append(parts, e.Buffer.Lines[r])
	}

	lastRunes := []rune(e.Buffer.Lines[endRow])
	if endCol > len(lastRunes) {
		endCol = len(lastRunes)
	}
	parts = append(parts, string(lastRunes[:endCol]))

	return strings.Join(parts, "\n")
}

func (e *Editor) deleteSelection() {
	if !e.Selecting {
		return
	}
	startRow, startCol := e.SelectAnchorRow, e.SelectAnchorCol
	endRow, endCol := e.Buffer.CursorRow, e.Buffer.CursorCol

	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, startCol, endRow, endCol = endRow, endCol, startRow, startCol
	}

	firstRunes := []rune(e.Buffer.Lines[startRow])
	if startCol > len(firstRunes) {
		startCol = len(firstRunes)
	}
	lastRunes := []rune(e.Buffer.Lines[endRow])
	if endCol > len(lastRunes) {
		endCol = len(lastRunes)
	}

	merged := string(firstRunes[:startCol]) + string(lastRunes[endCol:])
	newLines := append(e.Buffer.Lines[:startRow], append([]string{merged}, e.Buffer.Lines[endRow+1:]...)...)
	e.Buffer.Lines = newLines
	e.Buffer.CursorRow = startRow
	e.Buffer.CursorCol = startCol
	e.Buffer.ClampCursor()
	e.Selecting = false
	e.Buffer.Modified = true
}

func (e *Editor) pasteText(text string) {
	if text == "" {
		return
	}
	if e.Selecting {
		e.deleteSelection()
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		for _, r := range lines[0] {
			e.Buffer.InsertRune(r)
		}
		return
	}

	currRunes := []rune(e.Buffer.Lines[e.Buffer.CursorRow])
	col := e.Buffer.CursorCol
	if col > len(currRunes) {
		col = len(currRunes)
	}
	left := string(currRunes[:col])
	right := string(currRunes[col:])

	var newLines []string
	newLines = append(newLines, e.Buffer.Lines[:e.Buffer.CursorRow]...)
	newLines = append(newLines, left+lines[0])
	for i := 1; i < len(lines)-1; i++ {
		newLines = append(newLines, lines[i])
	}
	lastLine := lines[len(lines)-1]
	newLines = append(newLines, lastLine+right)
	newLines = append(newLines, e.Buffer.Lines[e.Buffer.CursorRow+1:]...)

	e.Buffer.Lines = newLines
	e.Buffer.CursorRow = e.Buffer.CursorRow + len(lines) - 1
	e.Buffer.CursorCol = len([]rune(lastLine))
	e.Buffer.ClampCursor()
	e.Buffer.Modified = true
}

func (e *Editor) saveFile() {
	if e.Buffer.FilePath == "" {
		e.PromptMode = PromptSaveAs
		e.PromptText = ""
		return
	}

	lang := ""
	if e.CurrentSyntax != nil {
		lang = e.CurrentSyntax.Name
	}

	content := strings.Join(e.Buffer.Lines, "\n")
	for _, p := range e.Plugins {
		if !p.MatchesLanguage(lang) {
			continue
		}
		for _, hook := range p.Hooks["on_save"] {
			if hook.Action == "pipe_buffer" {
				formatted, err := plugin.ExecutePipeBuffer(hook.Command, content)
				if err == nil {
					content = formatted
					e.Buffer.Lines = strings.Split(strings.TrimRight(content, "\n"), "\n")
					e.Buffer.ClampCursor()
				} else {
					e.SetStatus(fmt.Sprintf("[%s error] %v", p.Name, err), 3*time.Second)
				}
			}
		}
	}

	if err := e.Buffer.Save(); err != nil {
		e.SetStatus(fmt.Sprintf("Error guardando: %v", err), 3*time.Second)
	} else {
		e.detectCurrentSyntax()
		e.QuitWarned = false
		e.SetStatus(fmt.Sprintf("Guardado '%s' (%d líneas)", filepath.Base(e.Buffer.FilePath), e.Buffer.LineCount()), 2*time.Second)
		e.WorkspaceFiles = ScanWorkspaceFiles(3000)
		if e.FileTree != nil {
			e.FileTree.Refresh()
		}
		if e.GitManager != nil {
			e.GitManager.RefreshAsync(e.getWorkspaceDir(), func() {
				e.Screen.PostEvent(tcell.NewEventInterrupt(nil))
			})
		}
	}
}

func (e *Editor) runPluginCommand(cmdID string) {
	for _, p := range e.Plugins {
		for _, cmd := range p.Commands {
			if cmd.ID == cmdID {
				out, err := plugin.ExecuteShellCommand(cmd.Command)
				if err != nil {
					e.SetStatus(fmt.Sprintf("[Error %s] %s", cmd.Title, out), 4*time.Second)
				} else {
					msg := out
					if len(msg) > 60 {
						msg = msg[:60] + "..."
					}
					if msg == "" {
						msg = "Comando completado exitosamente"
					}
					e.SetStatus(fmt.Sprintf("[%s] %s", cmd.Title, msg), 4*time.Second)
				}
				return
			}
		}
	}
}

func (e *Editor) findNext(term string) {
	if term == "" || e.Buffer.LineCount() == 0 {
		return
	}

	lowerTerm := strings.ToLower(term)
	startRow := e.Buffer.CursorRow
	startCol := e.Buffer.CursorCol + 1

	for r := startRow; r < e.Buffer.LineCount(); r++ {
		lineRunes := []rune(e.Buffer.Lines[r])
		from := 0
		if r == startRow {
			from = startCol
		}
		if from < len(lineRunes) {
			subStr := strings.ToLower(string(lineRunes[from:]))
			byteIdx := strings.Index(subStr, lowerTerm)
			if byteIdx != -1 {
				runeOffset := len([]rune(subStr[:byteIdx]))
				e.Buffer.CursorRow = r
				e.Buffer.CursorCol = from + runeOffset
				e.Buffer.ClampCursor()
				e.SetStatus(fmt.Sprintf("Coincidencia en línea %d", r+1), 2*time.Second)
				return
			}
		}
	}

	for r := 0; r <= startRow && r < e.Buffer.LineCount(); r++ {
		lineRunes := []rune(e.Buffer.Lines[r])
		to := len(lineRunes)
		if r == startRow {
			to = startCol
			if to > len(lineRunes) {
				to = len(lineRunes)
			}
		}
		if to > 0 {
			subStr := strings.ToLower(string(lineRunes[:to]))
			byteIdx := strings.Index(subStr, lowerTerm)
			if byteIdx != -1 {
				runeOffset := len([]rune(subStr[:byteIdx]))
				e.Buffer.CursorRow = r
				e.Buffer.CursorCol = runeOffset
				e.Buffer.ClampCursor()
				e.SetStatus(fmt.Sprintf("Coincidencia en línea %d (bucle)", r+1), 2*time.Second)
				return
			}
		}
	}

	e.SetStatus(fmt.Sprintf("No se encontró '%s'", term), 2*time.Second)
}

func keyToString(ev *tcell.EventKey) string {
	var prefix string
	if ev.Modifiers()&tcell.ModCtrl != 0 {
		prefix += "Ctrl+"
	}
	if ev.Modifiers()&tcell.ModAlt != 0 {
		prefix += "Alt+"
	}

	switch ev.Key() {
	case tcell.KeyF1:
		return "F1"
	case tcell.KeyF2:
		return "F2"
	case tcell.KeyCtrlA:
		return "Ctrl+A"
	case tcell.KeyCtrlB:
		return "Ctrl+B"
	case tcell.KeyCtrlC:
		return "Ctrl+C"
	case tcell.KeyCtrlD:
		return "Ctrl+D"
	case tcell.KeyCtrlE:
		return "Ctrl+E"
	case tcell.KeyCtrlF:
		return "Ctrl+F"
	case tcell.KeyCtrlG:
		return "Ctrl+G"
	case tcell.KeyCtrlK:
		return "Ctrl+K"
	case tcell.KeyCtrlN:
		return "Ctrl+N"
	case tcell.KeyCtrlO:
		return "Ctrl+O"
	case tcell.KeyCtrlP:
		return "Ctrl+P"
	case tcell.KeyCtrlQ:
		return "Ctrl+Q"
	case tcell.KeyCtrlS:
		return "Ctrl+S"
	case tcell.KeyCtrlV:
		return "Ctrl+V"
	case tcell.KeyCtrlW:
		return "Ctrl+W"
	case tcell.KeyCtrlX:
		return "Ctrl+X"
	case tcell.KeyCtrlY:
		return "Ctrl+Y"
	case tcell.KeyCtrlZ:
		return "Ctrl+Z"
	case tcell.KeyTab:
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			return "Ctrl+Tab"
		}
	case tcell.KeyLeft:
		if ev.Modifiers()&tcell.ModAlt != 0 {
			return "Alt+Left"
		}
	case tcell.KeyRight:
		if ev.Modifiers()&tcell.ModAlt != 0 {
			return "Alt+Right"
		}
	case tcell.KeyUp:
		if ev.Modifiers()&tcell.ModAlt != 0 {
			return "Alt+Up"
		}
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			return "Ctrl+Up"
		}
	case tcell.KeyDown:
		if ev.Modifiers()&tcell.ModAlt != 0 {
			return "Alt+Down"
		}
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			return "Ctrl+Down"
		}
	case tcell.KeyPgUp:
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			return "Ctrl+PgUp"
		}
	case tcell.KeyPgDn:
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			return "Ctrl+PgDn"
		}
	}

	if ev.Key() == tcell.KeyRune {
		return prefix + string(ev.Rune())
	}

	return ""
}
