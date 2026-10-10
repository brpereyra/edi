package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindRepoRoot(t *testing.T) {
	// Probar directorio actual (que sabemos es un repo git)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("error obteniendo cwd: %v", err)
	}

	root, isRepo := FindRepoRoot(cwd)
	if !isRepo {
		t.Fatalf("se esperaba que '%s' sea detectado como repositorio Git", cwd)
	}
	if root == "" {
		t.Fatalf("se esperaba ruta no vacía para root")
	}

	// Probar subdirectorio dentro del repo
	subDir := filepath.Join(cwd, "internal")
	rootSub, isRepoSub := FindRepoRoot(subDir)
	if !isRepoSub || rootSub != root {
		t.Errorf("se esperaba mismo root desde subdirectorio, obtenido %s (esperado %s)", rootSub, root)
	}

	// Probar carpeta temporal sin Git
	tmpDir := t.TempDir()
	_, isRepoTmp := FindRepoRoot(tmpDir)
	if isRepoTmp {
		t.Errorf("directorio temporal no debería ser repo git")
	}
}

func TestGitManagerWithRealRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Inicializar un repo git temporal
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Skip("git no disponible o error inicializando git de prueba")
	}

	// Configurar user.name y user.email temporal para git
	_ = exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
	_ = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()

	mgr := NewManager(tmpDir)
	if !mgr.IsRepo() {
		t.Fatalf("se esperaba IsRepo == true para repo temporal inicializado")
	}

	// Crear archivo untracked
	testFile := filepath.Join(tmpDir, "hola.txt")
	_ = os.WriteFile(testFile, []byte("contenido"), 0644)

	mgr.RefreshSync(tmpDir)
	st := mgr.GetFileStatus(testFile)
	if st != StatusUntracked {
		t.Errorf("se esperaba StatusUntracked para archivo nuevo, obtenido: %v", st)
	}
	if st.Badge() != "?" {
		t.Errorf("badge esperado '?', obtenido '%s'", st.Badge())
	}

	if !mgr.DirHasChanges(tmpDir) {
		t.Errorf("DirHasChanges debería ser true para la carpeta con archivo nuevo")
	}

	// Añadir y hacer commit
	_ = exec.Command("git", "-C", tmpDir, "add", "hola.txt").Run()
	_ = exec.Command("git", "-C", tmpDir, "commit", "-m", "primer commit").Run()

	mgr.RefreshSync(tmpDir)
	stClean := mgr.GetFileStatus(testFile)
	if stClean != StatusNone {
		t.Errorf("se esperaba StatusNone tras commit, obtenido %v", stClean)
	}

	// Modificar archivo
	_ = os.WriteFile(testFile, []byte("contenido modificado"), 0644)
	mgr.RefreshSync(tmpDir)

	stMod := mgr.GetFileStatus(testFile)
	if stMod != StatusModified {
		t.Errorf("se esperaba StatusModified tras modificación, obtenido %v", stMod)
	}
	if stMod.Badge() != "M" {
		t.Errorf("badge esperado 'M', obtenido '%s'", stMod.Badge())
	}

	summary := mgr.GetSummary()
	if summary == "" {
		t.Errorf("summary no debería estar vacío")
	}
}

func TestGitManagerNonRepoGraceful(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	if mgr.IsRepo() {
		t.Errorf("directorio temporal vacío no debe ser repo")
	}
	if mgr.GetFileStatus(filepath.Join(tmpDir, "foo.txt")) != StatusNone {
		t.Errorf("status debe ser StatusNone fuera de un repo")
	}
	if mgr.DirHasChanges(tmpDir) {
		t.Errorf("DirHasChanges debe ser false fuera de repo")
	}
	if mgr.GetSummary() != "" {
		t.Errorf("GetSummary debe ser vacío fuera de repo")
	}
}
