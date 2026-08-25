//go:build cgo

package libav

import _ "embed"

// minimalTS is a tiny, self-contained, valid MPEG-TS: a 0.2 s h264 (High) +
// aac (LC) segment (PAT/PMT + a handful of real samples). It is written verbatim
// as the terminal segment body when a zero-video-frame window makes the muxer
// emit nothing at all — an empty 200 body sends hls.js into a fatal parse-error
// retry storm, whereas a valid TS carrying real samples ends playback cleanly
// (the same behaviour an audio-only degenerate TS already produced). Generated
// with:
//
//	ffmpeg -f lavfi -i "color=c=black:s=64x64:r=25:d=0.2" \
//	       -f lavfi -i anullsrc=channel_layout=mono:sample_rate=48000 -t 0.2 \
//	       -c:v libx264 -profile:v high -level 3.1 -pix_fmt yuv420p -g 1 -b:v 50k \
//	       -c:a aac -b:a 24k -ac 1 -ar 48000 \
//	       -muxpreload 0 -muxdelay 0 -f mpegts minimal.ts
//
//go:embed minimal.ts
var minimalTS []byte
