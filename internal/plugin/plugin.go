package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"edi/internal/config"
)

// HookAction describe una acción ejecutada al dispararse un evento del editor.
type HookAction struct {
	Action      string `json:"action"` // "pipe_buffer", "run_shell"
	Command     string `json:"command"`
	Description string `json:"description"`
}

// PluginCommand describe un comando expuesto por un plugin para la paleta de comandos.
type PluginCommand struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Action  string `json:"action"`
	Command string `json:"command"`
}

// Plugin representa la configuración y extensiones declaradas en un archivo JSON de plugin.
type Plugin struct {
	Name      string                  `json:"name"`
	Version   string                  `json:"version"`
	Languages []string                `json:"languages"`
	Hooks     map[string][]HookAction `json:"hooks"`
	Commands  []PluginCommand         `json:"commands"`
}

// LoadAllPlugins carga todos los complementos declarados en carpetas físicas o embebidos.
func LoadAllPlugins() []*Plugin {
	files := config.ListJSONFilesInSubdir("plugins")
	var plugins []*Plugin

	for _, fileName := range files {
		subPath := filepath.Join("plugins", fileName)
		data, err := config.ReadFileFromConfigOrEmbed(subPath)
		if err != nil {
			continue
		}

		cleanData := config.StripJSONComments(data)
		var p Plugin
		if err := json.Unmarshal(cleanData, &p); err != nil {
			continue
		}

		plugins = append(plugins, &p)
	}

	return plugins
}

// MatchesLanguage verifica si el plugin aplica al lenguaje dado.
func (p *Plugin) MatchesLanguage(lang string) bool {
	if len(p.Languages) == 0 {
		return true // Aplica a todos si no especifica
	}
	langLower := strings.ToLower(lang)
	for _, l := range p.Languages {
		if strings.ToLower(l) == langLower {
			return true
		}
	}
	return false
}

// ExecutePipeBuffer pasa el texto actual por stdin a un comando externo (como gofmt o prettier)
// y retorna la salida formateada.
func ExecutePipeBuffer(cmdLine string, input string) (string, error) {
	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		return input, fmt.Errorf("comando vacío")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = strings.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errOut := strings.TrimSpace(stderr.String())
		if errOut != "" {
			return input, fmt.Errorf("%s: %s", err, errOut)
		}
		return input, err
	}

	return stdout.String(), nil
}

// ExecuteShellCommand ejecuta un comando en segundo plano y retorna su salida.
func ExecuteShellCommand(cmdLine string) (string, error) {
	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		return "", fmt.Errorf("comando vacío")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), err
	}
	return strings.TrimSpace(string(out)), nil
}
