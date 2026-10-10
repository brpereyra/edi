package editor

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"edi/internal/buffer"
	"edi/internal/config"
	"edi/internal/plugin"
	"edi/internal/syntax"
	"edi/internal/theme"

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
)

// PaletteItem representa un comando en la paleta de acciones rápidas.
type PaletteItem struct {
	Title       string
	Action      string
	Description string
}

// Editor representa la sesión activa del editor de texto.
type Editor struct {
	Screen        tcell.Screen
	Buffer        *buffer.Buffer
	Config        *config.EditorConfig
	Keybindings   map[string]string
	Theme         *theme.CompiledTheme
	Syntaxes      []*syntax.CompiledSyntax
	CurrentSyntax *syntax.CompiledSyntax
	Plugins       []*plugin.Plugin

	Running       bool
	StatusMessage string
	StatusExpires time.Time
	QuitWarned    bool

	PromptMode     PromptType
	PromptText     string
	PromptCursor   int
	PaletteItems   []PaletteItem
	PaletteIndex   int
	FilteredItems  []PaletteItem
	LastSearchTerm string
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

	ed := &Editor{
		Screen:       screen,
		Buffer:       buf,
		Config:       cfg,
		Keybindings:  keybindings,
		Theme:        th,
		Syntaxes:     syntaxes,
		Plugins:      plugins,
		Running:      true,
		PromptMode:   PromptNone,
		PaletteItems: buildDefaultPalette(),
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
		{Title: "Guardar archivo", Action: "buffer:save", Description: "Ctrl+S"},
		{Title: "Guardar como...", Action: "buffer:save_as", Description: "Guardar con nuevo nombre"},
		{Title: "Ir a línea...", Action: "buffer:goto_line", Description: "Ctrl+G"},
		{Title: "Buscar texto...", Action: "buffer:find", Description: "Ctrl+F"},
		{Title: "Deshacer (Undo)", Action: "buffer:undo", Description: "Ctrl+Z"},
		{Title: "Rehacer (Redo)", Action: "buffer:redo", Description: "Ctrl+Y"},
		{Title: "Duplicar línea", Action: "line:duplicate", Description: "Ctrl+D"},
		{Title: "Eliminar línea", Action: "line:delete", Description: "Ctrl+K"},
		{Title: "Cambiar tema: Dracula", Action: "theme:dracula", Description: "Tema oscuro clásico"},
		{Title: "Cambiar tema: Monokai", Action: "theme:monokai", Description: "Tema oscuro retro"},
		{Title: "Alternar números de línea", Action: "toggle:line_numbers", Description: "Mostrar/Ocultar"},
		{Title: "Alternar resaltado línea actual", Action: "toggle:current_line", Description: "Mostrar/Ocultar"},
		{Title: "Salir de edi", Action: "app:quit", Description: "Ctrl+Q"},
	}
}

