package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"edi/internal/buffer"
	"edi/internal/config"
	"github.com/gdamore/tcell/v2"
)

func TestFindNextBoundarySafety(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla de simulación: %v", err)
	}
	defer screen.Fini()

	buf := buffer.NewEmptyBuffer()
	buf.Lines = []string{
		"package main",
		"func main() {",
		"    // test",
		"}",
	}
	buf.CursorRow = 0
	buf.CursorCol = len([]rune(buf.Lines[0]))

	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	ed.findNext("main")

	if ed.Buffer.CursorRow != 1 {
		t.Errorf("se esperaba salto a línea 1 (func main), obtenido %d", ed.Buffer.CursorRow)
	}
}

func TestCommandPaletteFilter(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla de simulación: %v", err)
	}
	defer screen.Fini()

	buf := buffer.NewEmptyBuffer()
	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	ed.PromptMode = PromptPalette
	ed.PromptText = "guardar"
	ed.updatePaletteFilter()

	if len(ed.FilteredItems) == 0 {
		t.Errorf("se esperaba encontrar comando 'Guardar archivo'")
	}
}

func TestMultipleBuffersAndTabs(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()

	buf1 := buffer.NewEmptyBuffer()
	buf1.FilePath = "doc1.txt"

	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf1, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// Abrir segundo buffer
	tmpDir := t.TempDir()
	path2 := filepath.Join(tmpDir, "doc2.txt")
	ed.OpenFile(path2)

	if len(ed.Buffers) != 2 {
		t.Fatalf("se esperaban 2 buffers, obtenidos %d", len(ed.Buffers))
	}
	if ed.ActiveBufferIdx != 1 {
		t.Errorf("se esperaba buffer activo 1, obtenido %d", ed.ActiveBufferIdx)
	}

	// Alternar pestañas
	ed.executeAction("buffer:next")
	if ed.ActiveBufferIdx != 0 {
		t.Errorf("se esperaba buffer activo 0 tras buffer:next, obtenido %d", ed.ActiveBufferIdx)
	}

	// Cerrar pestaña activa
	ed.executeAction("buffer:close")
	if len(ed.Buffers) != 1 {
		t.Errorf("se esperaba 1 buffer tras cerrar, obtenidos %d", len(ed.Buffers))
	}
}

func TestSelectionAndClipboard(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()

	buf := buffer.NewEmptyBuffer()
	buf.Lines = []string{
		"primera linea",
		"segunda linea",
	}

	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// Seleccionar "primera"
	ed.Selecting = true
	ed.SelectAnchorRow = 0
	ed.SelectAnchorCol = 0
	ed.Buffer.CursorRow = 0
	ed.Buffer.CursorCol = 7

	selected := ed.getSelectedText()
	if selected != "primera" {
		t.Fatalf("texto seleccionado esperado 'primera', obtenido '%s'", selected)
	}

	// Copiar
	ed.executeAction("selection:copy")
	if ed.Clipboard != "primera" {
		t.Errorf("portapapeles esperado 'primera', obtenido '%s'", ed.Clipboard)
	}

	// Pegar en la segunda línea
	ed.Selecting = false
	ed.Buffer.CursorRow = 1
	ed.Buffer.CursorCol = 0
	ed.executeAction("selection:paste")

	if ed.Buffer.Lines[1] != "primerasegunda linea" {
		t.Errorf("contenido esperado 'primerasegunda linea', obtenido '%s'", ed.Buffer.Lines[1])
	}
}

func TestFileFinderScanAndFilter(t *testing.T) {
	files := ScanWorkspaceFiles(500)
	if len(files) == 0 {
		t.Fatalf("se esperaban archivos encontrados en el repositorio")
	}

	filtered := FilterFiles(files, "editor.go")
	if len(filtered) == 0 {
		t.Errorf("se esperaba encontrar 'editor.go' en el directorio del paquete")
	}
}

func TestEditorSidebarAndOpenFolder(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()

	buf := buffer.NewEmptyBuffer()
	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// Abrir carpeta temporal
	tmpDir := t.TempDir()
	ed.OpenFolder(tmpDir)

	if !ed.SidebarOpen {
		t.Errorf("se esperaba SidebarOpen = true tras OpenFolder")
	}
	if !ed.SidebarFocus {
		t.Errorf("se esperaba SidebarFocus = true tras OpenFolder")
	}

	// Alternar con Ctrl+B (view:toggle_tree)
	ed.executeAction("view:toggle_tree")
	if ed.SidebarOpen {
		t.Errorf("se esperaba SidebarOpen = false tras toggle")
	}
}

