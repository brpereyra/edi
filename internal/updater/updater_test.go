package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		target   string
		current  string
		expected bool
	}{
		{"v0.2.0", "v0.1.0", true},
		{"v0.1.1", "v0.1.0", true},
		{"0.1.1", "0.1.0", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.0.9", "v0.1.0", false},
		{"v0.1.0", "v0.2.0", false},
		{"v0.1.0-beta", "v0.1.0", false},
	}

	for _, tt := range tests {
		got := IsNewer(tt.target, tt.current)
		if got != tt.expected {
			t.Errorf("IsNewer(%q, %q) = %v; esperado %v", tt.target, tt.current, got, tt.expected)
		}
	}
}

func TestFindMatchingAsset(t *testing.T) {
	assets := []ReleaseAsset{
		{Name: "edi-linux-amd64", BrowserDownloadURL: "https://example.com/linux-amd64"},
		{Name: "edi-linux-arm64", BrowserDownloadURL: "https://example.com/linux-arm64"},
		{Name: "edi-darwin-amd64", BrowserDownloadURL: "https://example.com/darwin-amd64"},
		{Name: "edi-darwin-arm64", BrowserDownloadURL: "https://example.com/darwin-arm64"},
		{Name: "edi-windows-amd64.exe", BrowserDownloadURL: "https://example.com/windows-amd64.exe"},
	}

	url, err := FindMatchingAsset(assets, "darwin", "arm64")
	if err != nil {
		t.Fatalf("error buscando asset darwin/arm64: %v", err)
	}
	if url != "https://example.com/darwin-arm64" {
		t.Errorf("url inesperada: %s", url)
	}

	urlWin, err := FindMatchingAsset(assets, "windows", "amd64")
	if err != nil {
		t.Fatalf("error buscando asset windows/amd64: %v", err)
	}
	if urlWin != "https://example.com/windows-amd64.exe" {
		t.Errorf("url windows inesperada: %s", urlWin)
	}

	_, errNotFound := FindMatchingAsset(assets, "freebsd", "riscv64")
	if errNotFound == nil {
		t.Errorf("se esperaba error para plataforma inexistente")
	}
}

func TestMockCheckLatestRelease(t *testing.T) {
	mockRel := ReleaseInfo{
		TagName: "v0.2.0",
		Name:    "edi Release v0.2.0",
		Assets: []ReleaseAsset{
			{Name: "edi-darwin-arm64", BrowserDownloadURL: "https://example.com/download"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockRel)
	}))
	defer server.Close()

	// Probar decodificación de ReleaseInfo
	res, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("error en request mock: %v", err)
	}
	defer res.Body.Close()

	var rel ReleaseInfo
	if err := json.NewDecoder(res.Body).Decode(&rel); err != nil {
		t.Fatalf("error decodificando json: %v", err)
	}

	if rel.TagName != "v0.2.0" {
		t.Errorf("tag esperado v0.2.0, obtenido %s", rel.TagName)
	}
	if len(rel.Assets) != 1 || rel.Assets[0].Name != "edi-darwin-arm64" {
		t.Errorf("assets no coinciden con el mock: %v", rel.Assets)
	}
}
