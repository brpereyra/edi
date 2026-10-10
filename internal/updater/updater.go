package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"edi/internal/version"
)

const (
	DefaultRepo = "brpereyra/edi"
	userAgent   = "edi-updater"
)

// ReleaseAsset representa un archivo adjunto a una versión en GitHub.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// ReleaseInfo contiene los metadatos de una versión en GitHub.
type ReleaseInfo struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	PublishedAt string         `json:"published_at"`
	HTMLURL     string         `json:"html_url"`
	Body        string         `json:"body"`
	Assets      []ReleaseAsset `json:"assets"`
}

// CheckLatestRelease consulta GitHub API para obtener la versión más reciente.
func CheckLatestRelease(repo string) (*ReleaseInfo, error) {
	if repo == "" {
		repo = DefaultRepo
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	client := &http.Client{Timeout: 12 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando solicitud: %w", err)
	}

	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", userAgent, version.Version))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error conectando con GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no se encontraron versiones publicadas en %s", repo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API retornó código HTTP %d", resp.StatusCode)
	}

	var rel ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("error deserializando respuesta de GitHub: %w", err)
	}

	return &rel, nil
}

// IsNewer compara semánticamente dos versiones (ej: "v0.2.0" y "v0.1.0").
// Retorna true si targetVersion es más reciente que currentVersion.
func IsNewer(targetVersion, currentVersion string) bool {
	tNorm := strings.TrimPrefix(strings.TrimSpace(targetVersion), "v")
	cNorm := strings.TrimPrefix(strings.TrimSpace(currentVersion), "v")

	tParts := strings.Split(tNorm, ".")
	cParts := strings.Split(cNorm, ".")

	maxLen := len(tParts)
	if len(cParts) > maxLen {
		maxLen = len(cParts)
	}

	for i := 0; i < maxLen; i++ {
		tNum := 0
		cNum := 0

		if i < len(tParts) {
			tNum, _ = strconv.Atoi(tParts[i])
		}
		if i < len(cParts) {
			cNum, _ = strconv.Atoi(cParts[i])
		}

		if tNum > cNum {
			return true
		}
		if tNum < cNum {
			return false
		}
	}

	return false
}

// FindMatchingAsset encuentra el nombre de binario esperado para la arquitectura actual.
func FindMatchingAsset(assets []ReleaseAsset, goos, goarch string) (string, error) {
	expectedName := fmt.Sprintf("edi-%s-%s", goos, goarch)
	if goos == "windows" {
		expectedName += ".exe"
	}

	for _, a := range assets {
		if strings.EqualFold(a.Name, expectedName) {
			return a.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no se encontró binario para %s/%s (esperado: %s)", goos, goarch, expectedName)
}

// DownloadFile descarga una URL a un archivo destino.
func DownloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 45 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", userAgent, version.Version))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("código HTTP %d al descargar binario", resp.StatusCode)
	}

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// ReplaceBinary reemplaza atómicamente el ejecutable actual con el nuevo binario descargado.
// Preserva intacto el directorio de configuración del usuario (~/.edi-config/).
func ReplaceBinary(newBinaryPath string) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("error localizando ejecutable actual: %w", err)
	}

	realDest, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realDest = execPath
	}

	// Verificar permisos de escritura en el directorio del ejecutable
	destDir := filepath.Dir(realDest)
	testFile := filepath.Join(destDir, fmt.Sprintf(".edi_perm_test_%d", time.Now().UnixNano()))
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		return realDest, fmt.Errorf("permiso denegado para escribir en %s (intenta ejecutar con sudo o instalar en ~/.local/bin)", destDir)
	}
	_ = os.Remove(testFile)

	// En sistemas Unix, podemos renombrar el ejecutable en ejecución a .old
	// y colocar el nuevo en su lugar.
	oldBackup := realDest + ".old"
	_ = os.Remove(oldBackup)

	if err := os.Rename(realDest, oldBackup); err != nil {
		return realDest, fmt.Errorf("error respaldando ejecutable actual: %w", err)
	}

	if err := copyAndMakeExecutable(newBinaryPath, realDest); err != nil {
		// Rollback si falla
		_ = os.Rename(oldBackup, realDest)
		return realDest, fmt.Errorf("error instalando nuevo binario: %w", err)
	}

	// Limpiar archivo antiguo
	_ = os.Remove(oldBackup)
	return realDest, nil
}