func TestEditorOpenHelp(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()

	buf := buffer.NewEmptyBuffer()
	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// Abrir ayuda
	ed.OpenHelp()

	if len(ed.Buffers) != 2 {
		t.Fatalf("se esperaban 2 buffers tras OpenHelp, obtenidos %d", len(ed.Buffers))
	}

	activeBuf := ed.Buffer
	if activeBuf.FilePath != "ayuda.md" {
		t.Errorf("se esperaba buffer 'ayuda.md', obtenido '%s'", activeBuf.FilePath)
	}
	if len(activeBuf.Lines) < 5 {
		t.Errorf("se esperaba contenido no vacío en buffer de ayuda")
	}
	if ed.CurrentSyntax == nil || ed.CurrentSyntax.Name != "Markdown" {
		t.Errorf("se esperaba resaltado Markdown en la ayuda, obtenido: %v", ed.CurrentSyntax)
	}
}

func TestSaveAsPromptAndAutoSave(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()

	buf := buffer.NewEmptyBuffer()
	buf.Lines = []string{"package main", "func main() {}"}
	buf.Modified = true

	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// Al intentar guardar un buffer sin nombre, debe activar el modal PromptSaveAs
	ed.executeAction("buffer:save")
	if ed.PromptMode != PromptSaveAs {
		t.Fatalf("se esperaba PromptMode = PromptSaveAs, obtenido %v", ed.PromptMode)
	}

	// Ingresar el nombre del archivo y enviar (Enter)
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "auto_saved.go")
	ed.PromptText = targetPath
	ed.submitPrompt()

	if ed.PromptMode != PromptNone {
		t.Errorf("se esperaba PromptMode = PromptNone tras submitPrompt, obtenido %v", ed.PromptMode)
	}
	if ed.Buffer.FilePath != targetPath {
		t.Errorf("se esperaba ruta '%s', obtenida '%s'", targetPath, ed.Buffer.FilePath)
	}
	if ed.Buffer.Modified {
		t.Errorf("el buffer debería estar marcado como no modificado tras guardado exitoso")
	}
	if ed.CurrentSyntax == nil || ed.CurrentSyntax.Name != "Go" {
		t.Errorf("se esperaba detección de sintaxis Go, obtenido: %v", ed.CurrentSyntax)
	}
}

func TestSidebarCreateNewFileWithKey(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()

	buf := buffer.NewEmptyBuffer()
	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	tmpDir := t.TempDir()
	ed.OpenFolder(tmpDir)

	// Simular tecla 'a' en el explorador
	ev := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	ed.handleSidebarKey(ev)

	if ed.PromptMode != PromptSaveAs {
		t.Fatalf("se esperaba que 'a' en sidebar abra PromptSaveAs, obtenido: %v", ed.PromptMode)
	}
	if len(ed.Buffers) != 2 {
		t.Fatalf("se esperaban 2 buffers tras presionar 'a', obtenidos: %d", len(ed.Buffers))
	}
}

func TestEditorMoveLine(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()

	buf := buffer.NewEmptyBuffer()
	buf.Lines = []string{"linea 1", "linea 2", "linea 3"}
	buf.CursorRow = 1 // linea 2
	buf.CursorCol = 2

	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// Mover linea 2 hacia arriba
	ed.executeAction("line:move_up")
	if ed.Buffer.Lines[0] != "linea 2" || ed.Buffer.Lines[1] != "linea 1" {
		t.Errorf("falló line:move_up, contenido: %v", ed.Buffer.Lines)
	}
	if ed.Buffer.CursorRow != 0 {
		t.Errorf("CursorRow esperado 0 tras mover arriba, obtenido %d", ed.Buffer.CursorRow)
	}

	// Mover linea 2 hacia abajo
	ed.executeAction("line:move_down")
	if ed.Buffer.Lines[0] != "linea 1" || ed.Buffer.Lines[1] != "linea 2" {
		t.Errorf("falló line:move_down, contenido: %v", ed.Buffer.Lines)
	}
	if ed.Buffer.CursorRow != 1 {
		t.Errorf("CursorRow esperado 1 tras mover abajo, obtenido %d", ed.Buffer.CursorRow)
	}

	// Probar con selección multi-línea
	ed.Selecting = true
	ed.SelectAnchorRow = 0
	ed.Buffer.CursorRow = 1 // Seleccionando lineas 0 y 1 ("linea 1", "linea 2")
	ed.executeAction("line:move_down")

	// Ahora linea 3 debe ser la primera, y linea 1 y 2 las siguientes
	if ed.Buffer.Lines[0] != "linea 3" || ed.Buffer.Lines[1] != "linea 1" || ed.Buffer.Lines[2] != "linea 2" {
		t.Errorf("falló line:move_down con selección, contenido: %v", ed.Buffer.Lines)
	}
	if ed.SelectAnchorRow != 1 || ed.Buffer.CursorRow != 2 {
		t.Errorf("esperado anchor 1 y cursor 2, obtenido anchor %d y cursor %d", ed.SelectAnchorRow, ed.Buffer.CursorRow)
	}
}

