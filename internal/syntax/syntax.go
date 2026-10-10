package syntax

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"edi/internal/config"
)

// SyntaxRule define una regla léxica que asocia una expresión regular con un scope de color.
type SyntaxRule struct {
	Scope   string `json:"scope"`
	Pattern string `json:"pattern"`
}

// RawSyntax representa el archivo JSON de definición de sintaxis de un lenguaje.
type RawSyntax struct {
	Name       string       `json:"name"`
	Extensions []string     `json:"extensions"`
	Rules      []SyntaxRule `json:"rules"`
}

// CompiledRule contiene la expresión regular precompilada en memoria para máxima velocidad.
type CompiledRule struct {
	Scope string
	Regex *regexp.Regexp
}

// CompiledSyntax representa una definición de lenguaje lista para resaltar código.
type CompiledSyntax struct {
	Name       string
	Extensions []string
	Rules      []CompiledRule
}

// TokenSpan representa un rango de caracteres en una línea con un scope semántico.
type TokenSpan struct {
	Start int
	End   int
	Scope string
}

// LoadAllSyntaxes descubre y compila todos los lenguajes disponibles (físicos y embebidos).
func LoadAllSyntaxes() []*CompiledSyntax {
	files := config.ListJSONFilesInSubdir("syntax")
	var syntaxes []*CompiledSyntax

	for _, fileName := range files {
		subPath := filepath.Join("syntax", fileName)
		data, err := config.ReadFileFromConfigOrEmbed(subPath)
		if err != nil {
			continue
		}

		cleanData := config.StripJSONComments(data)
		var raw RawSyntax
		if err := json.Unmarshal(cleanData, &raw); err != nil {
			continue
		}

		compiled := &CompiledSyntax{
			Name:       raw.Name,
			Extensions: raw.Extensions,
		}

		for _, rule := range raw.Rules {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				continue
			}
			compiled.Rules = append(compiled.Rules, CompiledRule{
				Scope: rule.Scope,
				Regex: re,
			})
		}

		syntaxes = append(syntaxes, compiled)
	}

	return syntaxes
}

// DetectSyntax encuentra la sintaxis correspondiente basándose en la extensión o nombre del archivo.
func DetectSyntax(filePath string, syntaxes []*CompiledSyntax) *CompiledSyntax {
	if filePath == "" || len(syntaxes) == 0 {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	base := strings.ToLower(filepath.Base(filePath))

	for _, s := range syntaxes {
		for _, e := range s.Extensions {
			eLower := strings.ToLower(e)
			if eLower == ext || eLower == base {
				return s
			}
		}
	}

	return nil
}

// HighlightLine calcula los spans coloreados para una línea dada.
// Implementa resolución de prioridad (comentarios y strings tienen precedencia sobre palabras clave).
func HighlightLine(s *CompiledSyntax, line string) []TokenSpan {
	if s == nil || len(s.Rules) == 0 || line == "" {
		return nil
	}

	// Arreglo para marcar scopes por cada byte de la línea
	byteScopes := make([]string, len(line))

	// Pasada 1: Comentarios y Strings (prioridad alta)
	for _, rule := range s.Rules {
		if rule.Scope != "comment" && rule.Scope != "string" {
			continue
		}
		matches := rule.Regex.FindAllStringIndex(line, -1)
		for _, match := range matches {
			start, end := match[0], match[1]
			for i := start; i < end; i++ {
				byteScopes[i] = rule.Scope
			}
		}
	}

	// Pasada 2: Resto de reglas (keywords, types, numbers, functions) solo donde no haya comentario o string
	for _, rule := range s.Rules {
		if rule.Scope == "comment" || rule.Scope == "string" {
			continue
		}
		matches := rule.Regex.FindAllStringIndex(line, -1)
		for _, match := range matches {
			start, end := match[0], match[1]
			// Solo aplicar si ningún byte en el rango está ya cubierto por string o comment
			alreadyCovered := false
			for i := start; i < end; i++ {
				if byteScopes[i] == "comment" || byteScopes[i] == "string" {
					alreadyCovered = true
					break
				}
			}
			if !alreadyCovered {
				for i := start; i < end; i++ {
					byteScopes[i] = rule.Scope
				}
			}
		}
	}

	// Agrupar scopes continuos en TokenSpans
	var spans []TokenSpan
	start := 0
	for start < len(line) {
		scope := byteScopes[start]
		end := start + 1
		for end < len(line) && byteScopes[end] == scope {
			end++
		}
		if scope != "" {
			spans = append(spans, TokenSpan{
				Start: start,
				End:   end,
				Scope: scope,
			})
		}
		start = end
	}

	return spans
}
