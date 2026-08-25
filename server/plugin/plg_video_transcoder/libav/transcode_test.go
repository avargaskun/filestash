package libav

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// testdata/degenerate.mkv is a 33 KB fixture generated with:
//
//	ffmpeg -f lavfi -i "testsrc2=duration=3:size=128x72:rate=5" \
//	       -f lavfi -i "sine=frequency=440:duration=5.2" \
//	       -c:v libx264 -preset veryfast -g 25 -pix_fmt yuv420p -b:v 40k \
//	       -c:a aac -b:a 24k -ac 1 degenerate.mkv
//
// The video ends at v_last=2.8 s while the container runs to 5.223 s (audio
// tail), so the playlist emits 2 HLS segments and the final in-range one,
// segment 1 = [5,10) s, holds ZERO video frames — the exact F2 crash path
// (the lazily-built video filter graph is never created, s->src stays NULL,
// and the old flush-loop push_filter(s, NULL) dereferenced it → SIGSEGV that
// killed the whole server process).
//
// This package needs cgo + the libav headers and an available libx264/aac
// encoder, so it is NOT part of the CI test job; run it inside the build image
// (docker build --target builder_backend). Cases skip gracefully when the
// encoder is unavailable.
func TestTranscodeSegmentZeroFrameWindow(t *testing.T) {
	ENCODER = "libx264"
	fixture := filepath.Join("testdata", "degenerate.mkv")

	transcode := func(segment int) (int, error) {
		var buf bytes.Buffer
		err := transcodeSegment(context.Background(), fixture, segment, &buf, 480, 1500000)
		return buf.Len(), err
	}

	// Normal window (segment 0, [0,5) s) — has video frames. This also probes
	// encoder availability: an encoder-not-found error means skip, not fail.
	n0, err := transcode(0)
	if err != nil {
		if isEncoderUnavailable(err) {
			t.Skipf("libx264/aac encoder unavailable in this environment: %v", err)
		}
		t.Fatalf("normal segment 0 failed: %v", err)
	}
	if n0 == 0 {
		t.Fatalf("normal segment 0 produced an empty body; expected encoded packets")
	}

	// Degenerate in-range window (segment 1, [5,10) s) — no video frame. Pre-fix
	// this SIGSEGV'd and took the process (hence this test binary) down; post-fix
	// it must return without error and without crashing.
	n1, err := transcode(1)
	if err != nil {
		t.Fatalf("degenerate in-range segment 1 returned an error (expected clean success): %v", err)
	}
	t.Logf("degenerate in-range segment 1: body=%d bytes (0 => muxer emitted nothing)", n1)

	// Out-of-range segment index — degenerate by construction (no frame can be
	// near [250,255) s). Same crash path; must survive.
	n50, err := transcode(50)
	if err != nil {
		t.Fatalf("out-of-range segment 50 returned an error (expected clean success): %v", err)
	}
	t.Logf("out-of-range segment 50: body=%d bytes", n50)

	// Reaching here at all proves the guard held: a NULL-deref SIGSEGV inside
	// cgo cannot be recovered and would have aborted the test process.
}

func isEncoderUnavailable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not implemented")
}
