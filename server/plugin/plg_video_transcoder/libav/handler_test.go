package libav

import (
	"testing"

	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/preset"
)

// TestMasterPlaylistCodecsMatchTheSource is the regression test for the stall a
// video-only source used to cause: the master advertised AAC unconditionally,
// so MSE opened an audio buffer no segment would ever fill and playback froze
// after the first segment.
//
// This package needs cgo + the libav headers, so it is not part of the CI test
// job; run it inside the build image (see ffmpeg/handler_test.go for the
// variant CI does run).
func TestMasterPlaylistCodecsMatchTheSource(t *testing.T) {
	p, _ := preset.Resolve("480p")

	want := "#EXTM3U\n" +
		"#EXT-X-VERSION:3\n" +
		`#EXT-X-STREAM-INF:BANDWIDTH=1628000,CODECS="avc1.64001f,mp4a.40.2"` + "\n" +
		"/hls/index.m3u8?path=vid_a_b.dat&preset=480p\n"
	if got := MasterPlaylist("vid_a_b.dat", p, true); got != want {
		t.Errorf("master playlist changed for an audio source:\ngot:\n%s\nwant:\n%s", got, want)
	}

	want = "#EXTM3U\n" +
		"#EXT-X-VERSION:3\n" +
		`#EXT-X-STREAM-INF:BANDWIDTH=1500000,CODECS="avc1.64001f"` + "\n" +
		"/hls/index.m3u8?path=vid_a_b.dat&preset=480p\n"
	if got := MasterPlaylist("vid_a_b.dat", p, false); got != want {
		t.Errorf("video-only master playlist is wrong:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
