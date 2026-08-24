package mediaref

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const validKey = "vid_abc123_DEF456.dat"

func reset() {
	mu.Lock()
	entries = make(map[string]entry)
	mu.Unlock()
}

// TestResolveRejectsTraversal is the regression test for the property the HLS
// routes depend on: they are session-unauthenticated, so `?path=` must never be
// usable to read a file the registry was not told about.
func TestResolveRejectsTraversal(t *testing.T) {
	reset()
	if err := Register(validKey, "/srv/media/movie.mkv", time.Hour); err != nil {
		t.Fatalf("Register: %v", err)
	}

	hostile := []string{
		"../../../../etc/passwd",
		"../../etc/shadow",
		"vid_a_b.dat/../../../../etc/passwd",
		"vid_a_b.dat/../" + validKey,
		"/etc/passwd",
		"/app/data/cache/video/" + validKey,
		"./" + validKey,
		"..%2F..%2Fetc%2Fpasswd",
		"....//....//etc/passwd",
		"vid_abc123_DEF456.dat\x00.mkv",
		"vid_abc123_DEF456.dat ",
		" vid_abc123_DEF456.dat",
		"vid_abc123_DEF456.dat\n",
		"vid_../_x.dat",
		"vid__.dat",
		"vid_abc123_DEF456.dat.mkv",
		"VID_abc123_DEF456.DAT",
		"",
		"/",
		"..",
		"~/.ssh/id_rsa",
		"file:///etc/passwd",
		"\\..\\..\\etc\\passwd",
	}
	for _, key := range hostile {
		if got, ok := Resolve(key); ok {
			t.Errorf("Resolve(%q) resolved to %q, want no resolution", key, got)
		}
	}
}

// TestRegisterRejectsRelativeOrUncleanPaths keeps the registry itself from
// becoming the traversal primitive.
func TestRegisterRejectsRelativeOrUncleanPaths(t *testing.T) {
	reset()
	for _, path := range []string{
		"relative/path.mkv",
		"",
		"/srv/media/../../etc/passwd",
		"/srv/media/./movie.mkv",
		"/srv/media//movie.mkv",
	} {
		if err := Register(validKey, path, time.Hour); err != ErrInvalidPath {
			t.Errorf("Register(_, %q) = %v, want ErrInvalidPath", path, err)
		}
		if _, ok := Resolve(validKey); ok {
			t.Fatalf("Register(_, %q) bound the key anyway", path)
		}
	}
}

func TestRegisterRejectsMalformedKeys(t *testing.T) {
	reset()
	for _, key := range []string{"../escape.dat", "vid_a_b.dat/x", "", "vid_a_b.txt", strings.Repeat("a", 200)} {
		if err := Register(key, "/srv/media/movie.mkv", time.Hour); err != ErrInvalidKey {
			t.Errorf("Register(%q, _) = %v, want ErrInvalidKey", key, err)
		}
	}
	if n := Count(); n != 0 {
		t.Errorf("Count() = %d, want 0", n)
	}
}

func TestResolveRoundTrip(t *testing.T) {
	reset()
	const path = "/srv/media/some movie (2024)/movie.mkv"
	if err := Register(validKey, path, time.Hour); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, ok := Resolve(validKey)
	if ok == false || got != path {
		t.Fatalf("Resolve = (%q, %v), want (%q, true)", got, ok, path)
	}
}

// TestResolveUnknownWellFormedKey covers the case that upstream got wrong: a
// key with the right shape that was never registered must not fall back to
// probing the cache directory.
func TestResolveUnknownWellFormedKey(t *testing.T) {
	reset()
	if _, ok := Resolve("vid_notregistered_0000.dat"); ok {
		t.Fatal("an unregistered key resolved")
	}
}

func TestResolveExpired(t *testing.T) {
	reset()
	if err := Register(validKey, "/srv/media/movie.mkv", time.Nanosecond); err != nil {
		t.Fatalf("Register: %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, ok := Resolve(validKey); ok {
		t.Fatal("an expired key resolved")
	}
	if n := Count(); n != 0 {
		t.Errorf("Count() = %d, want the expired entry dropped", n)
	}
}

func TestForget(t *testing.T) {
	reset()
	if err := Register(validKey, "/srv/media/movie.mkv", time.Hour); err != nil {
		t.Fatalf("Register: %v", err)
	}
	Forget(validKey)
	if _, ok := Resolve(validKey); ok {
		t.Fatal("a forgotten key resolved")
	}
}

func TestRegisterRefreshesAndBounds(t *testing.T) {
	reset()
	for i := 0; i < MaxEntries; i++ {
		key := fmt.Sprintf("vid_k%d_h%d.dat", i, i)
		if err := Register(key, fmt.Sprintf("/srv/media/%d.mkv", i), time.Hour); err != nil {
			t.Fatalf("Register #%d: %v", i, err)
		}
	}
	if err := Register("vid_overflow_0.dat", "/srv/media/x.mkv", time.Hour); err != ErrTooMany {
		t.Errorf("Register past MaxEntries = %v, want ErrTooMany", err)
	}
	// a known key is refreshed rather than counted as new
	if err := Register("vid_k0_h0.dat", "/srv/media/0.mkv", time.Hour); err != nil {
		t.Errorf("refreshing a known key: %v", err)
	}
	if n := Count(); n != MaxEntries {
		t.Errorf("Count() = %d, want %d", n, MaxEntries)
	}
}

func TestConcurrentAccess(t *testing.T) {
	reset()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("vid_c%d_h%d.dat", i, i)
			Register(key, fmt.Sprintf("/srv/media/%d.mkv", i), time.Hour)
			Resolve(key)
			Resolve("../../../etc/passwd")
			Forget(key)
		}(i)
	}
	wg.Wait()
}

// TestHandlersResolveThroughRegistry is an architectural guard: neither HLS
// handler may rebuild a filesystem path out of the client-supplied `path`
// query parameter, which is how the traversal got in.
func TestHandlersResolveThroughRegistry(t *testing.T) {
	for _, name := range []string{"../libav/handler.go", "../ffmpeg/handler.go"} {
		b, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		src := string(b)
		if strings.Contains(src, "mediaref.Resolve(") == false {
			t.Errorf("%s does not resolve through the registry", name)
		}
		if strings.Contains(src, "GetAbsolutePath(") {
			t.Errorf("%s builds a filesystem path from a request parameter", name)
		}
	}
}
