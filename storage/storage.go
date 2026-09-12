package storage

import (
	"github.com/meimolihan/fan-files/auth"
	"github.com/meimolihan/fan-files/settings"
	"github.com/meimolihan/fan-files/share"
	"github.com/meimolihan/fan-files/users"
)

// Storage is a storage powered by a Backend which makes the necessary
// verifications when fetching and saving data to ensure consistency.
type Storage struct {
	Users    users.Store
	Share    *share.Storage
	Auth     *auth.Storage
	Settings *settings.Storage
}
