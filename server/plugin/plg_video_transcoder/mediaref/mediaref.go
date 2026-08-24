// Package mediaref maps the opaque keys used in HLS URLs to the file the
// transcoder should read.
//
// The HLS playlist and segment routes are not behind the session middleware:
// their only protection is that a caller cannot guess the key. Resolving a key
// through this registry - rather than joining it onto a directory - is what
// keeps `?path=` from being usable as an arbitrary-file read.
package mediaref

import (
	"errors"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

const MaxEntries = 4096

var (
	ErrInvalidKey  = errors.New("mediaref: malformed key")
	ErrInvalidPath = errors.New("mediaref: path must be absolute and clean")
	ErrInvalidTTL  = errors.New("mediaref: ttl must be positive")
	ErrTooMany     = errors.New("mediaref: registry is full")
)

// keyPattern is deliberately stricter than "no traversal": a key is an opaque
// token, so anything that is not one is rejected before it reaches a syscall.
var keyPattern = regexp.MustCompile(`^vid_[A-Za-z0-9]{1,64}_[A-Za-z0-9]{1,64}\.dat$`)

type entry struct {
	path      string
	expiresAt time.Time
}

var (
	mu      sync.Mutex
	entries = make(map[string]entry)
)

// IsValidKey reports whether s has the shape of a media key.
func IsValidKey(s string) bool {
	return keyPattern.MatchString(s)
}

// Register binds key to an absolute path for the given duration. Registering a
// known key refreshes it.
func Register(key string, path string, ttl time.Duration) error {
	if IsValidKey(key) == false {
		return ErrInvalidKey
	}
	if filepath.IsAbs(path) == false || filepath.Clean(path) != path {
		return ErrInvalidPath
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}

	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	if _, known := entries[key]; known == false && len(entries) >= MaxEntries {
		evictExpired(now)
		if len(entries) >= MaxEntries {
			return ErrTooMany
		}
	}
	entries[key] = entry{path: path, expiresAt: now.Add(ttl)}
	return nil
}

// Resolve returns the path bound to key. A malformed, unknown or expired key
// resolves to nothing - it is never interpreted as a filesystem path.
func Resolve(key string) (string, bool) {
	if IsValidKey(key) == false {
		return "", false
	}
	mu.Lock()
	defer mu.Unlock()
	e, ok := entries[key]
	if ok == false {
		return "", false
	}
	if time.Now().After(e.expiresAt) {
		delete(entries, key)
		return "", false
	}
	return e.path, true
}

// Forget drops key from the registry.
func Forget(key string) {
	mu.Lock()
	defer mu.Unlock()
	delete(entries, key)
}

// Count returns the number of entries currently held.
func Count() int {
	mu.Lock()
	defer mu.Unlock()
	return len(entries)
}

func evictExpired(now time.Time) {
	for key, e := range entries {
		if now.After(e.expiresAt) {
			delete(entries, key)
		}
	}
}
