package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// FileGitStatus representa el estado de un archivo en Git.
type FileGitStatus int

const (
	StatusNone FileGitStatus = iota
	StatusModified
	StatusUntracked
	StatusAdded
	StatusDeleted
	StatusRenamed
)

// Badge retorna el indicador de 1 carácter para el estado.
func (s FileGitStatus) Badge() string {
	switch s {
	case StatusModified:
		return "M"
	case StatusUntracked:
		return "?"
	case StatusAdded:
		return "+"
	case StatusDeleted:
		return "D"
	case StatusRenamed:
		return "R"
	default:
		return ""
	}
}

// RepoStatus almacena el estado parseado del repositorio Git.
type RepoStatus struct {
	IsRepo         bool
	RootDir        string
	Branch         string
	Files          map[string]FileGitStatus // clave: ruta absoluta limpia
	ModifiedCount  int
	UntrackedCount int
	StagedCount    int
	DeletedCount   int
}

// FindRepoRoot busca el directorio raíz de git hacia arriba buscando la carpeta o archivo .git.
func FindRepoRoot(startPath string) (string, bool) {
	if startPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", false
		}
		startPath = cwd
	}

	abs, err := filepath.Abs(startPath)
	if err != nil {
		abs = startPath
	}

	fi, err := os.Stat(abs)
	if err == nil && !fi.IsDir() {
		abs = filepath.Dir(abs)
	}

	curr := abs
	for {
		gitPath := filepath.Join(curr, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return curr, true
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	return "", false
}

// Manager coordina la detección y actualización de Git de forma segura y concurrente.
type Manager struct {
	mu         sync.RWMutex
	status     *RepoStatus
	workDir    string
	refreshing bool
}

// NewManager inicializa un nuevo gestor de Git.
func NewManager(initialDir string) *Manager {
	m := &Manager{
		workDir: initialDir,
		status: &RepoStatus{
			Files: make(map[string]FileGitStatus),
		},
	}
	m.RefreshSync(initialDir)
	return m
}

// IsRepo indica si la carpeta actual pertenece a un repositorio Git.
func (m *Manager) IsRepo() bool {
	if m == nil {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status != nil && m.status.IsRepo
}

// GetRootDir retorna la raíz del repositorio.
func (m *Manager) GetRootDir() string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status == nil {
		return ""
	}
	return m.status.RootDir
}

// GetBranch retorna el nombre de la rama actual.
func (m *Manager) GetBranch() string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status == nil {
		return ""
	}
	return m.status.Branch
}

// GetFileStatus retorna el estado Git de un archivo por su ruta (relativa o absoluta).
func (m *Manager) GetFileStatus(path string) FileGitStatus {
	if m == nil {
		return StatusNone
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.Clean(abs)

	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status == nil || !m.status.IsRepo {
		return StatusNone
	}

	if st, ok := m.status.Files[abs]; ok {
		return st
	}
	return StatusNone
}

// DirHasChanges indica si un directorio contiene algún archivo modificado o nuevo.
func (m *Manager) DirHasChanges(dirPath string) bool {
	if m == nil {
		return false
	}
	abs, err := filepath.Abs(dirPath)
	if err != nil {
		abs = dirPath
	}
	abs = filepath.Clean(abs)
	prefix := abs + string(filepath.Separator)

	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status == nil || !m.status.IsRepo {
		return false
	}

	for fPath := range m.status.Files {
		if strings.HasPrefix(fPath, prefix) {
			return true
		}
	}
	return false
}

// GetSummary retorna un resumen legible para la barra de estado (ej: "git: main [~2 +1]").
func (m *Manager) GetSummary() string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status == nil || !m.status.IsRepo {
		return ""
	}

	branch := m.status.Branch
	if branch == "" {
		branch = "HEAD"
	}

	totalChanges := m.status.ModifiedCount + m.status.UntrackedCount + m.status.StagedCount + m.status.DeletedCount
	if totalChanges == 0 {
		return fmt.Sprintf("git: %s", branch)
	}

	var parts []string
	if m.status.ModifiedCount > 0 {
		parts = append(parts, fmt.Sprintf("~%d", m.status.ModifiedCount))
	}
	if m.status.UntrackedCount > 0 {
		parts = append(parts, fmt.Sprintf("+%d", m.status.UntrackedCount))
	}
	if m.status.StagedCount > 0 {
		parts = append(parts, fmt.Sprintf("✓%d", m.status.StagedCount))
	}
	if m.status.DeletedCount > 0 {
		parts = append(parts, fmt.Sprintf("-%d", m.status.DeletedCount))
	}

	return fmt.Sprintf("git: %s [%s]", branch, strings.Join(parts, " "))
}