func copyAndMakeExecutable(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return os.Chmod(dest, 0755)
}

// RunCLIUpdate ejecuta el flujo de actualización interactivo en la consola.
// IMPORTANTE: Solo actualiza el ejecutable de edi. Tus configuraciones,
// temas y atajos en ~/.edi-config/ se mantienen 100% intactos.
func RunCLIUpdate(force bool) error {
	fmt.Printf("edi - Verificando actualizaciones...\n")
	fmt.Printf("Versión actual: %s (%s/%s)\n", version.Version, runtime.GOOS, runtime.GOARCH)

	rel, err := CheckLatestRelease(DefaultRepo)
	if err != nil {
		// Si no hay releases publicados aún, podemos ofrecer compilar desde Go si está disponible
		fmt.Printf("No se pudo obtener información de releases en GitHub: %v\n", err)
		if hasGoCompiler() {
			fmt.Printf("Detectado compilador Go. ¿Deseas actualizar compilando la última versión del repositorio? (s/N): ")
			var ans string
			fmt.Scanln(&ans)
			if strings.ToLower(strings.TrimSpace(ans)) == "s" {
				return updateFromGoInstall()
			}
		}
		return nil
	}

	fmt.Printf("Última versión disponible: %s\n", rel.TagName)

	isNew := IsNewer(rel.TagName, version.Version)
	if !isNew && !force {
		fmt.Printf("✓ edi ya está en la versión más reciente (%s). No se requieren cambios.\n", version.Version)
		return nil
	}

	if force && !isNew {
		fmt.Printf("Forzando reinstalación de la versión %s...\n", rel.TagName)
	} else {
		fmt.Printf("¡Nueva versión encontrada! Descargando actualización a %s...\n", rel.TagName)
	}

	downloadURL, err := FindMatchingAsset(rel.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Printf("Advertencia: %v\n", err)
		if hasGoCompiler() {
			fmt.Printf("Compilando actualización mediante Go...\n")
			return updateFromGoInstall()
		}
		return fmt.Errorf("no se encontró binario para tu plataforma y no hay compilador Go instalado")
	}

	// Descargar binario en directorio temporal
	tmpDir, err := os.MkdirTemp("", "edi-update-*")
	if err != nil {
		return fmt.Errorf("error creando directorio temporal: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpBinary := filepath.Join(tmpDir, "edi-new")
	fmt.Printf("Descargando binario desde GitHub Releases...\n")
	if err := DownloadFile(downloadURL, tmpBinary); err != nil {
		return fmt.Errorf("error descargando actualización: %w", err)
	}

	destPath, err := ReplaceBinary(tmpBinary)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("✓ edi se ha actualizado exitosamente a %s\n", rel.TagName)
	fmt.Printf("  Ubicación del ejecutable: %s\n", destPath)
	fmt.Printf("  Nota: Tus configuraciones en ~/.edi-config/ se mantuvieron intactas.\n")
	return nil
}

func hasGoCompiler() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

func updateFromGoInstall() error {
	fmt.Printf("Ejecutando: go install github.com/%s/cmd/edi@latest ...\n", DefaultRepo)
	cmd := exec.Command("go", "install", fmt.Sprintf("github.com/%s/cmd/edi@latest", DefaultRepo))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error ejecutando go install: %w", err)
	}

	fmt.Printf("✓ edi actualizado exitosamente mediante Go.\n")
	fmt.Printf("  Tus configuraciones en ~/.edi-config/ se mantuvieron intactas.\n")
	return nil
}
