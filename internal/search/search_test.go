package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchWorkspaceBasicAndCase(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	os.WriteFile(file1, []byte("Hello World\nhello gophers\nGOODBYE\n"), 0644)

	file2 := filepath.Join(tmpDir, "file2.go")
	os.WriteFile(file2, []byte("package main\n\nfunc Hello() {}\n"), 0644)

	ctx := context.Background()

	// 1. Búsqueda insensible a mayúsculas
	results, count, err := SearchWorkspace(ctx, tmpDir, SearchOptions{
		Query:         "hello",
		CaseSensitive: false,
	})
	if err != nil {
		t.Fatalf("error en búsqueda: %v", err)
	}

	if count != 3 || len(results) != 2 {
		t.Errorf("se esperaban 3 coincidencias para 'hello' (case insensitive), se obtuvieron %d en %d archivos", count, len(results))
	}

	// 2. Búsqueda sensible a mayúsculas
	resultsExact, countExact, err := SearchWorkspace(ctx, tmpDir, SearchOptions{
		Query:         "Hello",
		CaseSensitive: true,
	})
	if err != nil {
		t.Fatalf("error en búsqueda: %v", err)
	}

	if countExact != 2 {
		t.Errorf("se esperaban 2 coincidencias exactas para 'Hello', se obtuvieron %d (en %d archivos)", countExact, len(resultsExact))
	}
}

func TestSearchWorkspaceRegex(t *testing.T) {
	tmpDir := t.TempDir()

	codeFile := filepath.Join(tmpDir, "sample.go")
	os.WriteFile(codeFile, []byte("var a1 = 10\nvar b2 = 20\nvar test = 30\n"), 0644)

	ctx := context.Background()

	results, count, err := SearchWorkspace(ctx, tmpDir, SearchOptions{
		Query: `var [a-z]\d`,
		Regex: true,
	})
	if err != nil {
		t.Fatalf("error en regex search: %v", err)
	}

	if count != 2 {
		t.Errorf("se esperaban 2 coincidencias con regex, obtenidas %d en %d archivos", count, len(results))
	}
}

func TestSearchWorkspaceFuzzy(t *testing.T) {
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "sample.txt")
	os.WriteFile(file, []byte("drawStringInternal\notherFunction\ndrawSidebarWidget\n"), 0644)

	ctx := context.Background()

	results, count, err := SearchWorkspace(ctx, tmpDir, SearchOptions{
		Query: "ds",
		Fuzzy: true,
	})
	if err != nil {
		t.Fatalf("error en fuzzy search: %v", err)
	}

	if count != 2 {
		t.Errorf("se esperaban 2 coincidencias fuzzy ('drawString' y 'drawSidebar'), obtenidas %d en %d archivos", count, len(results))
	}
}

func TestSearchWorkspaceIgnoreAndBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Carpeta ignorada (.git)
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte("secret_keyword"), 0644)

	// Archivo binario con byte nulo 0x00
	binFile := filepath.Join(tmpDir, "sample.bin")
	os.WriteFile(binFile, []byte{'s', 'e', 'c', 'r', 'e', 't', 0x00, 'k', 'e', 'y'}, 0644)

	// Archivo de texto válido
	validFile := filepath.Join(tmpDir, "valid.txt")
	os.WriteFile(validFile, []byte("this has secret_keyword inside\n"), 0644)

	ctx := context.Background()

	results, count, err := SearchWorkspace(ctx, tmpDir, SearchOptions{
		Query: "secret_keyword",
	})
	if err != nil {
		t.Fatalf("error en búsqueda: %v", err)
	}

	if count != 1 {
		t.Errorf("se esperaba solo 1 coincidencia en archivo válido (ignorando .git y binarios), obtenidas %d", count)
	}
	if len(results) != 1 || results[0].RelPath != "valid.txt" {
		t.Errorf("archivo coincidente inesperado: %+v", results)
	}
}

func TestSearchWorkspaceCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test query line\n"), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancelar inmediatamente

	_, _, err := SearchWorkspace(ctx, tmpDir, SearchOptions{
		Query: "test",
	})

	if err != nil && err != context.Canceled {
		t.Fatalf("error inesperado tras cancelación: %v", err)
	}
}

func TestSearchWorkspaceMaxLimit(t *testing.T) {
	tmpDir := t.TempDir()
	content := "match line\nmatch line\nmatch line\nmatch line\nmatch line\n"
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte(content), 0644)

	ctx := context.Background()
	_, count, err := SearchWorkspace(ctx, tmpDir, SearchOptions{
		Query:      "match",
		MaxMatches: 3,
	})
	if err != nil {
		t.Fatalf("error en búsqueda limitada: %v", err)
	}

	if count > 3 {
		t.Errorf("se esperaba máximo 3 coincidencias, obtenidas %d", count)
	}
}
