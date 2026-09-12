package bolt

import (
	"github.com/asdine/storm/v3"

	"github.com/meimolihan/fan-files/auth"
	fberrors "github.com/meimolihan/fan-files/errors"
	"github.com/meimolihan/fan-files/settings"
)

type authBackend struct {
	db *storm.DB
}

func (s authBackend) Get(t settings.AuthMethod) (auth.Auther, error) {
	var auther auth.Auther

	switch t {
	case auth.MethodJSONAuth:
		auther = &auth.JSONAuth{}
	case auth.MethodProxyAuth:
		auther = &auth.ProxyAuth{}
	case auth.MethodHookAuth:
		auther = &auth.HookAuth{}
	case auth.MethodNoAuth:
		auther = &auth.NoAuth{}
	default:
		return nil, fberrors.ErrInvalidAuthMethod
	}

	return auther, get(s.db, "auther", auther)
}

func (s authBackend) Save(a auth.Auther) error {
	return save(s.db, "auther", a)
}
