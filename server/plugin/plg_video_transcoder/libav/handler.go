package libav

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	. "github.com/mickael-kerjean/filestash/server/common"
	. "github.com/mickael-kerjean/filestash/server/middleware"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/hlsmath"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/mediaref"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/preset"

	"github.com/gorilla/mux"
)

// MasterPlaylist declares what the segments will actually contain: a
// video-only source must not advertise AAC, or MSE waits forever for an audio
// buffer that no segment will ever fill.
func MasterPlaylist(cacheName string, p preset.Preset, hasAudio bool) string {
	codecs := VIDEO_CODEC
	bandwidth := p.Bitrate
	if hasAudio {
		codecs += "," + AUDIO_CODEC
		bandwidth += AUDIO_BITRATE
	}
	master := "#EXTM3U\n"
	master += "#EXT-X-VERSION:3\n"
	master += fmt.Sprintf(`#EXT-X-STREAM-INF:BANDWIDTH=%d,CODECS="%s"`+"\n", bandwidth, codecs)
	master += fmt.Sprintf(WithBase("/hls/index.m3u8?path=%s&preset=%s\n"), cacheName, p.Name)
	return master
}

func RegisterRoutes(r *mux.Router, enc string, defaultPreset string) {
	ENCODER = enc
	DEFAULT_PRESET = defaultPreset
	r.PathPrefix(WithBase("/hls/index.m3u8")).Handler(NewMiddlewareChain(
		playlistHandler,
		[]Middleware{SecureHeaders},
	)).Methods("GET")
	r.PathPrefix(WithBase("/hls/segment_{segment}.ts")).Handler(NewMiddlewareChain(
		segmentHandler,
		[]Middleware{SecureHeaders},
	)).Methods("GET")
}

// mediaPath resolves the request's opaque key. These routes run without the
// session middleware, so the key is looked up in the registry, never joined
// onto a directory.
func mediaPath(req *http.Request) (string, string, bool) {
	key := req.URL.Query().Get("path")
	path, ok := mediaref.Resolve(key)
	if ok == false {
		return "", "", false
	}
	if _, err := os.Stat(path); err != nil {
		return "", "", false
	}
	return key, path, true
}

func playlistHandler(ctx *App, res http.ResponseWriter, req *http.Request) {
	quality, ok := preset.FromRequest(req.URL.Query().Get("preset"), DEFAULT_PRESET)
	if ok == false {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	key, path, ok := mediaPath(req)
	if ok == false {
		res.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	duration, contentEnd, _, err := probePlayable(req.Context(), path)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	end, total := hlsmath.PlaylistShape(duration, contentEnd, HLS_SEGMENT_LENGTH)
	if total < hlsmath.TotalSegments(duration, HLS_SEGMENT_LENGTH) {
		Log.Info("plg_video_transcoder::playlist::clamp path=%s container=%.3f content=%.3f segments=%d", path, duration, contentEnd, total)
	}
	response := hlsmath.MediaPlaylist(end, HLS_SEGMENT_LENGTH, ", nodesc", func(i int) string {
		return fmt.Sprintf(WithBase("/hls/segment_%d.ts?path=%s&preset=%s"), i, key, quality.Name)
	})
	res.Header().Set("Content-Type", "application/x-mpegURL")
	res.Write([]byte(response))
}

func segmentHandler(ctx *App, res http.ResponseWriter, req *http.Request) {
	segmentNumber, err := strconv.Atoi(mux.Vars(req)["segment"])
	if err != nil {
		Log.Info("[plugin hls] invalid segment request '%s'", mux.Vars(req)["segment"])
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	quality, ok := preset.FromRequest(req.URL.Query().Get("preset"), DEFAULT_PRESET)
	if ok == false {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	_, path, ok := mediaPath(req)
	if ok == false {
		Log.Info("[plugin hls]: invalid video")
		res.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	res.Header().Set("Content-Type", "video/mp2t")
	if err := transcodeSegment(req.Context(), path, segmentNumber, res, quality.Height, quality.Bitrate); err != nil {
		Log.Error("plg_video_transcoder::segment::run %s", err.Error())
	}
}
