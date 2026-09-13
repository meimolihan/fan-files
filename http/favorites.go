package fbhttp

import (
	"encoding/json"
	"net/http"
	gopath "path"
	"strings"

	fberrors "github.com/meimolihan/fan-files/errors"
	"github.com/meimolihan/fan-files/files"
	"github.com/meimolihan/fan-files/users"
)

type favoritesBody struct {
	Path string `json:"path"`
}

// cleanFavoritePath canonicalizes a user-supplied path to the "/"-separated
// virtual form stored in the user's favorites.
func cleanFavoritePath(p string) string {
	return gopath.Clean("/" + strings.TrimPrefix(p, "/"))
}

// resolveFavorite validates that path is a directory the user can reach and
// returns the favorite record with its real path resolved. Only directories
// can be favorited.
func resolveFavorite(d *data, path string) (*users.Favorite, error) {
	path = cleanFavoritePath(path)
	file, err := files.NewFileInfo(&files.FileOptions{
		Fs:         d.user.Fs,
		Path:       path,
		Modify:     d.user.Perm.Modify,
		Expand:     false,
		ReadHeader: false,
		Checker:    d,
		Content:    false,
	})
	if err != nil {
		return nil, err
	}
	if !file.IsDir {
		return nil, fberrors.ErrInvalidRequestParams
	}

	return &users.Favorite{Path: path, RealPath: file.RealPath()}, nil
}

func renderFavorites(w http.ResponseWriter, r *http.Request, list []users.Favorite) (int, error) {
	if list == nil {
		list = []users.Favorite{}
	}
	return renderJSON(w, r, list)
}

// favoritesGetHandler lists the current user's favorited directories.
var favoritesGetHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	return renderFavorites(w, r, d.user.Favorites)
})

// favoritesPostHandler pins a directory to the current user's favorites.
// Adding an already-favorited directory is a no-op.
var favoritesPostHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodySize)

	var body favoritesBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return http.StatusBadRequest, err
	}

	fav, err := resolveFavorite(d, body.Path)
	if err != nil {
		return errToStatus(err), err
	}

	for _, existing := range d.user.Favorites {
		if existing.Path == fav.Path {
			return renderFavorites(w, r, d.user.Favorites)
		}
	}

	d.user.Favorites = append(d.user.Favorites, *fav)
	if err := d.store.Users.Update(d.user, "Favorites"); err != nil {
		return http.StatusInternalServerError, err
	}

	return renderFavorites(w, r, d.user.Favorites)
})

// favoritesDeleteHandler unpins a directory from the current user's favorites.
// Removing a directory that is not favorited is a no-op.
var favoritesDeleteHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodySize)

	var body favoritesBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return http.StatusBadRequest, err
	}

	path := cleanFavoritePath(body.Path)
	filtered := d.user.Favorites[:0]
	found := false
	for _, fav := range d.user.Favorites {
		if fav.Path == path {
			found = true
			continue
		}
		filtered = append(filtered, fav)
	}

	if !found {
		return renderFavorites(w, r, d.user.Favorites)
	}

	d.user.Favorites = filtered
	if err := d.store.Users.Update(d.user, "Favorites"); err != nil {
		return http.StatusInternalServerError, err
	}

	return renderFavorites(w, r, d.user.Favorites)
})