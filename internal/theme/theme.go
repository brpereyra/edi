package theme

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"edi/internal/config"
	"github.com/gdamore/tcell/v2"
)

// RawTheme representa la estructura deserializada de un tema JSON.
type RawTheme struct {
	Name   string            `json:"name"`
	UI     RawUITheme        `json:"ui"`
	Syntax map[string]string `json:"syntax"`
}

// RawUITheme contiene los colores de los elementos de interfaz.
type RawUITheme struct {
	Background       string `json:"background"`
	Foreground       string `json:"foreground"`
	Cursor           string `json:"cursor"`
	CursorLine       string `json:"cursor_line"`
	LineNumber       string `json:"line_number"`
	LineNumberActive string `json:"line_number_active"`
	StatusBarBg      string `json:"status_bar_bg"`
	StatusBarFg      string `json:"status_bar_fg"`
	Selection        string `json:"selection"`
}

// CompiledTheme almacena los estilos tcell.Style precomputados para máximo rendimiento en el bucle de renderizado.
type CompiledTheme struct {
	Name             string
	DefaultStyle     tcell.Style
	CursorLineStyle  tcell.Style
	CursorLineBg     tcell.Color
	LineNumberStyle  tcell.Style
	LineNumberActive tcell.Style
	StatusBarStyle   tcell.Style
	SelectionStyle   tcell.Style
	SyntaxStyles     map[string]tcell.Style
}

// ParseHexColor convierte una cadena "#rrggbb" o "rrggbb" en un tcell.Color TrueColor.
func ParseHexColor(hexStr string) tcell.Color {
	hexStr = strings.TrimPrefix(hexStr, "#")
	if len(hexStr) == 3 {
		// Formato corto "#rgb" -> expandir a "#rrggbb"
		hexStr = string([]byte{hexStr[0], hexStr[0], hexStr[1], hexStr[1], hexStr[2], hexStr[2]})
	}
	if len(hexStr) != 6 {
		return tcell.ColorDefault
	}

	val, err := strconv.ParseInt(hexStr, 16, 32)
	if err != nil {
		return tcell.ColorDefault
	}

	r := int32((val >> 16) & 0xFF)
	g := int32((val >> 8) & 0xFF)
	b := int32(val & 0xFF)

	return tcell.NewRGBColor(r, g, b)
}

// LoadTheme carga y compila un tema por su nombre (ej: "dracula" o "monokai").
func LoadTheme(name string) (*CompiledTheme, error) {
	if name == "" {
		name = "dracula"
	}

	subPath := fmt.Sprintf("themes/%s.json", name)
	data, err := config.ReadFileFromConfigOrEmbed(subPath)
	if err != nil {
		// Fallback a dracula si el tema solicitado no existe
		subPath = "themes/dracula.json"
		data, err = config.ReadFileFromConfigOrEmbed(subPath)
		if err != nil {
			return nil, fmt.Errorf("no se pudo cargar ningún tema: %w", err)
		}
	}

	cleanData := config.StripJSONComments(data)
	var raw RawTheme
	if err := json.Unmarshal(cleanData, &raw); err != nil {
		return nil, fmt.Errorf("error parseando tema %s: %w", name, err)
	}

	return compileTheme(&raw), nil
}

func compileTheme(raw *RawTheme) *CompiledTheme {
	bg := ParseHexColor(raw.UI.Background)
	fg := ParseHexColor(raw.UI.Foreground)
	cursorLineBg := ParseHexColor(raw.UI.CursorLine)
	lineNumFg := ParseHexColor(raw.UI.LineNumber)
	lineNumActFg := ParseHexColor(raw.UI.LineNumberActive)
	statusBg := ParseHexColor(raw.UI.StatusBarBg)
	statusFg := ParseHexColor(raw.UI.StatusBarFg)
	selectionBg := ParseHexColor(raw.UI.Selection)

	defaultStyle := tcell.StyleDefault.Background(bg).Foreground(fg)

	cursorLineStyle := defaultStyle
	if raw.UI.CursorLine != "" {
		cursorLineStyle = cursorLineStyle.Background(cursorLineBg)
	}

	lineNumStyle := tcell.StyleDefault.Background(bg).Foreground(lineNumFg)
	lineNumActStyle := tcell.StyleDefault.Background(bg).Foreground(lineNumActFg).Bold(true)
	statusStyle := tcell.StyleDefault.Background(statusBg).Foreground(statusFg)
	selectionStyle := tcell.StyleDefault.Background(selectionBg).Foreground(fg)

	syntaxMap := make(map[string]tcell.Style)
	for scope, colorHex := range raw.Syntax {
		tokenFg := ParseHexColor(colorHex)
		syntaxMap[scope] = defaultStyle.Foreground(tokenFg)
	}

	return &CompiledTheme{
		Name:             raw.Name,
		DefaultStyle:     defaultStyle,
		CursorLineStyle:  cursorLineStyle,
		CursorLineBg:     cursorLineBg,
		LineNumberStyle:  lineNumStyle,
		LineNumberActive: lineNumActStyle,
		StatusBarStyle:   statusStyle,
		SelectionStyle:   selectionStyle,
		SyntaxStyles:     syntaxMap,
	}
}

// GetSyntaxStyle retorna el tcell.Style para un scope dado o el DefaultStyle si no existe.
func (ct *CompiledTheme) GetSyntaxStyle(scope string) tcell.Style {
	if style, ok := ct.SyntaxStyles[scope]; ok {
		return style
	}
	return ct.DefaultStyle
}
