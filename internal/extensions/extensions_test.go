package extensions

import (
	"os"
	"path/filepath"
	"testing"

	"edi/internal/config"
)

func TestListAllExtensions(t *testing.T) {
	items := ListAllExtensions()
	if len(items) == 0 {
		t.Fatalf("se esperaban extensiones cargadas, se obtuvieron 0")
	}

	foundRust := false
	foundPrettier := false
	for _, it := range items {
		if it.Name == "Rust" && it.Type == "sintaxis" {
			foundRust = true
		}
		if it.Name == "Prettier Formatter" && it.Type == "plugin" {
			foundPrettier = true
		}
	}

	if !foundRust {
		t.Errorf("no se encontro sintaxis Rust en ListAllExtensions")
	}
	if !foundPrettier {
		t.Errorf("no se encontro plugin Prettier en ListAllExtensions")
	}
}

func TestInstallFromLocalFile(t *testing.T) {
	tmpDir := t.TempDir()
	sampleSyntax := filepath.Join(tmpDir, "sample.json")
	content := `{
		"name": "SampleLang",
		"extensions": [".sample"],
		"rules": [
			{"scope": "keyword", "pattern": "\\bfoo\\b"}
		]
	}`

	if err := os.WriteFile(sampleSyntax, []byte(content), 0644); err != nil {
		t.Fatalf("error escribiendo sample: %v", err)
	}

	userDir, _ := config.GetUserConfigDir()
	installedPath := filepath.Join(userDir, "syntax", "sample.json")
	defer os.Remove(installedPath)

	// Probar instalar archivo local
	err := InstallFromSource(sampleSyntax)
	if err != nil {
		t.Fatalf("error instalando sintaxis valida: %v", err)
	}
}
