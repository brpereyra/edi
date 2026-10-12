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

func TestBinaryDetectionAndHexDump(t *testing.T) {
	// 1. Probar función IsBinaryData
	textSample := []byte("Hola mundo, esto es un texto plano sin bytes nulos.\n")
	if IsBinaryData(textSample) {
		t.Errorf("texto plano no debería ser detectado como binario")
	}

	binSample := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	if !IsBinaryData(binSample) {
		t.Errorf("muestra con bytes nulos debería ser detectada como binaria")
	}

	// 2. Probar FormatHexDump
	hexLines := FormatHexDump(binSample)
	if len(hexLines) != 1 {
		t.Fatalf("se esperaba 1 línea para 12 bytes, obtenidos %d", len(hexLines))
	}
	// Debe contener el offset 00000000 y el prefijo hex 89 50 4E 47
	if hexLines[0][:8] != "00000000" {
		t.Errorf("offset esperado 00000000, obtenido %s", hexLines[0][:8])
	}
	if hexLines[0][10:21] != "89 50 4E 47" {
		t.Errorf("primeros bytes esperados '89 50 4E 47', obtenidos '%s'", hexLines[0][10:21])
	}
}

func TestBinaryBufferReadOnlyProtections(t *testing.T) {
	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "avatar.png")
	data := []byte{0x89, 'P', 'N', 'G', 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D}
	if err := os.WriteFile(imagePath, data, 0644); err != nil {
		t.Fatalf("error creando archivo binario: %v", err)
	}

	b, err := NewBufferFromFile(imagePath)
	if err != nil {
		t.Fatalf("error abriendo buffer binario: %v", err)
	}

	if !b.IsBinary {
		t.Errorf("se esperaba b.IsBinary == true para archivo .png")
	}
	if !b.ReadOnly {
		t.Errorf("se esperaba b.ReadOnly == true para archivo binario")
	}

	initialLinesCount := len(b.Lines)
	if initialLinesCount < 2 {
		t.Errorf("se esperaban al menos 2 líneas para 18 bytes (16+2), obtenidas %d", initialLinesCount)
	}

	// Intentar mutaciones y verificar que son ignoradas
	b.InsertRune('X')
	b.SplitLine(false)
	b.DeleteRune()
	b.DeleteForwardRune()
	b.DuplicateLine()
	b.DeleteLine()
	b.MoveLineDown()
	b.MoveLineUp()

	if len(b.Lines) != initialLinesCount {
		t.Errorf("la cantidad de líneas no debería haber cambiado en modo ReadOnly")
	}
	if b.Modified {
		t.Errorf("el buffer ReadOnly no debería marcarse como modificado")
	}

	// Save() debe retornar error
	if err := b.Save(); err == nil {
		t.Errorf("se esperaba error al intentar guardar buffer binario ReadOnly")
	}
}

func TestForceHexAndForceText(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "sample.txt")
	_ = os.WriteFile(txtPath, []byte("Hello Plain Text\n"), 0644)

	// Abrir forzado a hex
	hexBuf, err := NewBufferFromFileForceHex(txtPath)
	if err != nil {
		t.Fatalf("error abriendo con ForceHex: %v", err)
	}
	if !hexBuf.IsBinary || !hexBuf.ReadOnly {
		t.Errorf("se esperaba IsBinary y ReadOnly true con ForceHex")
	}
	if hexBuf.Lines[0][:8] != "00000000" {
		t.Errorf("se esperaba formato hex dump")
	}

	// Abrir forzado a texto
	textBuf, err := NewBufferFromFileForceText(txtPath)
	if err != nil {
		t.Fatalf("error abriendo con ForceText: %v", err)
	}
	if textBuf.IsBinary || textBuf.ReadOnly {
		t.Errorf("se esperaba IsBinary y ReadOnly false con ForceText")
	}
	if textBuf.Lines[0] != "Hello Plain Text" {
		t.Errorf("se esperaba contenido de texto 'Hello Plain Text', obtenido '%s'", textBuf.Lines[0])
	}
}

