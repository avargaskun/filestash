package ffmpeg

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
	master += "#EXT-X-VERSION:6\n"
	master += fmt.Sprintf(`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="default",DEFAULT=YES,AUTOSELECT=YES,URI="%s"`+"\n", WithBase(fmt.Sprintf("/hls/audio.m3u8?path=%s", cacheName)))
	master += fmt.Sprintf(`#EXT-X-STREAM-INF:BANDWIDTH=%d,CODECS="avc1.64001f,mp4a.40.2",AUDIO="aud"`+"\n", p.Bitrate+AUDIO_BITRATE)
	master += fmt.Sprintf(WithBase("/hls/video.m3u8?path=%s&preset=%s\n"), cacheName, p.Name)
	return master
}

func RegisterRoutes(r *mux.Router, enc string, defaultPreset string) {
	ENCODER = enc
	DEFAULT_PRESET = defaultPreset
	r.PathPrefix(WithBase("/hls/audio.m3u8")).Handler(NewMiddlewareChain(
		playlistAudioHandler,
		[]Middleware{SecureHeaders},
	)).Methods("GET")
	r.PathPrefix(WithBase("/hls/video.m3u8")).Handler(NewMiddlewareChain(
		playlistVideoHandler,
		[]Middleware{SecureHeaders},
	)).Methods("GET")
	r.PathPrefix(WithBase("/hls/video_{segment}.ts")).Handler(NewMiddlewareChain(
		hlsVideoHandler,
		[]Middleware{SecureHeaders},
	)).Methods("GET")
	r.PathPrefix(WithBase("/hls/audio_{segment}.ts")).Handler(NewMiddlewareChain(
		hlsAudioHandler,
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

func playlistVideoHandler(ctx *App, res http.ResponseWriter, req *http.Request) {
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
	response += fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", HLS_VIDEO_SEGMENT_LENGTH)
	total := int(math.Ceil(duration / float64(HLS_VIDEO_SEGMENT_LENGTH)))
	for i := 0; i < total; i++ {
		response += fmt.Sprintf("#EXTINF:%.4f, nodesc\n", math.Min(
			float64(HLS_VIDEO_SEGMENT_LENGTH),
			duration-float64(i*HLS_VIDEO_SEGMENT_LENGTH),
		))
		response += fmt.Sprintf(WithBase("/hls/video_%d.ts?path=%s&preset=%s\n"), i, key, quality.Name)
	}
	response += "#EXT-X-ENDLIST\n"
	res.Header().Set("Content-Type", "application/x-mpegURL")
	res.Write([]byte(response))
}

func playlistAudioHandler(ctx *App, res http.ResponseWriter, req *http.Request) {
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
	response += fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", HLS_AUDIO_SEGMENT_LENGTH)
	total := int(math.Ceil(duration / float64(HLS_AUDIO_SEGMENT_LENGTH)))
	for i := 0; i < total; i++ {
		response += fmt.Sprintf("#EXTINF:%.4f,\n", math.Min(
			float64(HLS_AUDIO_SEGMENT_LENGTH),
			duration-float64(i*HLS_AUDIO_SEGMENT_LENGTH),
		))
		response += fmt.Sprintf(WithBase("/hls/audio_%d.ts?path=%s\n"), i, key)
	}
	response += "#EXT-X-ENDLIST\n"
	res.Header().Set("Content-Type", "application/x-mpegURL")
	res.Write([]byte(response))
}

func hlsAudioHandler(ctx *App, res http.ResponseWriter, req *http.Request) {
	segmentNumber, err := strconv.Atoi(mux.Vars(req)["segment"])
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	_, path, ok := mediaPath(req)
	if ok == false {
		res.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	res.Header().Set("Content-Type", "video/mp2t")
	if err := transcodeAudioSegment(req.Context(), path, segmentNumber, res); err != nil {
		Log.Error("plg_video_transcoder::audio::run %s", err.Error())
	}
}

func hlsVideoHandler(ctx *App, res http.ResponseWriter, req *http.Request) {
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
	if err := transcodeVideoSegment(req.Context(), path, segmentNumber, res, quality); err != nil {
		Log.Error("plg_video_transcoder::video::run %s", err.Error())
	}
}
