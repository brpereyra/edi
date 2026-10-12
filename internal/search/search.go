package search

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// SearchOptions configura los parámetros para la búsqueda en archivos.
type SearchOptions struct {
	Query         string
	CaseSensitive bool
	Regex         bool
	Fuzzy         bool
	MaxMatches    int
}

// MatchLocation describe la ubicación de una coincidencia dentro de una línea.
type MatchLocation struct {
	LineNum  int    // 1-indexed
	ColStart int    // 0-indexed en runas
	ColEnd   int    // 0-indexed en runas
	LineText string // Texto de la línea recortado
}

// FileSearchResult contiene todas las coincidencias encontradas dentro de un archivo.
type FileSearchResult struct {
	Path     string          // Ruta absoluta
	RelPath  string          // Ruta relativa al workspace
	Matches  []MatchLocation // Coincidencias en el archivo
}

// IgnoredDirs lista carpetas que no deben escanearse.
var IgnoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"build":        true,
	"bin":          true,
	"dist":         true,
	"target":       true,
	".edi-config":  true,
	".cache":       true,
	".idea":        true,
	".vscode":      true,
}

// isBinaryFile determina si un archivo es binario inspeccionando los primeros bytes.
func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return true
	}

	return bytes.IndexByte(buf[:n], 0) != -1
}

// fuzzyMatch verifica si pattern es una subsecuencia de s y retorna las posiciones aproximadas.
func fuzzyMatch(s, pattern string) (int, int, bool) {
	if pattern == "" {
		return 0, 0, true
	}
	sRunes := []rune(s)
	patRunes := []rune(pattern)

	pIdx := 0
	start := -1
	for sIdx, r := range sRunes {
		if r == patRunes[pIdx] {
			if start == -1 {
				start = sIdx
			}
			pIdx++
			if pIdx == len(patRunes) {
				return start, sIdx + 1, true
			}
		}
	}
	return -1, -1, false
}

// SearchWorkspace busca ocurrencias de opts.Query en todos los archivos de rootDir.
func SearchWorkspace(ctx context.Context, rootDir string, opts SearchOptions) ([]FileSearchResult, int, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return nil, 0, nil
	}

	if opts.MaxMatches <= 0 {
		opts.MaxMatches = 1000
	}

	// Precompilar regex si corresponde
	var regexMatcher *regexp.Regexp
	if opts.Regex {
		pattern := opts.Query
		if !opts.CaseSensitive {
			pattern = "(?i)" + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, 0, fmt.Errorf("expresión regular inválida: %w", err)
		}
		regexMatcher = re
	}

	// 1. Recolectar lista de archivos de texto válidos
	var filesToSearch []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			name := info.Name()
			if IgnoredDirs[name] || (strings.HasPrefix(name, ".") && name != ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Omitir archivos mayores a 5MB
		if info.Size() > 5*1024*1024 {
			return nil
		}

		filesToSearch = append(filesToSearch, path)
		return nil
	})

	if err != nil && err != context.Canceled {
		return nil, 0, err
	}

	// 2. Procesar archivos concurrentemente
	type fileJob struct {
		absPath string
		relPath string
	}

	type fileResult struct {
		res FileSearchResult
	}

	jobs := make(chan fileJob, 100)
	results := make(chan fileResult, 100)
	var wg sync.WaitGroup

	numWorkers := 8
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if isBinaryFile(job.absPath) {
					continue
				}

				matches := searchInFile(ctx, job.absPath, opts, regexMatcher)
				if len(matches) > 0 {
					results <- fileResult{
						res: FileSearchResult{
							Path:    job.absPath,
							RelPath: job.relPath,
							Matches: matches,
						},
					}
				}
			}
		}()
	}

	// Despachar trabajos en goroutine
	go func() {
		for _, f := range filesToSearch {
			select {
			case <-ctx.Done():
				break
			default:
				rel, err := filepath.Rel(rootDir, f)
				if err != nil {
					rel = f
				}
				jobs <- fileJob{absPath: f, relPath: rel}
			}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var allResults []FileSearchResult
	totalMatches := 0

	for r := range results {
		if totalMatches >= opts.MaxMatches {
			continue
		}

		remaining := opts.MaxMatches - totalMatches
		if len(r.res.Matches) > remaining {
			r.res.Matches = r.res.Matches[:remaining]
		}
		totalMatches += len(r.res.Matches)
		allResults = append(allResults, r.res)
	}

	return allResults, totalMatches, nil
}

// searchInFile busca coincidencias línea por línea dentro de un archivo específico.
func searchInFile(ctx context.Context, path string, opts SearchOptions, regexMatcher *regexp.Regexp) []MatchLocation {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var matches []MatchLocation
	scanner := bufio.NewScanner(f)
	lineNum := 1

	targetQuery := opts.Query
	if !opts.CaseSensitive && !opts.Regex && !opts.Fuzzy {
		targetQuery = strings.ToLower(opts.Query)
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return matches
		default:
		}

		line := scanner.Text()
		lineRunes := []rune(line)

		if opts.Regex && regexMatcher != nil {
			locs := regexMatcher.FindAllStringIndex(line, -1)
			for _, loc := range locs {
				startByte, endByte := loc[0], loc[1]
				colStart := len([]rune(line[:startByte]))
				colEnd := len([]rune(line[:endByte]))

				matches = append(matches, MatchLocation{
					LineNum:  lineNum,
					ColStart: colStart,
					ColEnd:   colEnd,
					LineText: line,
				})
			}
		} else if opts.Fuzzy {
			lineCmp := line
			queryCmp := opts.Query
			if !opts.CaseSensitive {
				lineCmp = strings.ToLower(line)
				queryCmp = strings.ToLower(opts.Query)
			}
			if start, end, ok := fuzzyMatch(lineCmp, queryCmp); ok {
				matches = append(matches, MatchLocation{
					LineNum:  lineNum,
					ColStart: start,
					ColEnd:   end,
					LineText: line,
				})
			}
		} else {
			// Subcadena exacta
			lineCmp := line
			if !opts.CaseSensitive {
				lineCmp = strings.ToLower(line)
			}

			startPos := 0
			for {
				idx := strings.Index(lineCmp[startPos:], targetQuery)
				if idx == -1 {
					break
				}
				matchByteStart := startPos + idx
				matchByteEnd := matchByteStart + len(targetQuery)

				colStart := len([]rune(line[:matchByteStart]))
				colEnd := colStart + len([]rune(opts.Query))
				if colEnd > len(lineRunes) {
					colEnd = len(lineRunes)
				}

				matches = append(matches, MatchLocation{
					LineNum:  lineNum,
					ColStart: colStart,
					ColEnd:   colEnd,
					LineText: line,
				})

				startPos = matchByteEnd
				if startPos >= len(lineCmp) {
					break
				}
			}
		}

		lineNum++
	}

	return matches
}
