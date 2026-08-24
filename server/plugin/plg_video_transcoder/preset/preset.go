// Package preset is the server-side whitelist of transcoding quality presets.
//
// A preset name arriving on an HLS URL only ever selects one of a fixed set of
// (height, bitrate) pairs compiled in here; the name itself never reaches an
// ffmpeg/libav argument. Anything not on the whitelist is rejected before a
// transcode is started, so the quality parameter carries no injection surface.
package preset

// Preset is a resolved quality tier. Bitrate is the target video bitrate in
// bits per second and drives rate control on every encoder.
type Preset struct {
	Name    string
	Height  int
	Bitrate int
}

// order is the menu order (highest quality first) and the source of truth for
// which names exist.
var order = []string{"720p", "480p", "360p"}

var table = map[string]Preset{
	"720p": {Name: "720p", Height: 720, Bitrate: 4000000},
	"480p": {Name: "480p", Height: 480, Bitrate: 1500000},
	"360p": {Name: "360p", Height: 360, Bitrate: 800000},
}

// Default is the preset used when a request does not name one and the
// configured default is itself invalid. It must always be a whitelisted name.
const Default = "720p"

// Resolve returns the preset named by s. An unknown name resolves to nothing -
// it never falls back to a default here, so callers can distinguish "not
// provided" from "provided but bogus".
func Resolve(s string) (Preset, bool) {
	p, ok := table[s]
	return p, ok
}

// FromRequest resolves the preset for an HLS request. raw is the value taken
// from the URL; def is the operator-configured default used when raw is empty.
// A non-empty raw that is not whitelisted returns ok=false so the handler can
// answer 400 without ever starting a transcode.
func FromRequest(raw, def string) (Preset, bool) {
	if raw == "" {
		if p, ok := Resolve(def); ok {
			return p, true
		}
		return table[Default], true
	}
	return Resolve(raw)
}

// IsValid reports whether s is a whitelisted preset name.
func IsValid(s string) bool {
	_, ok := table[s]
	return ok
}

// Names returns the preset names in menu order (a copy; callers may not mutate
// the internal ordering).
func Names() []string {
	out := make([]string, len(order))
	copy(out, order)
	return out
}
