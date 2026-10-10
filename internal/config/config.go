package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"edi/defaults"
)

// EditorConfig contiene las preferencias globales del editor.
type EditorConfig struct {
	Theme                string `json:"theme"`
	TabSize              int    `json:"tab_size"`
	ExpandTabs           bool   `json:"expand_tabs"`
	LineNumbers          bool   `json:"line_numbers"`
	HighlightCurrentLine bool   `json:"highlight_current_line"`
	StatusBar            bool   `json:"status_bar"`
	AutoIndent           bool   `json:"auto_indent"`
	GitIntegration       bool   `json:"git_integration"`
}

// DefaultConfig retorna la configuración predeterminada en memoria.
func DefaultConfig() EditorConfig {
	return EditorConfig{
		Theme:                "dracula",
		TabSize:              4,
		ExpandTabs:           true,
		LineNumbers:          true,
		HighlightCurrentLine: true,
		StatusBar:            true,
		AutoIndent:           true,
		GitIntegration:       true,
	}
}

// StripJSONComments elimina comentarios estilo // y /* */ de un JSON (formato JSONC)
// respetando strings y caracteres escapados.
func StripJSONComments(data []byte) []byte {
	var out bytes.Buffer
	inString := false
	inSingleLineComment := false
	inMultiLineComment := false
	escaped := false

	for i := 0; i < len(data); i++ {
		c := data[i]

		if inSingleLineComment {
			if c == '\n' {
				inSingleLineComment = false
				out.WriteByte(c)
			}
			continue
		}

		if inMultiLineComment {
			if c == '*' && i+1 < len(data) && data[i+1] == '/' {
				inMultiLineComment = false
				i++
			}
			continue
		}

		if inString {
			out.WriteByte(c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			continue
		}

		// Fuera de string y comentarios
		if c == '"' {
			inString = true
			out.WriteByte(c)
			continue
		}

		if c == '/' && i+1 < len(data) {
			if data[i+1] == '/' {
				inSingleLineComment = true
				i++
				continue
			} else if data[i+1] == '*' {
				inMultiLineComment = true
				i++
				continue
			}
		}

		out.WriteByte(c)
	}

	return bytes.TrimSpace(out.Bytes())
}

// FindConfigDir localiza la carpeta de configuración activa siguiendo la jerarquía:
// 1. ./.edi-config (local del proyecto)
// 2. ~/.edi-config (raíz del usuario)
// 3. ~/.config/edi (estándar XDG)
func FindConfigDir() string {
	// 1. Proyecto local
	if stat, err := os.Stat(".edi-config"); err == nil && stat.IsDir() {
		return ".edi-config"
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		// 2. Raíz del usuario ~/.edi-config
		userRoot := filepath.Join(home, ".edi-config")
		if stat, err := os.Stat(userRoot); err == nil && stat.IsDir() {
			return userRoot
		}

		// 3. XDG ~/.config/edi
		xdgRoot := filepath.Join(home, ".config", "edi")
		if stat, err := os.Stat(xdgRoot); err == nil && stat.IsDir() {
			return xdgRoot
		}
	}

	return ""
}

// GetUserConfigDir retorna la ruta canónica en la raíz del usuario (~/.edi-config).
func GetUserConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("no se pudo determinar el directorio home: %w", err)
	}
	return filepath.Join(home, ".edi-config"), nil
}

