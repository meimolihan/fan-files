package fbhttp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meimolihan/fan-files/diskcache"
	"github.com/meimolihan/fan-files/settings"
	"github.com/meimolihan/fan-files/users"
)

func TestStoragesEndpointListsExtraStorages(t *testing.T) {
	key := []byte("test-signing-key")
	perm := users.Permissions{Create: true}
	st := scopedUserStorage(t, t.TempDir(), perm, key)
	if err := st.Settings.Save(&settings.Settings{Key: key}); err != nil {
		t.Fatal(err)
	}

	server := &settings.Server{
		Root:       t.TempDir(),
		ExtraRoots: []settings.ExtraRoot{{Path: t.TempDir(), Label: "存储空间2"}},
	}

	req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Auth", signToken(t, perm, key))
	rec := httptest.NewRecorder()
	handle(storagesGetHandler, "", st, server).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	var body strings.Builder
	body.WriteString(rec.Body.String())
	if !strings.Contains(body.String(), "存储空间1") || !strings.Contains(body.String(), "存储空间2") {
		t.Errorf("storages response = %q, want both 存储空间1 and 存储空间2", body.String())
	}
}

func TestResourcePostCreatesIntoRequestedStorage(t *testing.T) {
	root := t.TempDir()
	userScope := filepath.Join(root, "user")
	extraRoot := filepath.Join(root, "extra")
	for _, d := range []string{userScope, extraRoot} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	key := []byte("test-signing-key")
	perm := users.Permissions{Create: true, Modify: true}
	st := scopedUserStorage(t, userScope, perm, key)
	if err := st.Settings.Save(&settings.Settings{Key: key}); err != nil {
		t.Fatal(err)
	}
	server := &settings.Server{ExtraRoots: []settings.ExtraRoot{{Path: extraRoot, Label: "存储空间2"}}}
	signed := signToken(t, perm, key)

	doPost := func(target string, storage string, body string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodPost, target, strings.NewReader(body))
		if storage != "" {
			q := req.URL.Query()
			q.Set("storage", storage)
			req.URL.RawQuery = q.Encode()
		}
		req.Header.Set("X-Auth", signed)
		rec := httptest.NewRecorder()
		handle(resourcePostHandler(diskcache.NewNoOp()), "", st, server).ServeHTTP(rec, req)
		return rec
	}

	t.Run("new directory lands on the target storage", func(t *testing.T) {
		if rec := doPost("/brand-new-dir/", "存储空间2", ""); rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
		}
		if _, err := os.Stat(filepath.Join(extraRoot, "brand-new-dir")); err != nil {
			t.Errorf("directory should exist on the extra storage: %v", err)
		}
		if _, err := os.Stat(filepath.Join(userScope, "brand-new-dir")); err == nil {
			t.Error("directory should not exist on the primary storage")
		}
	})

	t.Run("new file lands on the target storage", func(t *testing.T) {
		if rec := doPost("/brand-new-file.txt", "存储空间2", "hello"); rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
		}
		data, err := os.ReadFile(filepath.Join(extraRoot, "brand-new-file.txt"))
		if err != nil {
			t.Errorf("file should exist on the extra storage: %v", err)
		} else if string(data) != "hello" {
			t.Errorf("file content = %q, want %q", string(data), "hello")
		}
		if _, err := os.Stat(filepath.Join(userScope, "brand-new-file.txt")); err == nil {
			t.Error("file should not exist on the primary storage")
		}
	})

	t.Run("without a storage the primary volume is used", func(t *testing.T) {
		if rec := doPost("/default-dir/", "", ""); rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
		}
		if _, err := os.Stat(filepath.Join(userScope, "default-dir")); err != nil {
			t.Errorf("directory should exist on the primary storage: %v", err)
		}
	})

	t.Run("an unknown storage label is a bad request", func(t *testing.T) {
		if rec := doPost("/bad-dir/", "存储空间9", ""); rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for unknown storage, got %d body=%q", rec.Code, rec.Body.String())
		}
		if _, err := os.Stat(filepath.Join(userScope, "bad-dir")); err == nil {
			t.Error("bad-dir should not be created")
		}
		if _, err := os.Stat(filepath.Join(extraRoot, "bad-dir")); err == nil {
			t.Error("bad-dir should not be created")
		}
	})
}
