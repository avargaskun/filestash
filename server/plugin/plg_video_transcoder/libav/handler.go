package libav

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"

	. "github.com/mickael-kerjean/filestash/server/common"
	. "github.com/mickael-kerjean/filestash/server/middleware"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/mediaref"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/preset"

	"github.com/gorilla/mux"
)

func MasterPlaylist(cacheName string, p preset.Preset) string {
	master := "#EXTM3U\n"
	master += "#EXT-X-VERSION:3\n"
	master += fmt.Sprintf(`#EXT-X-STREAM-INF:BANDWIDTH=%d,CODECS="avc1.64001f,mp4a.40.2"`+"\n", p.Bitrate+AUDIO_BITRATE)
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
	duration, err := probeDuration(path)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := "#EXTM3U\n"
	response += "#EXT-X-VERSION:3\n"
	response += "#EXT-X-MEDIA-SEQUENCE:0\n"
	response += "#EXT-X-ALLOW-CACHE:YES\n"
	response += "#EXT-X-PLAYLIST-TYPE:VOD\n"
	response += fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", HLS_SEGMENT_LENGTH)
	total := int(math.Ceil(duration / float64(HLS_SEGMENT_LENGTH)))
	for i := 0; i < total; i++ {
		response += fmt.Sprintf("#EXTINF:%.4f, nodesc\n", math.Min(
			float64(HLS_SEGMENT_LENGTH),
			duration-float64(i*HLS_SEGMENT_LENGTH),
		))
		response += fmt.Sprintf(WithBase("/hls/segment_%d.ts?path=%s&preset=%s\n"), i, key, quality.Name)
	}
	response += "#EXT-X-ENDLIST\n"
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