func (e *Editor) detectCurrentSyntax() {
	if e.Buffer.FilePath != "" {
		e.CurrentSyntax = syntax.DetectSyntax(e.Buffer.FilePath, e.Syntaxes)
	}
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

	if height < 3 {
		e.Screen.Show()
		return
	}

	editorHeight := height - 2 // Reservar 2 líneas para status bar y prompt

	// Ajustar scroll vertical
	if e.Buffer.CursorRow < e.Buffer.ScrollRow {
		e.Buffer.ScrollRow = e.Buffer.CursorRow
	}
	if e.Buffer.CursorRow >= e.Buffer.ScrollRow+editorHeight {
		e.Buffer.ScrollRow = e.Buffer.CursorRow - editorHeight + 1
	}

	// Calcular ancho del margen de números de línea
	gutterWidth := 0
	if e.Config.LineNumbers {
		totalLines := e.Buffer.LineCount()
		gutterWidth = len(strconv.Itoa(totalLines)) + 2
	}

	// Ajustar scroll horizontal
	contentWidth := width - gutterWidth
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
		lineIdx := e.Buffer.ScrollRow + y
		isCursorLine := (lineIdx == e.Buffer.CursorRow)

		lineBgStyle := e.Theme.DefaultStyle
		if isCursorLine && e.Config.HighlightCurrentLine {
			lineBgStyle = e.Theme.CursorLineStyle
		}

		// Rellenar fondo de la línea
		for x := 0; x < width; x++ {
			e.Screen.SetContent(x, y, ' ', nil, lineBgStyle)
		}

		if lineIdx >= e.Buffer.LineCount() {
			// Más allá del final del archivo
			if !e.Config.LineNumbers {
				e.Screen.SetContent(0, y, '~', nil, e.Theme.LineNumberStyle)
			}
			continue
		}

		// Dibujar número de línea
		if e.Config.LineNumbers {
			numStr := fmt.Sprintf("%*d ", gutterWidth-1, lineIdx+1)
			numStyle := e.Theme.LineNumberStyle
			if isCursorLine {
				numStyle = e.Theme.LineNumberActive
			}
			for i, ch := range numStr {
				e.Screen.SetContent(i, y, ch, nil, numStyle)
			}
		}

		// Dibujar contenido de la línea con resaltado de sintaxis
		lineText := e.Buffer.GetLine(lineIdx)
		spans := syntax.HighlightLine(e.CurrentSyntax, lineText)

		// Expandir tabulaciones si aplica
		var renderedRunes []rune
		var runeStyles []tcell.Style

		lineRunes := []rune(lineText)
		byteOffset := 0

		for _, r := range lineRunes {
			runeLen := len(string(r))

			// Determinar estilo para esta posición
			st := lineBgStyle
			for _, span := range spans {
				if byteOffset >= span.Start && byteOffset < span.End {
					st = e.Theme.GetSyntaxStyle(span.Scope)
					if isCursorLine && e.Config.HighlightCurrentLine && e.Theme.CursorLineBg != tcell.ColorDefault {
						st = st.Background(e.Theme.CursorLineBg)
					}
					break
				}
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
		screenX := gutterWidth
		for colIdx := e.Buffer.ScrollCol; colIdx < len(renderedRunes) && screenX < width; colIdx++ {
			r := renderedRunes[colIdx]
			st := runeStyles[colIdx]
			e.Screen.SetContent(screenX, y, r, nil, st)
			screenX++
		}
	}

	// Renderizar Barra de Estado (Línea height - 2)
	e.drawStatusBar(width, height-2)

	// Renderizar Barra Inferior / Prompt / Paleta (Línea height - 1)
	e.drawBottomBar(width, height-1)

	// Posicionar el cursor
	if e.PromptMode != PromptNone {
		e.Screen.ShowCursor(e.PromptCursor, height-1)
	} else {
		cursorX := gutterWidth + (e.Buffer.CursorCol - e.Buffer.ScrollCol)
		cursorY := e.Buffer.CursorRow - e.Buffer.ScrollRow
		if cursorX >= gutterWidth && cursorX < width && cursorY >= 0 && cursorY < editorHeight {
			e.Screen.ShowCursor(cursorX, cursorY)
		} else {
			e.Screen.HideCursor()
		}
	}

	e.Screen.Show()
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

	left := fmt.Sprintf(" edi | %s%s", fileName, modTag)
	right := fmt.Sprintf("Ln %d, Col %d | %s | %s ", e.Buffer.CursorRow+1, e.Buffer.CursorCol+1, langName, e.Theme.Name)

	// Escribir izquierda
	for i, ch := range left {
		if i < width {
			e.Screen.SetContent(i, y, ch, nil, st)
		}
	}

	// Escribir derecha
	startX := width - len(right)
	if startX > len(left) {
		for i, ch := range right {
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
		e.PromptCursor = len("Buscar: ") + len([]rune(e.PromptText))

	case PromptGoto:
		prompt := fmt.Sprintf("Ir a línea: %s", e.PromptText)
		e.drawString(0, y, prompt, e.Theme.DefaultStyle)
		e.PromptCursor = len("Ir a línea: ") + len([]rune(e.PromptText))

	case PromptSaveAs:
		prompt := fmt.Sprintf("Guardar como: %s", e.PromptText)
		e.drawString(0, y, prompt, e.Theme.DefaultStyle)
		e.PromptCursor = len("Guardar como: ") + len([]rune(e.PromptText))

	case PromptPalette:
		prompt := fmt.Sprintf("Comandos: %s", e.PromptText)
		e.drawString(0, y, prompt, e.Theme.DefaultStyle)
		e.PromptCursor = len("Comandos: ") + len([]rune(e.PromptText))

		// Mostrar sugerencia destacada
		if len(e.FilteredItems) > 0 {
			selected := e.FilteredItems[e.PaletteIndex]
			hint := fmt.Sprintf("  -> [%s] %s", selected.Title, selected.Description)
			e.drawString(len(prompt)+2, y, hint, e.Theme.GetSyntaxStyle("keyword"))
		}

	default:
		// Mostrar status message temporal o atajos de ayuda
		if e.StatusMessage != "" && time.Now().Before(e.StatusExpires) {
			e.drawString(1, y, e.StatusMessage, e.Theme.GetSyntaxStyle("function"))
		} else {
			help := "^S: Guardar | ^Q: Salir | ^P: Comandos | ^F: Buscar | ^G: Ir a línea"
			e.drawString(1, y, help, e.Theme.LineNumberStyle)
		}
	}
}

func (e *Editor) drawString(x, y int, str string, st tcell.Style) {
	for i, ch := range str {
		e.Screen.SetContent(x+i, y, ch, nil, st)
	}
}

// HandleEvent procesa teclado, redimensionado y mouse.
func (e *Editor) HandleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		e.Screen.Sync()

	case *tcell.EventKey:
		if e.PromptMode != PromptNone {
			e.handlePromptKey(ev)
		} else {
			e.handleNormalKey(ev)
		}
	}
}

