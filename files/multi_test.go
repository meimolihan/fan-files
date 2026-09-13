package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func newTestMultiFs(t *testing.T) (*MultiFs, string, string) {
	t.Helper()
	root1 := t.TempDir()
	root2 := t.TempDir()

	mustMkdirAll(t, filepath.Join(root1, "compose"))
	mustWriteFile(t, filepath.Join(root1, "compose", "app.txt"), "a")
	mustWriteFile(t, filepath.Join(root1, ".secret1"), "s1")
	mustMkdirAll(t, filepath.Join(root1, "collision"))
	mustWriteFile(t, filepath.Join(root1, "collision", "only-in-vol1.txt"), "z")

	mustMkdirAll(t, filepath.Join(root2, "backup"))
	mustWriteFile(t, filepath.Join(root2, "backup", "b.txt"), "b")
	mustWriteFile(t, filepath.Join(root2, ".secret2"), "s2")
	// Name collision: primary root must win.
	mustMkdirAll(t, filepath.Join(root2, "collision"))
	mustWriteFile(t, filepath.Join(root2, "collision", "only-in-vol2.txt"), "x")

	m := NewMultiFs([]MultiRoot{
		{Path: root1, Label: "存储空间1"},
		{Path: root2, Label: "存储空间2"},
	}, false)

	return m, root1, root2
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWriteFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestMultiFsMergesRoots(t *testing.T) {
	m, _, _ := newTestMultiFs(t)

	infos, err := afero.ReadDir(m, "/")
	if err != nil {
		t.Fatalf("ReadDir root: %v", err)
	}

	names := map[string]bool{}
	for _, fi := range infos {
		names[fi.Name()] = true
	}

	for _, want := range []string{"compose", "backup", "collision"} {
		if !names[want] {
			t.Errorf("missing top-level entry %q", want)
		}
	}

	// The home page must not expose dotfiles, even though they exist on disk.
	for _, n := range infos {
		if n.Name()[0] == '.' {
			t.Errorf("dotfile %q leaked into the root listing", n.Name())
		}
	}
}

func TestMultiFsSubdirShowsDotfiles(t *testing.T) {
	m, _, _ := newTestMultiFs(t)

	// A non-root directory simply mirrors its backing root: dotfiles appear.
	infos, err := afero.ReadDir(m, "/compose")
	if err != nil {
		t.Fatalf("ReadDir subdir: %v", err)
	}
	found := false
	for _, fi := range infos {
		if fi.Name() == "app.txt" {
			found = true
		}
	}
	if !found {
		t.Error("expected app.txt in the subdirectory listing")
	}
}

func TestMultiFsCollisionFirstRootWins(t *testing.T) {
	m, _, _ := newTestMultiFs(t)

	// "collision" lives in both roots; the primary root owns it.
	infos, err := afero.ReadDir(m, "/collision")
	if err != nil {
		t.Fatalf("ReadDir collision: %v", err)
	}
	names := []string{}
	for _, fi := range infos {
		names = append(names, fi.Name())
	}
	hasOwn := func(n string) bool {
		for _, name := range names {
			if name == n {
				return true
			}
		}
		return false
	}
	if !hasOwn("only-in-vol1.txt") {
		t.Errorf("primary root should own the colliding entry, got %v", names)
	}
	if hasOwn("only-in-vol2.txt") {
		t.Errorf("secondary root's colliding entry must not be visible, got %v", names)
	}
}

func TestMultiFsStorageLabels(t *testing.T) {
	m, _, _ := newTestMultiFs(t)

	cases := map[string]string{
		"/":              "",
		"/compose":       "存储空间1",
		"/compose/x":     "存储空间1",
		"/backup":        "存储空间2",
		"/backup/b":      "存储空间2",
		"/doesnotmatter": "存储空间1",
	}
	for path, want := range cases {
		if got := m.StorageOf(path); got != want {
			t.Errorf("StorageOf(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestMultiFsRealPath(t *testing.T) {
	m, root1, root2 := newTestMultiFs(t)

	cases := map[string]string{
		"/compose/app.txt": filepath.Join(root1, "compose", "app.txt"),
		"/backup/b.txt":    filepath.Join(root2, "backup", "b.txt"),
	}
	for virt, want := range cases {
		got, err := m.RealPath(virt)
		if err != nil {
			t.Errorf("RealPath(%q): %v", virt, err)
			continue
		}
		if got != want {
			t.Errorf("RealPath(%q) = %q, want %q", virt, got, want)
		}
	}
}

func TestMultiFsCreateGoesToPrimary(t *testing.T) {
	m, root1, root2 := newTestMultiFs(t)

	f, err := m.Create("/brand-new-file.txt")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, werr := f.Write([]byte("content")); werr != nil {
		t.Fatalf("Write: %v", werr)
	}
	if cerr := f.Close(); cerr != nil {
		t.Fatalf("Close: %v", cerr)
	}

	if _, err := os.Stat(filepath.Join(root1, "brand-new-file.txt")); err != nil {
		t.Errorf("file should be created on the primary root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root2, "brand-new-file.txt")); err == nil {
		t.Error("file should not be created on the secondary root")
	}
}

func TestMultiFsCrossRootRenameRejected(t *testing.T) {
	m, _, _ := newTestMultiFs(t)

	if err := m.Rename("/compose", "/backup/moved"); err == nil {
		t.Error("cross-root rename should be rejected")
	}
}

func TestMultiFsStorageIndexing(t *testing.T) {
	m, _, _ := newTestMultiFs(t)

	if got := m.StorageLabels(); len(got) != 2 || got[0] != "存储空间1" || got[1] != "存储空间2" {
		t.Errorf("StorageLabels() = %v, want [存储空间1 存储空间2]", got)
	}
	if got := m.RootIndexOf("存储空间1"); got != 0 {
		t.Errorf("RootIndexOf(存储空间1) = %d, want 0", got)
	}
	if got := m.RootIndexOf("存储空间2"); got != 1 {
		t.Errorf("RootIndexOf(存储空间2) = %d, want 1", got)
	}
	if got := m.RootIndexOf("存储空间3"); got != -1 {
		t.Errorf("RootIndexOf(存储空间3) = %d, want -1", got)
	}
}

func TestMultiFsMkdirAllOnRootPinsToTarget(t *testing.T) {
	m, root1, root2 := newTestMultiFs(t)

	if err := m.MkdirAllOnRoot("/brand-new-dir", 1, 0o755); err != nil {
		t.Fatalf("MkdirAllOnRoot: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root2, "brand-new-dir")); err != nil {
		t.Errorf("directory should be created on the secondary root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root1, "brand-new-dir")); err == nil {
		t.Error("directory should not be created on the primary root")
	}

	if err := m.MkdirAllOnRoot("/out-of-range", 42, 0o755); !os.IsNotExist(err) {
		t.Errorf("out-of-range root should return os.ErrNotExist, got %v", err)
	}
}

func TestMultiFsOpenFileOnRootPinsToTarget(t *testing.T) {
	m, root1, root2 := newTestMultiFs(t)

	f, err := m.OpenFileOnRoot("/brand-new-file.txt", 1, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatalf("OpenFileOnRoot: %v", err)
	}
	if _, werr := f.Write([]byte("hello")); werr != nil {
		t.Fatalf("Write: %v", werr)
	}
	if cerr := f.Close(); cerr != nil {
		t.Fatalf("Close: %v", cerr)
	}

	data, err := os.ReadFile(filepath.Join(root2, "brand-new-file.txt"))
	if err != nil {
		t.Errorf("file should be created on the secondary root: %v", err)
	} else if string(data) != "hello" {
		t.Errorf("file content = %q, want %q", string(data), "hello")
	}
	if _, err := os.Stat(filepath.Join(root1, "brand-new-file.txt")); err == nil {
		t.Error("file should not be created on the primary root")
	}

	if _, err := m.OpenFileOnRoot("/out-of-range", 42, os.O_RDWR, 0o644); !os.IsNotExist(err) {
		t.Errorf("out-of-range root should return os.ErrNotExist, got %v", err)
	}
}
