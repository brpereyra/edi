package editor

import (
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


