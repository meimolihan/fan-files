package fbhttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/filebrowser/filebrowser/v2/files"
	"github.com/filebrowser/filebrowser/v2/settings"
	"github.com/filebrowser/filebrowser/v2/users"
	"github.com/spf13/afero"
)

func pathTestHandler(t *testing.T, userScope string) http.Handler {
	t.Helper()

	key := []byte("test-signing-key")
	perm := users.Permissions{Create: true, Modify: true, Download: true}
	st := scopedUserStorage(t, userScope, perm, key)
	// Mirror production: the user's filesystem is rooted at userScope, so the
	// resolved real path is userScope joined with the virtual path.
	st.Users = &customFSUser{
		Store:          st.Users,
		fs:             files.NewFs(afero.NewOsFs(), userScope, false),
		followExternal: true,
	}

	return handle(absolutePathHandler, "", st, &settings.Server{})
}

func TestPathResolvesRealAbsolutePath(t *testing.T) {
	userScope := t.TempDir()
	rel := filepath.Join("sub", "nested", "file.txt")
	if err := os.MkdirAll(filepath.Join(userScope, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userScope, rel), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}

	handler := pathTestHandler(t, userScope)

	req, _ := http.NewRequest(http.MethodGet, "/api/path?path=/sub/nested/file.txt", http.NoBody)
	req.Header.Set("X-Auth", signToken(t, users.Permissions{Download: true}, []byte("test-signing-key")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %q", rec.Code, rec.Body.String())
	}

	var resp absolutePathResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := filepath.Join(userScope, rel)
	if resp.Path != want {
		t.Fatalf("got real path %q, want %q", resp.Path, want)
	}
}

func TestPathDefaultsToScopeRoot(t *testing.T) {
	userScope := t.TempDir()
	handler := pathTestHandler(t, userScope)

	req, _ := http.NewRequest(http.MethodGet, "/api/path", http.NoBody)
	req.Header.Set("X-Auth", signToken(t, users.Permissions{Download: true}, []byte("test-signing-key")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %q", rec.Code, rec.Body.String())
	}

	var resp absolutePathResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Path != userScope {
		t.Fatalf("got real path %q, want %q", resp.Path, userScope)
	}
}

// Resolving a real absolute path on the server leaks the on-disk layout, so
// the endpoint must only answer authenticated requests.
func TestPathRequiresAuth(t *testing.T) {
	userScope := t.TempDir()
	handler := pathTestHandler(t, userScope)

	req, _ := http.NewRequest(http.MethodGet, "/api/path?path=/anything", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %q, want 401", rec.Code, rec.Body.String())
	}
}