func TestEditorGitIntegration(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sample.txt")
	if err := os.WriteFile(testFile, []byte("hello world\n"), 0644); err != nil {
		t.Fatalf("error creando archivo: %v", err)
	}

	buf, err := buffer.NewBufferFromFile(testFile)
	if err != nil {
		t.Fatalf("error leyendo archivo: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.GitIntegration = true
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// Sin git repo inicial
	ed.executeAction("git:diff_file")
	if ed.StatusMessage != "No se detectó repositorio Git activo" {
		t.Errorf("mensaje inesperado sin git repo: %s", ed.StatusMessage)
	}

	// Inicializar git repo en tmpDir
	exec.Command("git", "-C", tmpDir, "init").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Tester").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "tester@test.com").Run()
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run()

	// Modificar archivo
	if err := os.WriteFile(testFile, []byte("hello world\nnew line\n"), 0644); err != nil {
		t.Fatalf("error modificando archivo: %v", err)
	}

	// Refrescar git
	ed.GitManager.RefreshSync(tmpDir)
	if !ed.GitManager.IsRepo() {
		t.Fatalf("se esperaba que detectara el repositorio git")
	}

	status := ed.GitManager.GetFileStatus(testFile)
	if status.Badge() == "" {
		t.Errorf("se esperaba estado modificado para %s", testFile)
	}

	// Ejecutar git:diff_file
	ed.executeAction("git:diff_file")
	if len(ed.Buffers) != 2 {
		t.Fatalf("se esperaba buffer virtual con el diff, total buffers: %d", len(ed.Buffers))
	}
	if filepath.Ext(ed.Buffer.FilePath) != ".diff" {
		t.Errorf("se esperaba extensión .diff, obtenido: %s", ed.Buffer.FilePath)
	}

	// Ejecutar git:diff_project
	ed.executeAction("git:diff_project")
	if len(ed.Buffers) != 3 {
		t.Fatalf("se esperaba buffer virtual para el diff del proyecto, total buffers: %d", len(ed.Buffers))
	}

	// Ejecutar git:status_files
	ed.executeAction("git:status_files")
	if ed.PromptMode != PromptFileFinder {
		t.Errorf("se esperaba que git:status_files abriera el selector de archivos")
	}
}

func TestFileFinderAndTabSwitching(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()

	tmpDir := t.TempDir()
	pathA := filepath.Join(tmpDir, "fileA.txt")
	pathB := filepath.Join(tmpDir, "fileB.txt")
	os.WriteFile(pathA, []byte("contenido A"), 0644)
	os.WriteFile(pathB, []byte("contenido B"), 0644)

	bufA, err := buffer.NewBufferFromFile(pathA)
	if err != nil {
		t.Fatalf("error creando buffer A: %v", err)
	}

	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, bufA, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// Abrir fileB usando OpenFile
	ed.OpenFile(pathB)
	if len(ed.Buffers) != 2 {
		t.Fatalf("se esperaban 2 buffers, obtenidos %d", len(ed.Buffers))
	}
	if ed.ActiveBufferIdx != 1 {
		t.Fatalf("se esperaba buffer activo 1 (fileB)")
	}

	// Probar cambiar a fileA via file:find
	ed.executeAction("file:find")
	if ed.PromptMode != PromptFileFinder {
		t.Fatalf("se esperaba PromptFileFinder activo")
	}
	ed.PromptText = "fileA"
	ed.FilteredFiles = FilterFiles(ed.WorkspaceFiles, ed.PromptText)
	ed.FileFinderIdx = 0
	ed.submitPrompt()

	if ed.ActiveBufferIdx != 0 {
		t.Errorf("se esperaba salto a pestaña 0 (fileA), obtenido %d", ed.ActiveBufferIdx)
	}
	if len(ed.Buffers) != 2 {
		t.Errorf("no debería duplicar buffers, se esperaban 2, obtenidos %d", len(ed.Buffers))
	}

	// Probar cambiar via Paleta de Comandos
	ed.executeAction("palette:open")
	if ed.PromptMode != PromptPalette {
		t.Fatalf("se esperaba PromptPalette activo")
	}
	// La paleta debe contener los tabs abiertos al inicio
	foundTab := false
	for _, item := range ed.FilteredItems {
		if item.Title == "Pestaña: fileB.txt" {
			foundTab = true
			ed.executeAction(item.Action)
			break
		}
	}
	if !foundTab {
		t.Errorf("se esperaba encontrar 'Pestaña: fileB.txt' en la paleta de comandos")
	}
	if ed.ActiveBufferIdx != 1 {
		t.Errorf("se esperaba salto a pestaña 1 (fileB) desde la paleta, obtenido %d", ed.ActiveBufferIdx)
	}
}

func TestDrawStringAccentsRuneAlignment(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla de simulación: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)

	buf := buffer.NewEmptyBuffer()
	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// Probar dibujar texto con acentos y símbolos UTF-8
	testStr := "[↵ Guardar automáticamente | Esc Cancelar]"
	st := tcell.StyleDefault
	ed.drawString(2, 5, testStr, st)

	runes := []rune(testStr)
	for col, expectedRune := range runes {
		r, _, _, _ := screen.GetContent(2+col, 5)
		if r != expectedRune {
			t.Errorf("en columna %d se esperaba rune '%c' (%U), obtenido '%c' (%U)", 2+col, expectedRune, expectedRune, r, r)
		}
	}
}

func TestSidebarPositionLeftRight(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)

	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hola"), 0644)

	buf := buffer.NewEmptyBuffer()
	cfg := config.DefaultConfig()
	cfg.SidebarPosition = "right"

	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	ed.OpenFolder(tmpDir)
	if !ed.SidebarOpen {
		t.Fatalf("se esperaba sidebar abierto")
	}

	// Renderizar con sidebar a la derecha
	ed.Draw()

	// Probar toggle de posición via paleta
	ed.executeAction("view:toggle_sidebar_pos")
	if ed.Config.SidebarPosition != "left" {
		t.Errorf("se esperaba posición 'left' tras toggle, obtenida '%s'", ed.Config.SidebarPosition)
	}
	ed.Draw()

	ed.executeAction("view:toggle_sidebar_pos")
	if ed.Config.SidebarPosition != "right" {
		t.Errorf("se esperaba posición 'right' tras toggle, obtenida '%s'", ed.Config.SidebarPosition)
	}
	ed.Draw()

	// Probar clic del mouse en sidebar a la derecha (x >= 80 - 24 = 56)
	ed.handleMouseClick(70, 2)
	if !ed.SidebarFocus {
		t.Errorf("se esperaba que el clic en x=70 enfocara el sidebar a la derecha")
	}
}

