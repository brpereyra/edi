package buffer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Snapshot representa un estado completo del texto para Undo/Redo.
type Snapshot struct {
	Lines []string
	Row   int
	Col   int
}

// Buffer gestiona las líneas de texto, el estado de modificación y el cursor.
type Buffer struct {
	FilePath   string
	Lines      []string
	Modified   bool
	CursorRow  int
	CursorCol  int
	ScrollRow  int
	ScrollCol  int
	undoStack  []Snapshot
	redoStack  []Snapshot
	maxHistory int
}

// NewEmptyBuffer crea un buffer vacío con una sola línea en blanco.
func NewEmptyBuffer() *Buffer {
	return &Buffer{
		Lines:      []string{""},
		maxHistory: 100,
	}
}

// NewBufferFromFile lee un archivo desde el disco y crea un buffer.
func NewBufferFromFile(path string) (*Buffer, error) {
	b := &Buffer{
		FilePath:   path,
		maxHistory: 100,
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Si el archivo no existe todavía, lo inicializamos como nuevo
			b.Lines = []string{""}
			return b, nil
		}
		return nil, fmt.Errorf("error abriendo archivo %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		b.Lines = append(b.Lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error leyendo archivo %s: %w", path, err)
	}

	if len(b.Lines) == 0 {
		b.Lines = []string{""}
	}

	return b, nil
}

func (b *Buffer) recordSnapshot() {
	linesCopy := make([]string, len(b.Lines))
	copy(linesCopy, b.Lines)

	snap := Snapshot{
		Lines: linesCopy,
		Row:   b.CursorRow,
		Col:   b.CursorCol,
	}

	b.undoStack = append(b.undoStack, snap)
	if len(b.undoStack) > b.maxHistory {
		b.undoStack = b.undoStack[1:]
	}
	// Cualquier edición nueva invalida el stack de redo
	b.redoStack = nil
}

// Undo revierte la última acción.
func (b *Buffer) Undo() bool {
	if len(b.undoStack) == 0 {
		return false
	}

	// Guardar estado actual en redoStack
	linesCopy := make([]string, len(b.Lines))
	copy(linesCopy, b.Lines)
	currentSnap := Snapshot{
		Lines: linesCopy,
		Row:   b.CursorRow,
		Col:   b.CursorCol,
	}
	b.redoStack = append(b.redoStack, currentSnap)

	// Extraer último estado de undo
	last := b.undoStack[len(b.undoStack)-1]
	b.undoStack = b.undoStack[:len(b.undoStack)-1]

	b.Lines = last.Lines
	b.CursorRow = last.Row
	b.CursorCol = last.Col
	b.Modified = true
	b.ClampCursor()
	return true
}

// Redo rehace la última acción deshecha.
func (b *Buffer) Redo() bool {
	if len(b.redoStack) == 0 {
		return false
	}

	// Guardar estado actual en undoStack
	linesCopy := make([]string, len(b.Lines))
	copy(linesCopy, b.Lines)
	currentSnap := Snapshot{
		Lines: linesCopy,
		Row:   b.CursorRow,
		Col:   b.CursorCol,
	}
	b.undoStack = append(b.undoStack, currentSnap)

	// Extraer último estado de redo
	last := b.redoStack[len(b.redoStack)-1]
	b.redoStack = b.redoStack[:len(b.redoStack)-1]

	b.Lines = last.Lines
	b.CursorRow = last.Row
	b.CursorCol = last.Col
	b.Modified = true
	b.ClampCursor()
	return true
}

// InsertRune inserta un carácter en la posición del cursor actual.
func (b *Buffer) InsertRune(r rune) {
	b.recordSnapshot()
	b.ClampCursor()

	runes := []rune(b.Lines[b.CursorRow])
	col := b.CursorCol
	if col > len(runes) {
		col = len(runes)
	}

	newLine := string(runes[:col]) + string(r) + string(runes[col:])
	b.Lines[b.CursorRow] = newLine
	b.CursorCol++
	b.Modified = true
}

// SplitLine divide la línea actual en dos en la posición del cursor (tecla Enter).
func (b *Buffer) SplitLine(autoIndent bool) {
	b.recordSnapshot()
	b.ClampCursor()

	runes := []rune(b.Lines[b.CursorRow])
	col := b.CursorCol
	if col > len(runes) {
		col = len(runes)
	}

	left := string(runes[:col])
	right := string(runes[col:])

	indent := ""
	if autoIndent {
		for _, ch := range left {
			if ch == ' ' || ch == '\t' {
				indent += string(ch)
			} else {
				break
			}
		}
	}

	b.Lines[b.CursorRow] = left

	// Insertar nueva línea
	b.Lines = append(b.Lines[:b.CursorRow+1], append([]string{indent + right}, b.Lines[b.CursorRow+1:]...)...)
	b.CursorRow++
	b.CursorCol = len([]rune(indent))
	b.Modified = true
}

// DeleteRune elimina el carácter antes del cursor (Backspace).
func (b *Buffer) DeleteRune() {
	b.ClampCursor()

	if b.CursorCol > 0 {
		b.recordSnapshot()
		runes := []rune(b.Lines[b.CursorRow])
		col := b.CursorCol - 1
		newLine := string(runes[:col]) + string(runes[col+1:])
		b.Lines[b.CursorRow] = newLine
		b.CursorCol--
		b.Modified = true
	} else if b.CursorRow > 0 {
		// Unir la línea actual con la línea anterior
		b.recordSnapshot()
		prevRunes := []rune(b.Lines[b.CursorRow-1])
		currRunes := []rune(b.Lines[b.CursorRow])

		b.Lines[b.CursorRow-1] = string(prevRunes) + string(currRunes)
		// Eliminar línea actual
		b.Lines = append(b.Lines[:b.CursorRow], b.Lines[b.CursorRow+1:]...)
		b.CursorRow--
		b.CursorCol = len(prevRunes)
		b.Modified = true
	}
}

