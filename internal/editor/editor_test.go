package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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