// InitUserConfigDir inicializa la carpeta ~/.edi-config con subcarpetas y archivos de ejemplo.
func InitUserConfigDir() (string, error) {
	dir, err := GetUserConfigDir()
	if err != nil {
		return "", err
	}

	subdirs := []string{
		filepath.Join(dir, "themes"),
		filepath.Join(dir, "syntax"),
		filepath.Join(dir, "plugins"),
	}

	for _, d := range subdirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", fmt.Errorf("error creando directorio %s: %w", d, err)
		}
	}

	// Copiar archivos predeterminados si no existen
	copyIfNotExists := func(embedPath, destPath string) error {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			data, err := defaults.FS.ReadFile(embedPath)
			if err != nil {
				return err
			}
			return os.WriteFile(destPath, data, 0644)
		}
		return nil
	}

	_ = copyIfNotExists("config.json", filepath.Join(dir, "config.json"))
	_ = copyIfNotExists("keybindings.json", filepath.Join(dir, "keybindings.json"))
	_ = copyIfNotExists("themes/dracula.json", filepath.Join(dir, "themes", "dracula.json"))
	_ = copyIfNotExists("themes/monokai.json", filepath.Join(dir, "themes", "monokai.json"))
	_ = copyIfNotExists("syntax/go.json", filepath.Join(dir, "syntax", "go.json"))
	_ = copyIfNotExists("syntax/python.json", filepath.Join(dir, "syntax", "python.json"))
	_ = copyIfNotExists("syntax/json.json", filepath.Join(dir, "syntax", "json.json"))
	_ = copyIfNotExists("syntax/markdown.json", filepath.Join(dir, "syntax", "markdown.json"))

	return dir, nil
}

// LoadConfig carga la configuración del editor desde la carpeta activa o los defaults embebidos.
func LoadConfig() (*EditorConfig, error) {
	cfg := DefaultConfig()

	var data []byte
	configDir := FindConfigDir()

	if configDir != "" {
		filePath := filepath.Join(configDir, "config.json")
		if content, err := os.ReadFile(filePath); err == nil {
			data = content
		}
	}

	if data == nil {
		content, err := defaults.FS.ReadFile("config.json")
		if err == nil {
			data = content
		}
	}

	if data != nil {
		clean := StripJSONComments(data)
		if err := json.Unmarshal(clean, &cfg); err != nil {
			return &cfg, fmt.Errorf("error parseando config.json: %w", err)
		}
	}

	return &cfg, nil
}

// LoadKeybindings carga el mapeo de teclas desde la carpeta activa o los defaults embebidos.
func LoadKeybindings() (map[string]string, error) {
	bindings := make(map[string]string)

	var data []byte
	configDir := FindConfigDir()

	if configDir != "" {
		filePath := filepath.Join(configDir, "keybindings.json")
		if content, err := os.ReadFile(filePath); err == nil {
			data = content
		}
	}

	if data == nil {
		content, err := defaults.FS.ReadFile("keybindings.json")
		if err == nil {
			data = content
		}
	}

	if data != nil {
		clean := StripJSONComments(data)
		if err := json.Unmarshal(clean, &bindings); err != nil {
			return nil, fmt.Errorf("error parseando keybindings.json: %w", err)
		}
	}

	return bindings, nil
}

// ReadFileFromConfigOrEmbed busca archivos tanto en carpetas de configuración como en los embebidos.
func ReadFileFromConfigOrEmbed(subPath string) ([]byte, error) {
	configDir := FindConfigDir()
	if configDir != "" {
		localPath := filepath.Join(configDir, subPath)
		if content, err := os.ReadFile(localPath); err == nil {
			return content, nil
		}
	}

	return defaults.FS.ReadFile(filepath.ToSlash(subPath))
}

// ListJSONFilesInSubdir lista nombres de archivos JSON en una subcarpeta (ej. "themes", "syntax").
func ListJSONFilesInSubdir(subdir string) []string {
	seen := make(map[string]bool)
	var names []string

	// Revisar en configDir físico
	configDir := FindConfigDir()
	if configDir != "" {
		physicalPath := filepath.Join(configDir, subdir)
		if entries, err := os.ReadDir(physicalPath); err == nil {
			for _, e := range entries {
				if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
					seen[e.Name()] = true
					names = append(names, e.Name())
				}
			}
		}
	}

	// Revisar en embebidos si no han sido sobreescritos
	embedSubdir := filepath.ToSlash(subdir)
	if entries, err := fs.ReadDir(defaults.FS, embedSubdir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".json" && !seen[e.Name()] {
				seen[e.Name()] = true
				names = append(names, e.Name())
			}
		}
	}

	return names
}
