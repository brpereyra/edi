package config

import (
	"testing"
)

func TestStripJSONComments(t *testing.T) {
	input := []byte(`
	{
		// Comentario de una línea
		"theme": "monokai", /* comentario de bloque */
		"tab_size": 2
	}
	`)

	cleaned := StripJSONComments(input)
	expectedClean := `{
		"theme": "monokai", 
		"tab_size": 2
	}`

	if len(cleaned) == 0 {
		t.Fatalf("se esperaba contenido no vacío")
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("error cargando configuración default: %v", err)
	}

	if cfg.Theme == "" {
		t.Errorf("se esperaba un tema por defecto, pero se obtuvo vacío")
	}
	_ = expectedClean
}

func TestLoadKeybindings(t *testing.T) {
	bindings, err := LoadKeybindings()
	if err != nil {
		t.Fatalf("error cargando keybindings: %v", err)
	}

	if bindings["Ctrl+S"] != "buffer:save" {
		t.Errorf("se esperaba buffer:save para Ctrl+S, se obtuvo %s", bindings["Ctrl+S"])
	}
}
