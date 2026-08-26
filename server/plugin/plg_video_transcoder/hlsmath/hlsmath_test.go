package hlsmath

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// TestPlaylistShape is the regression table for the phantom tail: a container
// that declares more duration than it holds content used to be advertised in
// full, so every segment past the real end was a window a player could seek
// into and nothing could fill. The counts here are the whole point — asserting
// on the clamped time alone would not have caught the defect.
func TestPlaylistShape(t *testing.T) {
	for _, tc := range []struct {
		name      string
		container float64
		content   float64
		segLen    int
		wantEnd   float64
		wantTotal int
	}{
		{"phantom tail is cut", 120.000, 59.977, 5, 59.977, 12},
		{"the real episode", 1484.050, 1430.093, 5, 1430.093, 287},
		{"unclamped", 1484.050, 0, 5, 1484.050, 297},
		{"content past the container is ignored", 60.021, 60.500, 5, 60.021, 13},
		{"content equal to the container is not a clamp", 60.021, 60.021, 5, 60.021, 13},
		{"probe failed, fail open", 60.021, 0, 5, 60.021, 13},
		{"negative content, fail open", 60.021, -1, 5, 60.021, 13},
		{"NaN content, fail open", 60.021, math.NaN(), 5, 60.021, 13},
		{"exact boundary loses the last frame, not a segment", 60.024, 60.000, 5, 60.000, 12},
		{"just under a boundary", 60.021, 59.989, 5, 59.989, 12},
		{"just over a boundary keeps its segment", 60.123, 60.093, 5, 60.093, 13},

		// HLS_SEGMENT_LENGTH is a var, not a const: it becomes 10 on a small arm
		// box, and no gate row ever exercises that path.
		{"arm: phantom tail is cut", 120.000, 59.977, 10, 59.977, 6},
		{"arm: unclamped", 120.000, 0, 10, 120.000, 12},
		{"arm: exact boundary", 60.024, 60.000, 10, 60.000, 6},
		{"arm: just under a boundary", 60.021, 59.989, 10, 59.989, 6},
		{"arm: just over a boundary", 60.123, 60.093, 10, 60.093, 7},
		{"arm: fail open", 60.021, 0, 10, 60.021, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			end, total := PlaylistShape(tc.container, tc.content, tc.segLen)
			if end != tc.wantEnd || total != tc.wantTotal {
				t.Errorf("PlaylistShape(%v, %v, %d) = (%v, %d), want (%v, %d)",
					tc.container, tc.content, tc.segLen, end, total, tc.wantEnd, tc.wantTotal)
			}
		})
	}
}

// TestMediaPlaylistClampedText asserts the text, because the text is where the
// defect lived: 24 entries were served for a file holding 12 segments' worth.
func TestMediaPlaylistClampedText(t *testing.T) {
	end, total := PlaylistShape(120.000, 59.977, 5)
	got := MediaPlaylist(end, 5, ", nodesc", func(i int) string {
		return fmt.Sprintf("/hls/segment_%d.ts?path=vid_a_b.dat&preset=480p", i)
	})

	if total != 12 {
		t.Fatalf("total = %d, want 12", total)
	}
	if n := strings.Count(got, "#EXTINF:"); n != 12 {
		t.Errorf("playlist has %d entries, want 12:\n%s", n, got)
	}
	if !strings.Contains(got, "#EXTINF:4.9770, nodesc\n/hls/segment_11.ts?path=vid_a_b.dat&preset=480p\n#EXT-X-ENDLIST\n") {
		t.Errorf("terminal entry is not the clamped sliver:\n%s", got)
	}
	if strings.Contains(got, "segment_12.ts") {
		t.Errorf("playlist still advertises a phantom segment:\n%s", got)
	}
	for _, header := range []string{
		"#EXTM3U\n", "#EXT-X-VERSION:3\n", "#EXT-X-MEDIA-SEQUENCE:0\n",
		"#EXT-X-ALLOW-CACHE:YES\n", "#EXT-X-PLAYLIST-TYPE:VOD\n", "#EXT-X-TARGETDURATION:5\n",
	} {
		if !strings.Contains(got, header) {
			t.Errorf("missing header %q:\n%s", header, got)
		}
	}
}

// TestMediaPlaylistIsByteIdenticalToTheOldBuilders is what licenses moving three
// hand-rolled playlist loops into one shared builder inside a fix PR: for an
// unclamped input every byte of all three playlist shapes must still match what
// the handlers emitted before the extraction.
func TestMediaPlaylistIsByteIdenticalToTheOldBuilders(t *testing.T) {
	const key = "vid_a_b.dat"

	for _, tc := range []struct {
		name     string
		duration float64
		segLen   int
		suffix   string
		segURL   func(i int) string
		want     func(duration float64, segLen int) string
	}{
		{
			name: "cgo media playlist", duration: 60.021, segLen: 5, suffix: ", nodesc",
			segURL: func(i int) string {
				return fmt.Sprintf("/hls/segment_%d.ts?path=%s&preset=%s", i, key, "480p")
			},
			want: func(duration float64, segLen int) string {
				return oldBuilder(duration, segLen, ", nodesc", func(i int) string {
					return fmt.Sprintf("/hls/segment_%d.ts?path=%s&preset=%s\n", i, key, "480p")
				})
			},
		},
		{
			name: "nocgo video sub-playlist", duration: 1484.050, segLen: 5, suffix: ", nodesc",
			segURL: func(i int) string {
				return fmt.Sprintf("/hls/video_%d.ts?path=%s&preset=%s", i, key, "360p")
			},
			want: func(duration float64, segLen int) string {
				return oldBuilder(duration, segLen, ", nodesc", func(i int) string {
					return fmt.Sprintf("/hls/video_%d.ts?path=%s&preset=%s\n", i, key, "360p")
				})
			},
		},
		{
			name: "nocgo audio sub-playlist", duration: 1484.050, segLen: 80, suffix: ",",
			segURL: func(i int) string {
				return fmt.Sprintf("/hls/audio_%d.ts?path=%s", i, key)
			},
			want: func(duration float64, segLen int) string {
				return oldBuilder(duration, segLen, ",", func(i int) string {
					return fmt.Sprintf("/hls/audio_%d.ts?path=%s\n", i, key)
				})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			end, _ := PlaylistShape(tc.duration, 0, tc.segLen)
			got := MediaPlaylist(end, tc.segLen, tc.suffix, tc.segURL)
			if want := tc.want(tc.duration, tc.segLen); got != want {
				t.Errorf("playlist drifted from the pre-extraction output:\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// oldBuilder is the playlist loop as all three handlers wrote it before the
// extraction, kept verbatim so the byte-identity test means something.
func oldBuilder(duration float64, segLen int, extinfSuffix string, segURL func(i int) string) string {
	response := "#EXTM3U\n"
	response += "#EXT-X-VERSION:3\n"
	response += "#EXT-X-MEDIA-SEQUENCE:0\n"
	response += "#EXT-X-ALLOW-CACHE:YES\n"
	response += "#EXT-X-PLAYLIST-TYPE:VOD\n"
	response += fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", segLen)
	total := int(math.Ceil(duration / float64(segLen)))
	for i := 0; i < total; i++ {
		response += fmt.Sprintf("#EXTINF:%.4f%s\n", math.Min(
			float64(segLen),
			duration-float64(i*segLen),
		), extinfSuffix)
		response += segURL(i)
	}
	response += "#EXT-X-ENDLIST\n"
	return response
}