func (e *Editor) handleNormalKey(ev *tcell.EventKey) {
	// Comprobar atajos registrados en keybindings
	chord := keyToString(ev)
	if action, ok := e.Keybindings[chord]; ok {
		e.executeAction(action)
		return
	}

	// Teclas de control básicas
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
		e.Buffer.CursorRow -= height - 2
		e.Buffer.ClampCursor()
	case tcell.KeyPgDn:
		_, height := e.Screen.Size()
		e.Buffer.CursorRow += height - 2
		e.Buffer.ClampCursor()
	case tcell.KeyEnter:
		e.Buffer.SplitLine(e.Config.AutoIndent)
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		e.Buffer.DeleteRune()
	case tcell.KeyDelete:
		e.Buffer.DeleteForwardRune()
	case tcell.KeyTab:
		if e.Config.ExpandTabs {
			for i := 0; i < e.Config.TabSize; i++ {
				e.Buffer.InsertRune(' ')
			}
		} else {
			e.Buffer.InsertRune('\t')
		}
	case tcell.KeyRune:
		e.Buffer.InsertRune(ev.Rune())
		e.QuitWarned = false
	}
}

func (e *Editor) handlePromptKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		e.PromptMode = PromptNone
		e.PromptText = ""

	case tcell.KeyEnter:
		e.submitPrompt()

	case tcell.KeyUp:
		if e.PromptMode == PromptPalette && len(e.FilteredItems) > 0 {
			e.PaletteIndex = (e.PaletteIndex - 1 + len(e.FilteredItems)) % len(e.FilteredItems)
		}

	case tcell.KeyDown:
		if e.PromptMode == PromptPalette && len(e.FilteredItems) > 0 {
			e.PaletteIndex = (e.PaletteIndex + 1) % len(e.FilteredItems)
		}

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		runes := []rune(e.PromptText)
		if len(runes) > 0 {
			e.PromptText = string(runes[:len(runes)-1])
			if e.PromptMode == PromptPalette {
				e.updatePaletteFilter()
			}
		}

	case tcell.KeyRune:
		e.PromptText += string(ev.Rune())
		if e.PromptMode == PromptPalette {
			e.updatePaletteFilter()
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
		}

	case PromptPalette:
		e.PromptMode = PromptNone
		if len(e.FilteredItems) > 0 {
			selected := e.FilteredItems[e.PaletteIndex]
			e.executeAction(selected.Action)
		}
	}
}

