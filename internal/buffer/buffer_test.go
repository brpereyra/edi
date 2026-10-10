package buffer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBufferBasicEditing(t *testing.T) {
	b := NewEmptyBuffer()

	// Escribir "Hello"
	for _, r := range "Hello" {
		b.InsertRune(r)
	}

	if b.Lines[0] != "Hello" {
		t.Errorf("se esperaba 'Hello', se obtuvo '%s'", b.Lines[0])
	}
	if b.CursorCol != 5 {
		t.Errorf("cursor col esperado 5, obtenido %d", b.CursorCol)
	}

	// Presionar Enter con autoIndent
	b.SplitLine(true)
	if len(b.Lines) != 2 {
		t.Fatalf("se esperaban 2 líneas, se obtuvieron %d", len(b.Lines))
	}
	if b.CursorRow != 1 || b.CursorCol != 0 {
		t.Errorf("cursor esperado (1, 0), obtenido (%d, %d)", b.CursorRow, b.CursorCol)
	}

	// Escribir "World"
	for _, r := range "World" {
		b.InsertRune(r)
	}
	if b.Lines[1] != "World" {
		t.Errorf("se esperaba 'World', se obtuvo '%s'", b.Lines[1])
	}

	// Backspace en el inicio de la línea 1 -> debe unir líneas
	b.CursorCol = 0
	b.DeleteRune()
	if len(b.Lines) != 1 {
		t.Fatalf("se esperaba 1 línea tras Join, se obtuvieron %d", len(b.Lines))
	}
	if b.Lines[0] != "HelloWorld" {
		t.Errorf("se esperaba 'HelloWorld', se obtuvo '%s'", b.Lines[0])
	}
}

func TestBufferUndoRedo(t *testing.T) {
	b := NewEmptyBuffer()
	b.InsertRune('A')
	b.InsertRune('B')

	if b.Lines[0] != "AB" {
		t.Fatalf("se esperaba 'AB', se obtuvo '%s'", b.Lines[0])
	}

	// Undo 1
	b.Undo()
	if b.Lines[0] != "A" {
		t.Errorf("se esperaba 'A' tras primer Undo, se obtuvo '%s'", b.Lines[0])
	}

	// Undo 2
	b.Undo()
	if b.Lines[0] != "" {
		t.Errorf("se esperaba '' tras segundo Undo, se obtuvo '%s'", b.Lines[0])
	}

	// Redo 1
	b.Redo()
	if b.Lines[0] != "A" {
		t.Errorf("se esperaba 'A' tras primer Redo, se obtuvo '%s'", b.Lines[0])
	}

	// Redo 2
	b.Redo()
	if b.Lines[0] != "AB" {
		t.Errorf("se esperaba 'AB' tras segundo Redo, se obtuvo '%s'", b.Lines[0])
	}
}

func TestBufferSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	b := NewEmptyBuffer()
	b.FilePath = filePath
	for _, r := range "Línea 1" {
		b.InsertRune(r)
	}
	b.SplitLine(false)
	for _, r := range "Línea 2" {
		b.InsertRune(r)
	}

	if err := b.Save(); err != nil {
		t.Fatalf("error guardando buffer: %v", err)
	}

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("archivo no encontrado en disco tras Save: %v", err)
	}

	loaded, err := NewBufferFromFile(filePath)
	if err != nil {
		t.Fatalf("error cargando buffer desde archivo: %v", err)
	}

	if len(loaded.Lines) != 2 {
		t.Fatalf("se esperaban 2 líneas en archivo cargado, se obtuvieron %d", len(loaded.Lines))
	}
	if loaded.Lines[0] != "Línea 1" || loaded.Lines[1] != "Línea 2" {
		t.Errorf("contenido inesperado: %v", loaded.Lines)
	}
}

func TestMoveLineUpDown(t *testing.T) {
	b := NewEmptyBuffer()
	b.Lines = []string{"A", "B", "C", "D"}
	b.CursorRow = 2 // Sobre "C"

	// Mover "C" hacia arriba
	if !b.MoveLineUp() {
		t.Fatalf("MoveLineUp debería retornar true")
	}
	if b.CursorRow != 1 {
		t.Errorf("CursorRow esperado 1 tras mover hacia arriba, obtenido %d", b.CursorRow)
	}
	expected := []string{"A", "C", "B", "D"}
	for i, line := range expected {
		if b.Lines[i] != line {
			t.Errorf("línea %d esperada '%s', obtenida '%s'", i, line, b.Lines[i])
		}
	}

	// Mover "C" hacia abajo
	if !b.MoveLineDown() {
		t.Fatalf("MoveLineDown debería retornar true")
	}
	if b.CursorRow != 2 {
		t.Errorf("CursorRow esperado 2 tras mover hacia abajo, obtenido %d", b.CursorRow)
	}
	expected2 := []string{"A", "B", "C", "D"}
	for i, line := range expected2 {
		if b.Lines[i] != line {
			t.Errorf("línea %d esperada '%s', obtenida '%s'", i, line, b.Lines[i])
		}
	}

	// Mover rango ["B", "C"] hacia abajo
	if !b.MoveLinesDown(1, 2) {
		t.Fatalf("MoveLinesDown rango debería retornar true")
	}
	expectedRangeDown := []string{"A", "D", "B", "C"}
	for i, line := range expectedRangeDown {
		if b.Lines[i] != line {
			t.Errorf("línea %d esperada '%s', obtenida '%s'", i, line, b.Lines[i])
		}
	}

	// Deshacer con Undo
	b.Undo()
	for i, line := range expected2 {
		if b.Lines[i] != line {
			t.Errorf("después de Undo línea %d esperada '%s', obtenida '%s'", i, line, b.Lines[i])
		}
	}
}

