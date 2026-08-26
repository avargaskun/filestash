package ffmpeg

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
// video-only source gets neither the AAC codec nor the audio rendition, or MSE
// waits forever for an audio buffer that no segment will ever fill.
func MasterPlaylist(cacheName string, p preset.Preset, hasAudio bool) string {
	codecs := VIDEO_CODEC
	bandwidth := p.Bitrate
	streamInf := ""
	master := "#EXTM3U\n"
	master += "#EXT-X-VERSION:6\n"
	if hasAudio {
		codecs += "," + AUDIO_CODEC
		bandwidth += AUDIO_BITRATE
		streamInf = `,AUDIO="aud"`
		master += fmt.Sprintf(`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="default",DEFAULT=YES,AUTOSELECT=YES,URI="%s"`+"\n", WithBase(fmt.Sprintf("/hls/audio.m3u8?path=%s", cacheName)))
	}
	master += fmt.Sprintf(`#EXT-X-STREAM-INF:BANDWIDTH=%d,CODECS="%s"%s`+"\n", bandwidth, codecs, streamInf)
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
	duration, _, err := probeMedia(path)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	contentEnd := probeContentEnd(req.Context(), path, duration)

	end, total := hlsmath.PlaylistShape(duration, contentEnd, HLS_VIDEO_SEGMENT_LENGTH)
	if total < hlsmath.TotalSegments(duration, HLS_VIDEO_SEGMENT_LENGTH) {
		Log.Info("plg_video_transcoder::playlist::clamp path=%s container=%.3f content=%.3f segments=%d", path, duration, contentEnd, total)
	}
	response := hlsmath.MediaPlaylist(end, HLS_VIDEO_SEGMENT_LENGTH, ", nodesc", func(i int) string {
		return fmt.Sprintf(WithBase("/hls/video_%d.ts?path=%s&preset=%s"), i, key, quality.Name)
	})
	res.Header().Set("Content-Type", "application/x-mpegURL")
	res.Write([]byte(response))
}

func playlistAudioHandler(ctx *App, res http.ResponseWriter, req *http.Request) {
	key, path, ok := mediaPath(req)
	if ok == false {
		res.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	duration, _, err := probeMedia(path)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	// the same max(video, audio) end as the video rendition: HLS wants the audio
	// rendition to cover the video timeline, and an audio segment past the audio
	// end is an empty body here, never a crash
	contentEnd := probeContentEnd(req.Context(), path, duration)

	end, total := hlsmath.PlaylistShape(duration, contentEnd, HLS_AUDIO_SEGMENT_LENGTH)
	if total < hlsmath.TotalSegments(duration, HLS_AUDIO_SEGMENT_LENGTH) {
		Log.Info("plg_video_transcoder::playlist::clamp path=%s container=%.3f content=%.3f segments=%d", path, duration, contentEnd, total)
	}
	response := hlsmath.MediaPlaylist(end, HLS_AUDIO_SEGMENT_LENGTH, ",", func(i int) string {
		return fmt.Sprintf(WithBase("/hls/audio_%d.ts?path=%s"), i, key)
	})
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