func TestGlobalSearchModalOpenAndNavigate(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla de simulación: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(100, 30)

	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte("package main\n\nfunc MySpecialTarget() {\n}\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("// MySpecialTarget reference\nvar MySpecialTarget = 10\n"), 0644)

	buf := buffer.NewEmptyBuffer()
	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	ed.OpenFolder(tmpDir)

	// 1. Abrir modal via shortcut chord Alt+F o acción
	ed.handleNormalKey(tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModAlt))
	if ed.PromptMode != PromptGlobalSearch {
		t.Fatalf("se esperaba PromptMode == PromptGlobalSearch, obtenido %v", ed.PromptMode)
	}

	// 2. Escribir texto de búsqueda "MySpecialTarget"
	for _, ch := range "MySpecialTarget" {
		ed.HandleEvent(tcell.NewEventKey(tcell.KeyRune, ch, 0))
	}

	// Esperar ejecución asíncrona del debounce
	time.Sleep(150 * time.Millisecond)

	ed.SearchMutex.Lock()
	matchesCount := ed.SearchTotalMatches
	filesCount := len(ed.SearchResults)
	ed.SearchMutex.Unlock()

	if matchesCount < 2 || filesCount < 2 {
		t.Fatalf("se esperaban al menos 2 matches en 2 archivos, obtenidos %d en %d", matchesCount, filesCount)
	}

	// 3. Renderizar modal
	ed.Draw()

	// 4. Navegar entre paneles con Tab
	ed.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if ed.SearchPanel != 1 {
		t.Errorf("se esperaba foco en panel 1 (archivos), obtenido %d", ed.SearchPanel)
	}

	ed.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if ed.SearchPanel != 2 {
		t.Errorf("se esperaba foco en panel 2 (incidencias), obtenido %d", ed.SearchPanel)
	}

	ed.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if ed.SearchPanel != 0 {
		t.Errorf("se esperaba foco en panel 0 (input), obtenido %d", ed.SearchPanel)
	}

	// 5. Flecha abajo desde input enfoca panel 1
	ed.HandleEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if ed.SearchPanel != 1 {
		t.Errorf("se esperaba que KeyDown cambiara foco a panel 1, obtenido %d", ed.SearchPanel)
	}

	// 6. Flecha derecha cambia a panel 2 (incidencias)
	ed.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if ed.SearchPanel != 2 {
		t.Errorf("se esperaba que KeyRight cambiara foco a panel 2, obtenido %d", ed.SearchPanel)
	}

	// 7. Flecha izquierda regresa a panel 1
	ed.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	if ed.SearchPanel != 1 {
		t.Errorf("se esperaba que KeyLeft cambiara foco a panel 1, obtenido %d", ed.SearchPanel)
	}

	// 8. Enter abre el archivo seleccionado y posiciona el cursor
	ed.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if ed.PromptMode != PromptNone {
		t.Errorf("se esperaba que Enter cerrara el modal, obtenido %v", ed.PromptMode)
	}
	if ed.Buffer.FilePath == "" {
		t.Errorf("se esperaba que se abriera un archivo en el buffer activo")
	}
}

