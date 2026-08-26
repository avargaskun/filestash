// Package hlsmath holds the playlist arithmetic and text generation shared by
// both transcoder variants, so the logic that ships is the logic CI tests.
package hlsmath

import (
	"fmt"
	"math"
	"strings"
)

// PlaylistShape decides how long the advertised media actually is and how many
// segments that is.
//
// A container's declared duration is not a promise that content exists: a bad
// mux can claim 1484 s while both streams end at 1430 s, and every segment past
// the real end is a window a player can seek into but nothing can fill.
// contentEnd is the measured end of the streams that get transcoded, or 0 when
// it could not be established. The clamp can only ever shorten a playlist:
// anything unusable about contentEnd leaves the container duration in place.
func PlaylistShape(container, contentEnd float64, segLen int) (float64, int) {
	end := container
	if contentEnd > 0 && contentEnd < container {
		end = contentEnd
	}
	return end, TotalSegments(end, segLen)
}

// TotalSegments is the number of segments a playlist of the given length
// advertises. ceil rather than floor+1: floor+1 is degenerate-free for every
// end, but it emits a terminal #EXTINF:0.0000 whenever the end is not an exact
// multiple of the segment length, which is a worse thing to hand a player than
// losing the single frame that sits exactly on a boundary.
func TotalSegments(end float64, segLen int) int {
	if !(end > 0) || math.IsInf(end, 0) || segLen <= 0 {
		return 0
	}
	return int(math.Ceil(end / float64(segLen)))
}

// MediaPlaylist builds a whole VOD media playlist. Both variants' three
// playlist bodies differ only in the #EXTINF suffix and the segment URL, so
// they share this and the headers can never drift apart. segURL returns the
// URL line for a segment index, without its trailing newline.
func MediaPlaylist(end float64, segLen int, extinfSuffix string, segURL func(i int) string) string {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")
	b.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	b.WriteString("#EXT-X-ALLOW-CACHE:YES\n")
	b.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	fmt.Fprintf(&b, "#EXT-X-TARGETDURATION:%d\n", segLen)
	total := TotalSegments(end, segLen)
	for i := 0; i < total; i++ {
		fmt.Fprintf(&b, "#EXTINF:%.4f%s\n", math.Min(
			float64(segLen),
			end-float64(i*segLen),
		), extinfSuffix)
		b.WriteString(segURL(i))
		b.WriteString("\n")
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	return b.String()
}
