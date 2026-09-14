package files

import (
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// MultiRoot describes one real directory tree that is exposed at the virtual
// root together with the label shown to users (e.g. "存储空间1").
type MultiRoot struct {
	Path  string
	Label string
}

// MultiFs merges several real directory trees into a single virtual root. The
// first root wins when two roots share a top-level entry name. Every operation
// is dispatched to the root that owns the top-level segment of the virtual
// path; paths that don't exist yet default to the first root, so new files are
// always created on the primary volume.
//
// Listings of the virtual root "/" never include dotfiles: the home page hides
// hidden files, while subdirectories keep showing them. The banner of the top
// resource, when it is the merged root itself, is the first root.
type MultiFs struct {
	roots []*multiRootFs
}

type multiRootFs struct {
	label string
	fs    afero.Fs
}

var (
	_ afero.Fs      = (*MultiFs)(nil)
	_ afero.Lstater = (*MultiFs)(nil)
)

// NewMultiFs builds a merged filesystem from the given real roots. Each root
// shares the same virtual "/" tree, so the virtual path passed to the result
// must be the same one used for the primary root (the user's scope).
func NewMultiFs(roots []MultiRoot, followExternal bool) *MultiFs {
	m := &MultiFs{roots: make([]*multiRootFs, 0, len(roots))}
	for _, r := range roots {
		m.roots = append(m.roots, &multiRootFs{
			label: r.Label,
			fs:    NewFs(afero.NewOsFs(), r.Path, followExternal),
		})
	}
	return m
}

// StorageOf returns the label of the root that owns name, or "" when name is
// the virtual root itself or no roots are configured.
func (m *MultiFs) StorageOf(name string) string {
	if len(m.roots) == 0 {
		return ""
	}
	seg, _ := splitFirst(name)
	if seg == "" {
		return ""
	}
	return m.owner(seg).label
}

// StorageLabels returns the label of every configured storage root, primary
// first. It lets the UI offer a target storage when creating new resources.
func (m *MultiFs) StorageLabels() []string {
	labels := make([]string, 0, len(m.roots))
	for _, r := range m.roots {
		labels = append(labels, r.label)
	}
	return labels
}

// RootIndexOf returns the index of the root whose label matches, or -1 when no
// root carries that label.
func (m *MultiFs) RootIndexOf(label string) int {
	for i, r := range m.roots {
		if r.label == label {
			return i
		}
	}
	return -1
}

// MkdirAllOnRoot creates a directory tree pinned to a specific storage root,
// regardless of where the path would normally be routed. It returns
// os.ErrNotExist when the root index is out of range.
func (m *MultiFs) MkdirAllOnRoot(name string, root int, perm os.FileMode) error {
	if root < 0 || root >= len(m.roots) {
		return os.ErrNotExist
	}
	return m.roots[root].fs.MkdirAll(name, perm)
}

// OpenFileOnRoot opens a file pinned to a specific storage root, regardless of
// where the path would normally be routed. It returns os.ErrNotExist when the
// root index is out of range.
func (m *MultiFs) OpenFileOnRoot(name string, root int, flag int, perm os.FileMode) (afero.File, error) {
	if root < 0 || root >= len(m.roots) {
		return nil, os.ErrNotExist
	}
	return m.roots[root].fs.OpenFile(name, flag, perm)
}

// ownerIndex returns the index of the root that owns a top-level segment, or
// 0 when the segment exists nowhere (so writes land on the primary volume).
func (m *MultiFs) ownerIndex(seg string) int {
	if len(m.roots) == 0 {
		return 0
	}
	for i, r := range m.roots {
		if _, err := lstatIfPossible(r.fs, "/"+seg); err == nil {
			return i
		}
	}
	return 0
}

func (m *MultiFs) owner(seg string) *multiRootFs { return m.roots[m.ownerIndex(seg)] }

// splitFirst returns the first segment of a virtual path and the remainder
// (with a leading "/"). For the root path it returns an empty segment.
func splitFirst(name string) (string, string) {
	name = normalize(name)
	from := strings.TrimLeft(name, "/")
	if from == "" {
		return "", ""
	}
	i := strings.IndexByte(from, '/')
	if i < 0 {
		return from, "/"
	}
	return from[:i], "/" + from[i:]
}

func normalize(name string) string {
	switch name {
	case "", ".":
		return "/"
	case "\\":
		return "/"
	}
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	return path.Clean(name)
}

// rootFor returns the filesystem and (normalized) name to operate on, or nil
// when no roots are configured.
func (m *MultiFs) rootFor(name string) (afero.Fs, string) {
	if len(m.roots) == 0 {
		return nil, name
	}
	seg, _ := splitFirst(name)
	if seg == "" {
		return m.roots[0].fs, name
	}
	return m.owner(seg).fs, name
}

func (m *MultiFs) Create(name string) (afero.File, error) {
	r, name := m.rootFor(name)
	if r == nil {
		return nil, os.ErrNotExist
	}
	return r.Create(name)
}

func (m *MultiFs) Mkdir(name string, perm os.FileMode) error {
	r, name := m.rootFor(name)
	if r == nil {
		return os.ErrNotExist
	}
	return r.Mkdir(name, perm)
}

func (m *MultiFs) MkdirAll(name string, perm os.FileMode) error {
	r, name := m.rootFor(name)
	if r == nil {
		return os.ErrNotExist
	}
	return r.MkdirAll(name, perm)
}

func (m *MultiFs) Open(name string) (afero.File, error) {
	name = normalize(name)
	if name == "/" {
		return m.mergedRootFile()
	}
	r, name := m.rootFor(name)
	if r == nil {
		return nil, os.ErrNotExist
	}
	return r.Open(name)
}

func (m *MultiFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	r, name := m.rootFor(name)
	if r == nil {
		return nil, os.ErrNotExist
	}
	return r.OpenFile(name, flag, perm)
}

func (m *MultiFs) Remove(name string) error {
	r, name := m.rootFor(name)
	if r == nil {
		return os.ErrNotExist
	}
	return r.Remove(name)
}

func (m *MultiFs) RemoveAll(name string) error {
	r, name := m.rootFor(name)
	if r == nil {
		return os.ErrNotExist
	}
	return r.RemoveAll(name)
}

func (m *MultiFs) Rename(oldname, newname string) error {
	oldR, oldName := m.rootFor(oldname)
	if oldR == nil {
		return os.ErrNotExist
	}
	newR, newName := m.rootFor(newname)
	if newR == nil {
		return os.ErrNotExist
	}
	if oldR != newR {
		// Different underlying volumes: a cross-device rename would fail, so
		// report a clean error and let the caller fall back to copy+delete.
		return os.ErrPermission
	}
	return oldR.Rename(oldName, newName)
}

func (m *MultiFs) Stat(name string) (os.FileInfo, error) {
	r, name := m.rootFor(name)
	if r == nil {
		return nil, os.ErrNotExist
	}
	return r.Stat(name)
}

func (m *MultiFs) Name() string { return "MultiFs" }

func (m *MultiFs) Chmod(name string, mode os.FileMode) error {
	r, name := m.rootFor(name)
	if r == nil {
		return os.ErrNotExist
	}
	return r.Chmod(name, mode)
}

func (m *MultiFs) Chown(name string, uid, gid int) error {
	r, name := m.rootFor(name)
	if r == nil {
		return os.ErrNotExist
	}
	return r.Chown(name, uid, gid)
}

func (m *MultiFs) Chtimes(name string, atime, mtime time.Time) error {
	r, name := m.rootFor(name)
	if r == nil {
		return os.ErrNotExist
	}
	return r.Chtimes(name, atime, mtime)
}

func (m *MultiFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	name = normalize(name)
	if name == "/" {
		r, _ := m.rootFor("/")
		if r == nil {
			return nil, false, os.ErrNotExist
		}
		info, err := lstatIfPossible(r, "/")
		return info, true, err
	}
	r, name := m.rootFor(name)
	if r == nil {
		return nil, false, os.ErrNotExist
	}
	info, err := lstatIfPossible(r, name)
	return info, true, err
}

// RealPath resolves a virtual path to the real on-disk path of its owning root.
func (m *MultiFs) RealPath(name string) (string, error) {
	name = normalize(name)
	r, name := m.rootFor(name)
	if r == nil {
		return name, os.ErrNotExist
	}
	if realPathFs, ok := r.(interface {
		RealPath(string) (string, error)
	}); ok {
		return realPathFs.RealPath(name)
	}
	return name, nil
}

// mergedList builds the sorted union of every root's top-level entries. It
// never includes dotfiles: the home page hides hidden files. When every root
// fails to be read, the first error is returned so the failure is not masked
// as an empty directory.
func (m *MultiFs) mergedList() ([]os.FileInfo, error) {
	var (
		firstErr error
		seen     = map[string]bool{}
		out      []os.FileInfo
	)
	for _, r := range m.roots {
		infos, err := afero.ReadDir(r.fs, "/")
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, fi := range infos {
			name := fi.Name()
			if seen[name] {
				continue
			}
			if strings.HasPrefix(name, ".") {
				continue
			}
			seen[name] = true
			out = append(out, fi)
		}
	}
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

func (m *MultiFs) mergedRootFile() (afero.File, error) {
	infos, err := m.mergedList()
	if err != nil {
		return nil, err
	}
	return &multiDirFile{m: m, infos: infos}, nil
}

// multiDirFile is a virtual in-memory directory file returned for the merged
// root. Reads and writes are rejected.
type multiDirFile struct {
	m     *MultiFs
	infos []os.FileInfo
	pos   int
}

func (f *multiDirFile) Name() string { return "/" }

func (f *multiDirFile) Readdir(count int) ([]os.FileInfo, error) {
	if count < 0 {
		part := f.infos[f.pos:]
		f.pos = len(f.infos)
		return part, nil
	}
	if f.pos >= len(f.infos) {
		return nil, io.EOF
	}
	end := f.pos + count
	if end > len(f.infos) {
		end = len(f.infos)
	}
	part := f.infos[f.pos:end]
	f.pos = end
	if len(part) == 0 {
		return nil, io.EOF
	}
	return part, nil
}

func (f *multiDirFile) Readdirnames(count int) ([]string, error) {
	infos, err := f.Readdir(count)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(infos))
	for i, fi := range infos {
		names[i] = fi.Name()
	}
	return names, nil
}

func (f *multiDirFile) Read(_ []byte) (int, error)               { return 0, io.EOF }
func (f *multiDirFile) ReadAt(_ []byte, _ int64) (int, error)    { return 0, io.EOF }
func (f *multiDirFile) Seek(_ int64, _ int) (int64, error) {
	return 0, os.ErrInvalid
}
func (f *multiDirFile) Stat() (os.FileInfo, error)  { return f.m.Stat("/") }
func (f *multiDirFile) Sync() error                 { return nil }
func (f *multiDirFile) Truncate(_ int64) error      { return os.ErrInvalid }
func (f *multiDirFile) Write(_ []byte) (int, error) { return 0, os.ErrInvalid }
func (f *multiDirFile) WriteString(_ string) (int, error) {
	return 0, os.ErrInvalid
}
func (f *multiDirFile) WriteAt(p []byte, off int64) (int, error) {
	return 0, os.ErrInvalid
}
func (f *multiDirFile) Close() error { return nil }
