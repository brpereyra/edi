package tree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileTreeNavigation(t *testing.T) {
	tmpDir := t.TempDir()

	// Crear estructura de prueba
	subDir := filepath.Join(tmpDir, "src")
	_ = os.Mkdir(subDir, 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(subDir, "main.go"), []byte("package main"), 0644)

	ft, err := NewFileTree(tmpDir)
	if err != nil {
		t.Fatalf("error inicializando árbol: %v", err)
	}

	if len(ft.VisibleRows) != 2 {
		t.Fatalf("se esperaban 2 nodos en raíz (src y README.md), obtenidos %d", len(ft.VisibleRows))
	}

	// El primer nodo debe ser el directorio "src"
	selected := ft.GetSelected()
	if selected == nil || selected.Name != "src" || !selected.IsDir {
		t.Fatalf("se esperaba primer nodo directorio 'src', obtenido: %v", selected)
	}

	// Expandir directorio src
	ft.ToggleCurrent()
	if !selected.Expanded {
		t.Fatalf("se esperaba directorio expandido")
	}

	if len(ft.VisibleRows) != 3 {
		t.Fatalf("se esperaban 3 nodos visibles tras expandir (src, main.go, README.md), obtenidos %d", len(ft.VisibleRows))
	}

	// Mover selección
	ft.MoveDown()
	newSelected := ft.GetSelected()
	if newSelected == nil || newSelected.Name != "main.go" {
		t.Errorf("se esperaba 'main.go', obtenido: %v", newSelected)
	}

	// Crear nuevo archivo y refrescar
	_ = os.WriteFile(filepath.Join(subDir, "helper.go"), []byte("package main"), 0644)
	ft.Refresh()

	// Debería tener 4 filas visibles (src, helper.go, main.go, README.md) manteniendo 'src' expandido
	if len(ft.VisibleRows) != 4 {
		t.Fatalf("se esperaban 4 nodos visibles tras Refresh, obtenidos %d", len(ft.VisibleRows))
	}
}

