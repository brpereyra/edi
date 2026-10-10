package plugin

import (
	"testing"
)

func TestLoadAllPlugins(t *testing.T) {
	plugins := LoadAllPlugins()
	if len(plugins) == 0 {
		t.Fatalf("se esperaba al menos 1 plugin cargado (go-tools), se obtuvieron 0")
	}

	foundGoTools := false
	for _, p := range plugins {
		if p.Name == "Go Tools" {
			foundGoTools = true
			if !p.MatchesLanguage("go") {
				t.Errorf("se esperaba que go-tools coincida con 'go'")
			}
			if len(p.Hooks["on_save"]) == 0 {
				t.Errorf("se esperaba gancho on_save en go-tools")
			}
		}
	}

	if !foundGoTools {
		t.Errorf("no se encontró plugin Go Tools")
	}
}

func TestExecutePipeBuffer(t *testing.T) {
	// Probar tubería con comando echo o cat
	output, err := ExecutePipeBuffer("cat", "hola mundo")
	if err != nil {
		t.Fatalf("error ejecutando pipe buffer con cat: %v", err)
	}

	if output != "hola mundo" {
		t.Errorf("se esperaba 'hola mundo', se obtuvo '%s'", output)
	}
}