func (e *Editor) executeAction(action string) {
	switch action {
	case "buffer:save":
		e.saveFile()
	case "buffer:save_as":
		e.PromptMode = PromptSaveAs
		e.PromptText = e.Buffer.FilePath
	case "app:quit":
		if e.Buffer.Modified && !e.QuitWarned {
			e.QuitWarned = true
			e.SetStatus("¡Cambios sin guardar! Presiona Ctrl+Q de nuevo para salir", 3*time.Second)
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
		e.PromptMode = PromptPalette
		e.PromptText = ""
		e.updatePaletteFilter()
	case "line:duplicate":
		e.Buffer.DuplicateLine()
	case "line:delete":
		e.Buffer.DeleteLine()
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
	default:
		if strings.HasPrefix(action, "plugin:") {
			e.runPluginCommand(strings.TrimPrefix(action, "plugin:"))
		}
	}
}

func (e *Editor) saveFile() {
	if e.Buffer.FilePath == "" {
		e.PromptMode = PromptSaveAs
		e.PromptText = ""
		return
	}

	// Ejecutar ganchos on_save de los plugins aplicables
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
					// Actualizar líneas del buffer con el formato
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
		e.SetStatus(fmt.Sprintf("Guardado '%s' (%d líneas)", e.Buffer.FilePath, e.Buffer.LineCount()), 2*time.Second)
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
	startRow := e.Buffer.CursorRow
	startCol := e.Buffer.CursorCol + 1

	for r := startRow; r < e.Buffer.LineCount(); r++ {
		line := e.Buffer.Lines[r]
		searchFrom := 0
		if r == startRow {
			searchFrom = startCol
		}
		if searchFrom < len(line) {
			idx := strings.Index(strings.ToLower(line[searchFrom:]), strings.ToLower(term))
			if idx != -1 {
				e.Buffer.CursorRow = r
				e.Buffer.CursorCol = searchFrom + idx
				e.Buffer.ClampCursor()
				e.SetStatus(fmt.Sprintf("Coincidencia en fila %d", r+1), 2*time.Second)
				return
			}
		}
	}

	// Si no se encuentra abajo, buscar desde el inicio hasta el cursor
	for r := 0; r <= startRow; r++ {
		line := e.Buffer.Lines[r]
		limit := len(line)
		if r == startRow {
			limit = startCol
		}
		idx := strings.Index(strings.ToLower(line[:limit]), strings.ToLower(term))
		if idx != -1 {
			e.Buffer.CursorRow = r
			e.Buffer.CursorCol = idx
			e.Buffer.ClampCursor()
			e.SetStatus(fmt.Sprintf("Coincidencia en fila %d (bucle)", r+1), 2*time.Second)
			return
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
	case tcell.KeyCtrlP:
		return "Ctrl+P"
	case tcell.KeyCtrlQ:
		return "Ctrl+Q"
	case tcell.KeyCtrlS:
		return "Ctrl+S"
	case tcell.KeyCtrlY:
		return "Ctrl+Y"
	case tcell.KeyCtrlZ:
		return "Ctrl+Z"
	}

	if ev.Key() == tcell.KeyRune {
		return prefix + string(ev.Rune())
	}

	return ""
}