// GetStatusSnapshot retorna una copia del estado actual para consultas complejas.
func (m *Manager) GetStatusSnapshot() *RepoStatus {
	if m == nil {
		return &RepoStatus{Files: make(map[string]FileGitStatus)}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status == nil {
		return &RepoStatus{Files: make(map[string]FileGitStatus)}
	}

	copyFiles := make(map[string]FileGitStatus, len(m.status.Files))
	for k, v := range m.status.Files {
		copyFiles[k] = v
	}

	return &RepoStatus{
		IsRepo:         m.status.IsRepo,
		RootDir:        m.status.RootDir,
		Branch:         m.status.Branch,
		Files:          copyFiles,
		ModifiedCount:  m.status.ModifiedCount,
		UntrackedCount: m.status.UntrackedCount,
		StagedCount:    m.status.StagedCount,
		DeletedCount:   m.status.DeletedCount,
	}
}

// RefreshSync actualiza el estado de Git de forma síncrona.
func (m *Manager) RefreshSync(targetDir string) {
	if targetDir == "" {
		targetDir = m.workDir
	}
	m.workDir = targetDir

	repoRoot, isRepo := FindRepoRoot(targetDir)
	if !isRepo {
		m.mu.Lock()
		m.status = &RepoStatus{
			IsRepo: false,
			Files:  make(map[string]FileGitStatus),
		}
		m.mu.Unlock()
		return
	}

	// Comprobar si git existe en el sistema
	if _, err := exec.LookPath("git"); err != nil {
		m.mu.Lock()
		m.status = &RepoStatus{
			IsRepo: false,
			Files:  make(map[string]FileGitStatus),
		}
		m.mu.Unlock()
		return
	}

	status := queryGitStatus(repoRoot)

	m.mu.Lock()
	m.status = status
	m.mu.Unlock()
}

// RefreshAsync actualiza el estado de Git en segundo plano en una goroutine sin bloquear el editor.
func (m *Manager) RefreshAsync(targetDir string, onComplete func()) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.refreshing {
		m.mu.Unlock()
		return
	}
	m.refreshing = true
	m.mu.Unlock()

	go func() {
		defer func() {
			m.mu.Lock()
			m.refreshing = false
			m.mu.Unlock()
			if onComplete != nil {
				onComplete()
			}
		}()

		m.RefreshSync(targetDir)
	}()
}

// queryGitStatus consulta rama y status porcela de Git.
func queryGitStatus(repoRoot string) *RepoStatus {
	res := &RepoStatus{
		IsRepo:  true,
		RootDir: repoRoot,
		Files:   make(map[string]FileGitStatus),
	}

	// 1. Obtener rama
	cmdBranch := exec.Command("git", "branch", "--show-current")
	cmdBranch.Dir = repoRoot
	if out, err := cmdBranch.Output(); err == nil && len(bytes.TrimSpace(out)) > 0 {
		res.Branch = string(bytes.TrimSpace(out))
	} else {
		// Fallback para detached HEAD
		cmdTag := exec.Command("git", "describe", "--tags", "--always")
		cmdTag.Dir = repoRoot
		if outTag, errTag := cmdTag.Output(); errTag == nil && len(bytes.TrimSpace(outTag)) > 0 {
			res.Branch = string(bytes.TrimSpace(outTag))
		} else {
			res.Branch = "HEAD"
		}
	}

	// 2. Obtener estado porcelain
	cmdStatus := exec.Command("git", "status", "--porcelain=v1", "-uall")
	cmdStatus.Dir = repoRoot
	outStatus, err := cmdStatus.Output()
	if err != nil {
		return res
	}

	lines := strings.Split(string(outStatus), "\n")
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}

		indexSt := line[0]
		workSt := line[1]
		filePath := strings.TrimSpace(line[3:])

		// Manejo de renombrados: "old.txt -> new.txt"
		if strings.Contains(filePath, " -> ") {
			parts := strings.Split(filePath, " -> ")
			filePath = strings.TrimSpace(parts[len(parts)-1])
		}
		// Quitar comillas si el archivo tiene espacios
		filePath = strings.Trim(filePath, "\"")

		absPath := filepath.Join(repoRoot, filePath)
		absClean := filepath.Clean(absPath)

		if indexSt == '?' && workSt == '?' {
			res.Files[absClean] = StatusUntracked
			res.UntrackedCount++
		} else if indexSt == 'A' || workSt == 'A' {
			res.Files[absClean] = StatusAdded
			res.StagedCount++
		} else if indexSt == 'D' || workSt == 'D' {
			res.Files[absClean] = StatusDeleted
			res.DeletedCount++
		} else if indexSt == 'R' || workSt == 'R' {
			res.Files[absClean] = StatusRenamed
			res.StagedCount++
		} else if indexSt == 'M' || workSt == 'M' {
			res.Files[absClean] = StatusModified
			res.ModifiedCount++
		} else {
			res.Files[absClean] = StatusModified
			res.ModifiedCount++
		}
	}

	return res
}

// GetFileDiff obtiene el diff en formato texto de un archivo específico frente a HEAD.
func (m *Manager) GetFileDiff(filePath string) (string, error) {
	if m == nil || !m.IsRepo() {
		return "", fmt.Errorf("no es un repositorio git")
	}

	rel, err := filepath.Rel(m.GetRootDir(), filePath)
	if err != nil {
		rel = filePath
	}

	cmd := exec.Command("git", "diff", "HEAD", "--", rel)
	cmd.Dir = m.GetRootDir()
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		// Intentar diff sin HEAD si es un archivo nuevo sin commit previo
		cmdUntracked := exec.Command("git", "diff", "--", rel)
		cmdUntracked.Dir = m.GetRootDir()
		out, err = cmdUntracked.Output()
	}

	return string(out), err
}

// GetProjectDiff obtiene el diff completo de todo el repositorio frente a HEAD.
func (m *Manager) GetProjectDiff() (string, error) {
	if m == nil || !m.IsRepo() {
		return "", fmt.Errorf("no es un repositorio git")
	}

	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = m.GetRootDir()
	out, err := cmd.Output()
	return string(out), err
}
