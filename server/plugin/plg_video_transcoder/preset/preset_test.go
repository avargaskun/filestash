package preset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWhitelist(t *testing.T) {
	want := map[string]Preset{
		"720p": {Name: "720p", Height: 720, Bitrate: 4000000},
		"480p": {Name: "480p", Height: 480, Bitrate: 1500000},
		"360p": {Name: "360p", Height: 360, Bitrate: 800000},
	}
	for name, exp := range want {
		got, ok := Resolve(name)
		if !ok {
			t.Fatalf("Resolve(%q) not ok", name)
		}
		if got != exp {
			t.Errorf("Resolve(%q) = %+v, want %+v", name, got, exp)
		}
	}
}

func TestResolveRejectsUnknown(t *testing.T) {
	// bogus / hostile inputs must never resolve - this is the property that
	// keeps the quality parameter from reaching an ffmpeg argument.
	for _, s := range []string{
		"", "1080p", "720", "720P", "4k", "high",
		"720p ", " 720p", "720p;rm -rf /", "$(id)", "`id`",
		"-vf", "--", "0", "-1", "999999999",
		"scale=1920:-2", "720p,480p", "720p\n360p", "480p\x00",
		"../720p", "%37%32%30p",
	} {
		if _, ok := Resolve(s); ok {
			t.Errorf("Resolve(%q) unexpectedly ok", s)
		}
		if IsValid(s) {
			t.Errorf("IsValid(%q) unexpectedly true", s)
		}
	}
}

func TestFromRequestEmptyUsesDefault(t *testing.T) {
	p, ok := FromRequest("", "480p")
	if !ok || p.Name != "480p" {
		t.Fatalf("FromRequest(\"\",\"480p\") = %+v ok=%v, want 480p", p, ok)
	}
	// empty raw + invalid configured default falls back to the hard Default,
	// never to nothing.
	p, ok = FromRequest("", "garbage")
	if !ok || p.Name != Default {
		t.Fatalf("FromRequest(\"\",\"garbage\") = %+v ok=%v, want %s", p, ok, Default)
	}
}

func TestFromRequestBogusRejected(t *testing.T) {
	// a non-empty but bogus value is rejected even if the default is valid -
	// the client cannot smuggle an arbitrary value past the whitelist.
	for _, raw := range []string{"1080p", "-vf", "720p;id", "0"} {
		if _, ok := FromRequest(raw, "720p"); ok {
			t.Errorf("FromRequest(%q, \"720p\") unexpectedly ok", raw)
		}
	}
}

func TestNamesOrderedAndCopied(t *testing.T) {
	got := Names()
	want := []string{"720p", "480p", "360p"}
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	got[0] = "mutated"
	if Names()[0] != "720p" {
		t.Fatalf("Names() returned a mutable reference to the internal order")
	}
}

func TestDefaultIsWhitelisted(t *testing.T) {
	if !IsValid(Default) {
		t.Fatalf("Default %q is not a whitelisted preset", Default)
	}
}

// TestHandlersValidatePreset is an architectural guard: both HLS handlers must
// route the quality parameter through the whitelist (preset.FromRequest) before
// it can influence a transcode. It fails by design if a future edit drops the
// validation and lets a raw value reach the encoder.
func TestHandlersValidatePreset(t *testing.T) {
	for _, name := range []string{"../libav/handler.go", "../ffmpeg/handler.go"} {
		b, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.Contains(string(b), "preset.FromRequest(") {
			t.Errorf("%s does not validate the preset through the whitelist", name)
		}
	}
}
