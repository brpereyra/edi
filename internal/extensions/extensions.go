package extensions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"edi/internal/config"
	"edi/internal/plugin"
	"edi/internal/syntax"
)

// ExtensionItem describe una sintaxis o plugin disponible en el sistema.
type ExtensionItem struct {
	Name        string
	Type        string // "sintaxis" o "plugin"
	Source      string // "embebido" o "usuario"
	Description string
	Extensions  []string
}

// ListAllExtensions retorna la lista completa de sintaxis y plugins activos.
func ListAllExtensions() []ExtensionItem {
	var items []ExtensionItem

	// 1. Sintaxis
	syntaxes := syntax.LoadAllSyntaxes()
	for _, s := range syntaxes {
		items = append(items, ExtensionItem{
			Name:        s.Name,
			Type:        "sintaxis",
			Source:      "activo",
			Description: fmt.Sprintf("Extensiones: %s", strings.Join(s.Extensions, ", ")),
			Extensions:  s.Extensions,
		})
	}

	// 2. Plugins
	plugins := plugin.LoadAllPlugins()
	for _, p := range plugins {
		langs := "todos"
		if len(p.Languages) > 0 {
			langs = strings.Join(p.Languages, ", ")
		}
		items = append(items, ExtensionItem{
			Name:        p.Name,
			Type:        "plugin",
			Source:      "activo",
			Description: fmt.Sprintf("v%s [lenguajes: %s]", p.Version, langs),
		})
	}

	return items
}

// InstallFromSource descarga o copia una definición de sintaxis o plugin a ~/.edi-config/
func InstallFromSource(source string) error {
	var data []byte
	var fileName string

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(source)
		if err != nil {
			return fmt.Errorf("error descargando extension: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("el servidor retorno codigo HTTP %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error leyendo contenido: %w", err)
		}
		data = body
		fileName = filepath.Base(source)
	} else {
		// Archivo local
		content, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("error leyendo archivo local: %w", err)
		}
		data = content
		fileName = filepath.Base(source)
	}

	if !strings.HasSuffix(strings.ToLower(fileName), ".json") {
		fileName += ".json"
	}

	cleanData := config.StripJSONComments(data)

	// Intentar validar como sintaxis
	var rawSyn syntax.RawSyntax
	if err := json.Unmarshal(cleanData, &rawSyn); err == nil && len(rawSyn.Rules) > 0 {
		// Validar que las expresiones regulares compilen
		for _, rule := range rawSyn.Rules {
			if _, reErr := regexp.Compile(rule.Pattern); reErr != nil {
				return fmt.Errorf("regla regex invalida en '%s': %w", rule.Scope, reErr)
			}
		}

		userDir, err := config.GetUserConfigDir()
		if err != nil {
			return err
		}
		targetDir := filepath.Join(userDir, "syntax")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}

		destPath := filepath.Join(targetDir, fileName)
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return err
		}
		fmt.Printf("Sintaxis '%s' instalada correctamente en:\n  %s\n", rawSyn.Name, destPath)
		return nil
	}

	// Intentar validar como plugin
	var rawPlug plugin.Plugin
	if err := json.Unmarshal(cleanData, &rawPlug); err == nil && (len(rawPlug.Hooks) > 0 || len(rawPlug.Commands) > 0) {
		userDir, err := config.GetUserConfigDir()
		if err != nil {
			return err
		}
		targetDir := filepath.Join(userDir, "plugins")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}

		destPath := filepath.Join(targetDir, fileName)
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return err
		}
		fmt.Printf("Plugin '%s' instalado correctamente en:\n  %s\n", rawPlug.Name, destPath)
		return nil
	}

	return fmt.Errorf("el archivo no contiene una definicion valida de sintaxis ni de plugin para edi")
}
