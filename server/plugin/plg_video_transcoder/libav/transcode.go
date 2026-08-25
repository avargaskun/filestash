package libav

// #cgo pkg-config: libavformat libavcodec libavfilter libavutil libswscale libswresample
// #include <stdlib.h>
// #include "transcode.h"
import "C"

import (
	"context"
	"fmt"
	"io"
	"runtime/cgo"
	"unsafe"

	. "github.com/mickael-kerjean/filestash/server/common"
)

func init() {
	C.ff_set_log_quiet()
}

//export goWriteCallback
func goWriteCallback(handle C.uintptr_t, buf *C.uint8_t, n C.int) C.int {
	if w, ok := cgo.Handle(handle).Value().(io.Writer); !ok {
		return -1
	} else if _, err := w.Write(C.GoBytes(unsafe.Pointer(buf), n)); err != nil {
		return -1
	}
	return n
}

//export goInterruptCallback
func goInterruptCallback(handle C.uintptr_t) C.int {
	if ctx, ok := cgo.Handle(handle).Value().(context.Context); ok && ctx.Err() != nil {
		return 1
	}
	return 0
}

// countingWriter records how many bytes the C muxer actually wrote to the
// response, so the caller can tell a zero-frame window that produced a valid
// (audio-only) TS from one where the muxer emitted nothing at all.
type countingWriter struct {
	w io.Writer
	n int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}

func transcodeSegment(ctx context.Context, cachePath string, segmentNumber int, w io.Writer, maxHeight int, videoBitrate int) (err error) {
	cw := &countingWriter{w: w}
	h := cgo.NewHandle(io.Writer(cw))
	hctx := cgo.NewHandle(ctx)
	req := C.FFRequest{
		path:          C.CString(cachePath),
		encoder:       C.CString(ENCODER),
		start_sec:     C.int(segmentNumber * HLS_SEGMENT_LENGTH),
		end_sec:       C.int((segmentNumber + 1) * HLS_SEGMENT_LENGTH),
		segment_len:   C.int(HLS_SEGMENT_LENGTH),
		max_height:    C.int(maxHeight),
		video_bitrate: C.int(videoBitrate),
		audio_bitrate: C.int(AUDIO_BITRATE),
		errbuf:        (*C.char)(C.malloc(512)), errlen: 512,
		interrupt: C.uintptr_t(hctx),
	}

	if ret := C.ff_transcode_segment(&req, C.uintptr_t(h)); ret < 0 && ctx.Err() == nil {
		err = fmt.Errorf("%s", C.GoString(req.errbuf))
	} else if ret == C.FF_SEGMENT_NO_FRAMES {
		Log.Info("plg_video_transcoder::segment::no-frames path=%s segment=%d bytes=%d", cachePath, segmentNumber, cw.n)
		if cw.n == 0 {
			if _, werr := cw.Write(minimalTS); werr != nil && err == nil {
				err = fmt.Errorf("write minimal segment: %w", werr)
			}
		}
	}

	C.free(unsafe.Pointer(req.path))
	C.free(unsafe.Pointer(req.encoder))
	C.free(unsafe.Pointer(req.errbuf))
	h.Delete()
	hctx.Delete()
	return err
}

func probeMedia(path string) (float64, bool, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var duration C.double
	var hasAudio C.int
	var errbuf [256]C.char
	if ret := C.ff_probe_media(cPath, &duration, &hasAudio, &errbuf[0], C.int(len(errbuf))); ret < 0 {
		return 0, false, fmt.Errorf("%s", C.GoString(&errbuf[0]))
	}
	return float64(duration), hasAudio != 0, nil
}

// HasAudio reports whether the source carries an audio stream. An unprobeable
// source is reported as audio-bearing: that keeps the historical playlist shape
// and leaves the failure to the segment encoder.
func HasAudio(path string) bool {
	_, hasAudio, err := probeMedia(path)
	if err != nil {
		return true
	}
	return hasAudio
}
