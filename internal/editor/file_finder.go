package editor

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ScanWorkspaceFiles descubre archivos en el directorio actual excluyendo carpetas ignoradas.
func ScanWorkspaceFiles(maxFiles int) []string {
	var files []string
	ignoreDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		".idea":        true,
		".vscode":      true,
	}

	_ = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if ignoreDirs[name] {
				return filepath.SkipDir
			}
			if strings.HasPrefix(name, ".") && name != "." && name != ".edi-config" {
				return filepath.SkipDir
			}
			return nil
		}

		// Filtrar binarios o temporales conocidos
		ext := filepath.Ext(path)
		if ext == ".exe" || ext == ".test" || ext == ".DS_Store" {
			return nil
		}

		// Normalizar ruta relativa limpia
		cleanPath := filepath.Clean(path)
		files = append(files, cleanPath)
		if len(files) >= maxFiles {
			return fs.SkipAll
		}
		return nil
	})

	return files
}

// FuzzyMatch verifica si una consulta coincide difusamente o como subcadena en la ruta del archivo.
func FuzzyMatch(path, query string) bool {
	if query == "" {
		return true
	}
	pathLower := strings.ToLower(path)
	queryLower := strings.ToLower(query)

	// Búsqueda simple de subcadena (muy rápida y natural)
	if strings.Contains(pathLower, queryLower) {
		return true
	}

	// Búsqueda difusa de caracteres en orden
	qIdx := 0
	qRunes := []rune(queryLower)
	for _, ch := range pathLower {
		if ch == qRunes[qIdx] {
			qIdx++
			if qIdx == len(qRunes) {
				return true
			}
		}
	}

	return false
}

// FilterFiles retorna los archivos que coinciden con la consulta.
func FilterFiles(files []string, query string) []string {
	if query == "" {
		return files
	}
	var matched []string
	for _, f := range files {
		if FuzzyMatch(f, query) {
			matched = append(matched, f)
		}
	}
	return matched
}

// IsBinaryFile realiza una comprobación rápida leyendo los primeros bytes.
func IsBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return false
	}

	// Si contiene byte nulo (0x00), generalmente es un archivo binario
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}
