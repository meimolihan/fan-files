package fbhttp

import (
	"net/http"
	gopath "path"
	"strings"

	"github.com/meimolihan/fan-files/files"
)

type absolutePathResponse struct {
	Path string `json:"path"`
}

// absolutePathHandler resolves a virtual path (e.g. "/compose") to the real
// absolute path on the server's filesystem (e.g. "/vol1/1000/compose"). It is
// deliberately not exposed on the public share endpoints: only authenticated
// users may learn the on-disk location of a resource.
var absolutePathHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	requested := r.URL.Query().Get("path")
	if requested == "" {
		requested = "/"
	}

	file, err := files.NewFileInfo(&files.FileOptions{
		Fs:         d.user.Fs,
		Path:       gopath.Clean("/" + strings.TrimPrefix(requested, "/")),
		Modify:     d.user.Perm.Modify,
		Expand:     false,
		ReadHeader: false,
		Checker:    d,
		Content:    false,
	})
	if err != nil {
		return errToStatus(err), err
	}

	return renderJSON(w, r, &absolutePathResponse{Path: file.RealPath()})
})