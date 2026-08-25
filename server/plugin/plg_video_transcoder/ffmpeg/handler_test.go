package ffmpeg

import (
	"strings"
	"testing"

	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/preset"
)

// TestMasterPlaylistCodecsMatchTheSource is the regression test for the stall a
// video-only source used to cause: the master advertised AAC unconditionally,
// so MSE opened an audio buffer no segment would ever fill and playback froze
// after the first segment.
func TestMasterPlaylistCodecsMatchTheSource(t *testing.T) {
	p, _ := preset.Resolve("480p")

	withAudio := MasterPlaylist("vid_a_b.dat", p, true)
	if !strings.Contains(withAudio, `CODECS="avc1.64001f,mp4a.40.2",AUDIO="aud"`) {
		t.Errorf("audio source lost its audio declaration:\n%s", withAudio)
	}
	if !strings.Contains(withAudio, "#EXT-X-MEDIA:TYPE=AUDIO") {
		t.Errorf("audio source lost its audio rendition:\n%s", withAudio)
	}
	if !strings.Contains(withAudio, "BANDWIDTH=1628000") {
		t.Errorf("audio source bandwidth is not video+audio:\n%s", withAudio)
	}

	noAudio := MasterPlaylist("vid_a_b.dat", p, false)
	if strings.Contains(noAudio, "mp4a") || strings.Contains(noAudio, "AUDIO") {
		t.Errorf("video-only source still declares audio:\n%s", noAudio)
	}
	if !strings.Contains(noAudio, `CODECS="avc1.64001f"`) {
		t.Errorf("video-only source lost its video declaration:\n%s", noAudio)
	}
	if !strings.Contains(noAudio, "BANDWIDTH=1500000") {
		t.Errorf("video-only source bandwidth still budgets audio:\n%s", noAudio)
	}

	// the media playlist URI is the same either way - only the declaration changes
	if !strings.Contains(withAudio, "/hls/video.m3u8?path=vid_a_b.dat&preset=480p") ||
		!strings.Contains(noAudio, "/hls/video.m3u8?path=vid_a_b.dat&preset=480p") {
		t.Error("the variant URI changed")
	}
}

// TestMasterPlaylistAudioCaseIsUnchanged pins the exact bytes served to every
// audio-bearing source, which is every source but a camera clip.
func TestMasterPlaylistAudioCaseIsUnchanged(t *testing.T) {
	p, _ := preset.Resolve("720p")
	want := "#EXTM3U\n" +
		"#EXT-X-VERSION:6\n" +
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="default",DEFAULT=YES,AUTOSELECT=YES,URI="/hls/audio.m3u8?path=vid_a_b.dat"` + "\n" +
		`#EXT-X-STREAM-INF:BANDWIDTH=4128000,CODECS="avc1.64001f,mp4a.40.2",AUDIO="aud"` + "\n" +
		"/hls/video.m3u8?path=vid_a_b.dat&preset=720p\n"
	if got := MasterPlaylist("vid_a_b.dat", p, true); got != want {
		t.Errorf("master playlist changed for an audio source:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