// DeleteForwardRune elimina el carácter bajo el cursor (tecla Supr / Delete).
func (b *Buffer) DeleteForwardRune() {
	b.ClampCursor()
	runes := []rune(b.Lines[b.CursorRow])

	if b.CursorCol < len(runes) {
		b.recordSnapshot()
		newLine := string(runes[:b.CursorCol]) + string(runes[b.CursorCol+1:])
		b.Lines[b.CursorRow] = newLine
		b.Modified = true
	} else if b.CursorRow < len(b.Lines)-1 {
		// Unir siguiente línea
		b.recordSnapshot()
		nextRunes := []rune(b.Lines[b.CursorRow+1])
		b.Lines[b.CursorRow] = string(runes) + string(nextRunes)
		b.Lines = append(b.Lines[:b.CursorRow+1], b.Lines[b.CursorRow+2:]...)
		b.Modified = true
	}
}

// DuplicateLine duplica la línea actual en la posición siguiente.
func (b *Buffer) DuplicateLine() {
	b.recordSnapshot()
	b.ClampCursor()

	curr := b.Lines[b.CursorRow]
	b.Lines = append(b.Lines[:b.CursorRow+1], append([]string{curr}, b.Lines[b.CursorRow+1:]...)...)
	b.CursorRow++
	b.Modified = true
}

// DeleteLine elimina la línea donde se encuentra el cursor.
func (b *Buffer) DeleteLine() {
	b.recordSnapshot()
	if len(b.Lines) <= 1 {
		b.Lines = []string{""}
		b.CursorRow = 0
		b.CursorCol = 0
	} else {
		b.Lines = append(b.Lines[:b.CursorRow], b.Lines[b.CursorRow+1:]...)
		if b.CursorRow >= len(b.Lines) {
			b.CursorRow = len(b.Lines) - 1
		}
	}
	b.ClampCursor()
	b.Modified = true
}

// MoveLinesUp mueve un rango de líneas [startRow, endRow] hacia arriba intercambiándolo con la línea anterior.
func (b *Buffer) MoveLinesUp(startRow, endRow int) bool {
	if startRow <= 0 || startRow > endRow || endRow >= len(b.Lines) {
		return false
	}

	b.recordSnapshot()
	prevLine := b.Lines[startRow-1]
	for i := startRow; i <= endRow; i++ {
		b.Lines[i-1] = b.Lines[i]
	}
	b.Lines[endRow] = prevLine

	b.Modified = true
	return true
}

// MoveLinesDown mueve un rango de líneas [startRow, endRow] hacia abajo intercambiándolo con la línea siguiente.
func (b *Buffer) MoveLinesDown(startRow, endRow int) bool {
	if endRow >= len(b.Lines)-1 || startRow > endRow || startRow < 0 {
		return false
	}

	b.recordSnapshot()
	nextLine := b.Lines[endRow+1]
	for i := endRow; i >= startRow; i-- {
		b.Lines[i+1] = b.Lines[i]
	}
	b.Lines[startRow] = nextLine

	b.Modified = true
	return true
}

// MoveLineUp mueve la línea actual hacia arriba una posición.
func (b *Buffer) MoveLineUp() bool {
	b.ClampCursor()
	if b.CursorRow <= 0 {
		return false
	}
	if b.MoveLinesUp(b.CursorRow, b.CursorRow) {
		b.CursorRow--
		b.ClampCursor()
		return true
	}
	return false
}

// MoveLineDown mueve la línea actual hacia abajo una posición.
func (b *Buffer) MoveLineDown() bool {
	b.ClampCursor()
	if b.CursorRow >= len(b.Lines)-1 {
		return false
	}
	if b.MoveLinesDown(b.CursorRow, b.CursorRow) {
		b.CursorRow++
		b.ClampCursor()
		return true
	}
	return false
}

// ClampCursor asegura que CursorRow y CursorCol se mantengan dentro de límites válidos.
func (b *Buffer) ClampCursor() {
	if b.CursorRow < 0 {
		b.CursorRow = 0
	}
	if b.CursorRow >= len(b.Lines) {
		b.CursorRow = len(b.Lines) - 1
	}

	lineRunes := []rune(b.Lines[b.CursorRow])
	if b.CursorCol < 0 {
		b.CursorCol = 0
	}
	if b.CursorCol > len(lineRunes) {
		b.CursorCol = len(lineRunes)
	}
}

// Save guarda el buffer en su ruta de archivo asignada.
func (b *Buffer) Save() error {
	if b.FilePath == "" {
		return fmt.Errorf("no hay ruta de archivo asignada para guardar")
	}

	dir := filepath.Dir(b.FilePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creando directorio: %w", err)
		}
	}

	content := strings.Join(b.Lines, "\n")
	if err := os.WriteFile(b.FilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("error escribiendo archivo %s: %w", b.FilePath, err)
	}

	b.Modified = false
	return nil
}

// LineCount retorna la cantidad total de líneas.
func (b *Buffer) LineCount() int {
	return len(b.Lines)
}

// GetLine retorna la línea en el índice indicado o cadena vacía si está fuera de rango.
func (b *Buffer) GetLine(index int) string {
	if index < 0 || index >= len(b.Lines) {
		return ""
	}
	return b.Lines[index]
}