func TestGlobalSearchOptionsToggleAndMouse(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla de simulación: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(100, 30)

	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("CaseText\ncasetext\n"), 0644)

	buf := buffer.NewEmptyBuffer()
	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}
	ed.OpenFolder(tmpDir)

	ed.openGlobalSearchModal()

	// Probar toggles de atajos Alt+C, Alt+R, Alt+Z
	ed.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModAlt))
	if !ed.SearchCaseSensitive {
		t.Errorf("se esperaba SearchCaseSensitive == true")
	}

	ed.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModAlt))
	if !ed.SearchRegex {
		t.Errorf("se esperaba SearchRegex == true")
	}

	ed.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModAlt))
	if !ed.SearchFuzzy {
		t.Errorf("se esperaba SearchFuzzy == true")
	}

	// Probar toggle de vuelta
	ed.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModAlt))
	if ed.SearchCaseSensitive {
		t.Errorf("se esperaba SearchCaseSensitive == false")
	}

	// Probar clic de mouse para cerrar fuera del modal (x=0, y=0)
	ed.handleMouseClick(0, 0)
	if ed.PromptMode != PromptNone {
		t.Errorf("se esperaba que clic fuera del modal lo cerrara")
	}
}

func TestEditorBinaryFileOpenAndHexView(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("error inicializando pantalla de simulación: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(100, 30)

	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "logo.png")
	pngBytes := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 'I', 'H', 'D', 'R', 0x00, 0x01}
	_ = os.WriteFile(imagePath, pngBytes, 0644)

	buf := buffer.NewEmptyBuffer()
	cfg := config.DefaultConfig()
	ed, err := NewEditor(screen, buf, &cfg)
	if err != nil {
		t.Fatalf("error creando editor: %v", err)
	}

	// 1. Abrir archivo de imagen binario
	ed.OpenFile(imagePath)

	if !ed.Buffer.IsBinary {
		t.Errorf("se esperaba ed.Buffer.IsBinary == true para imagen")
	}
	if !ed.Buffer.ReadOnly {
		t.Errorf("se esperaba ed.Buffer.ReadOnly == true para imagen")
	}

	// 2. Renderizar (debe renderizar modo hex sin gutter y con sintaxis Hex)
	ed.Draw()

	// 3. Intentar escribir caracteres (debe bloquearse y mostrar aviso)
	linesBefore := len(ed.Buffer.Lines)
	ed.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'Z', 0))
	if len(ed.Buffer.Lines) != linesBefore {
		t.Errorf("no se debería modificar el buffer en modo binario")
	}

	// 4. Intentar guardar (debe bloquearse y mostrar aviso)
	ed.executeAction("buffer:save")
	if ed.Buffer.Modified {
		t.Errorf("el buffer no debe marcarse como modificado")
	}

	// 5. Probar acción toggle_hex
	ed.executeAction("buffer:toggle_hex")
	if ed.Buffer.IsBinary {
		t.Errorf("se esperaba que toggle_hex conmutara a texto")
	}

	ed.executeAction("buffer:toggle_hex")
	if !ed.Buffer.IsBinary {
		t.Errorf("se esperaba que toggle_hex regresara a modo hexadecimal")
	}
}



