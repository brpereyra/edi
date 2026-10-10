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
