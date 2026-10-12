package buffer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
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

	// Soporte de modo binario / hexadecimal
	IsBinary bool
	ReadOnly bool
}

// binaryExtensions lista extensiones comunes que siempre deben abrirse en modo binario.
var binaryExtensions = map[string]bool{
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".webp":  true,
	".bmp":   true,
	".ico":   true,
	".tiff":  true,
	".tif":   true,
	".svgz":  true,
	".exe":   true,
	".bin":   true,
	".so":    true,
	".dylib": true,
	".dll":   true,
	".wasm":  true,
	".zip":   true,
	".tar":   true,
	".gz":    true,
	".bz2":   true,
	".xz":    true,
	".7z":    true,
	".pdf":   true,
	".mp3":   true,
	".mp4":   true,
	".wav":   true,
	".ogg":   true,
	".avi":   true,
	".mkv":   true,
	".flac":  true,
	".iso":   true,
	".dmg":   true,
	".class": true,
	".o":     true,
	".a":     true,
	".pyc":   true,
}

// IsBinaryData determina si una muestra de bytes contiene indicios claros de ser binario.
func IsBinaryData(sample []byte) bool {
	if len(sample) == 0 {
		return false
	}
	// Byte nulo (0x00) es el indicador estándar
	if bytes.IndexByte(sample, 0) != -1 {
		return true
	}
	// Caracteres UTF-8 inválidos
	if !utf8.Valid(sample) {
		return true
	}
	// Proporción de caracteres de control no imprimibles (excluyendo \t, \n, \r)
	nonPrintable := 0
	for _, b := range sample {
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			nonPrintable++
		}
	}
	return float64(nonPrintable)/float64(len(sample)) > 0.10
}

// FormatHexDump genera un volcado hexadecimal clásico (16 bytes por línea con offset y ASCII).
func FormatHexDump(data []byte) []string {
	if len(data) == 0 {
		return []string{"00000000                                                   ||"}
	}
	var lines []string
	total := len(data)
	for offset := 0; offset < total; offset += 16 {
		chunkLen := 16
		if offset+chunkLen > total {
			chunkLen = total - offset
		}
		chunk := data[offset : offset+chunkLen]

		var sb strings.Builder
		sb.Grow(78)
		sb.WriteString(fmt.Sprintf("%08X  ", offset))

		for i := 0; i < 16; i++ {
			if i == 8 {
				sb.WriteByte(' ')
			}
			if i < chunkLen {
				sb.WriteString(fmt.Sprintf("%02X ", chunk[i]))
			} else {
				sb.WriteString("   ")
			}
		}
		sb.WriteByte(' ')

		sb.WriteByte('|')
		for i := 0; i < chunkLen; i++ {
			b := chunk[i]
			if b >= 32 && b <= 126 {
				sb.WriteByte(b)
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteByte('|')

		lines = append(lines, sb.String())
	}
	return lines
}

// NewEmptyBuffer crea un buffer vacío con una sola línea en blanco.
func NewEmptyBuffer() *Buffer {
	return &Buffer{
		Lines:      []string{""},
		maxHistory: 100,
	}
}

// NewBufferFromFile lee un archivo desde el disco y crea un buffer. Si el archivo es binario, lo abre en modo Hex.
func NewBufferFromFile(path string) (*Buffer, error) {
	return newBufferFromFileInternal(path, false, false)
}

// NewBufferFromFileForceHex fuerza la apertura del archivo en modo hexadecimal.
func NewBufferFromFileForceHex(path string) (*Buffer, error) {
	return newBufferFromFileInternal(path, true, false)
}

// NewBufferFromFileForceText fuerza la apertura del archivo como texto plano.
func NewBufferFromFileForceText(path string) (*Buffer, error) {
	return newBufferFromFileInternal(path, false, true)
}

func newBufferFromFileInternal(path string, forceHex, forceText bool) (*Buffer, error) {
	b := &Buffer{
		FilePath:   path,
		maxHistory: 100,
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			b.Lines = []string{""}
			return b, nil
		}
		return nil, fmt.Errorf("error abriendo archivo %s: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo información de %s: %w", path, err)
	}

	isBin := false
	if !forceText && info.Size() > 0 {
		if forceHex {
			isBin = true
		} else {
			ext := strings.ToLower(filepath.Ext(path))
			if binaryExtensions[ext] {
				isBin = true
			} else {
				header := make([]byte, 4096)
				n, _ := file.Read(header)
				_, _ = file.Seek(0, io.SeekStart)
				if n > 0 && IsBinaryData(header[:n]) {
					isBin = true
				}
			}
		}
	}

	if isBin {
		const maxBinSize = 10 * 1024 * 1024
		readLimit := info.Size()
		truncated := false
		if readLimit > maxBinSize {
			readLimit = maxBinSize
			truncated = true
		}

		data := make([]byte, readLimit)
		n, err := io.ReadFull(file, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("error leyendo archivo binario %s: %w", path, err)
		}
		data = data[:n]

		b.Lines = FormatHexDump(data)
		if truncated {
			b.Lines = append(b.Lines, fmt.Sprintf("[Visualización truncada a %d MB]", maxBinSize/(1024*1024)))
		}
		b.IsBinary = true
		b.ReadOnly = true
		return b, nil
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		b.Lines = append(b.Lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		if !forceText {
			// Si falla la lectura por líneas excesivas o caracteres inválidos, abrir en modo hex automáticamente
			_, _ = file.Seek(0, io.SeekStart)
			data, readErr := io.ReadAll(io.LimitReader(file, 10*1024*1024))
			if readErr == nil && len(data) > 0 {
				b.Lines = FormatHexDump(data)
				b.IsBinary = true
				b.ReadOnly = true
				return b, nil
			}
		}
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
	if b.ReadOnly {
		return
	}
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
	if b.ReadOnly {
		return
	}
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
	if b.ReadOnly {
		return
	}
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
	if b.ReadOnly {
		return
	}
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
	if b.ReadOnly {
		return
	}
	b.recordSnapshot()
	b.ClampCursor()

	curr := b.Lines[b.CursorRow]
	b.Lines = append(b.Lines[:b.CursorRow+1], append([]string{curr}, b.Lines[b.CursorRow+1:]...)...)
	b.CursorRow++
	b.Modified = true
}

// DeleteLine elimina la línea donde se encuentra el cursor.
func (b *Buffer) DeleteLine() {
	if b.ReadOnly {
		return
	}
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
	if b.ReadOnly || startRow <= 0 || startRow > endRow || endRow >= len(b.Lines) {
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
	if b.ReadOnly || endRow >= len(b.Lines)-1 || startRow > endRow || startRow < 0 {
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
	if b.ReadOnly || b.IsBinary {
		return fmt.Errorf("archivo en modo binario / solo lectura no puede guardarse")
	}

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
