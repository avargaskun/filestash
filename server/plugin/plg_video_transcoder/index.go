package plg_video_transcoder

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/mickael-kerjean/filestash/server/common"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/mediaref"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/preset"

	"github.com/gorilla/mux"
)

const (
	CLEAR_CACHE_AFTER = 12
	VIDEO_CACHE_PATH  = "data/cache/video/"
)

// localBackend is implemented by backends that serve files straight off the
// filesystem the server itself runs on.
type localBackend interface {
	LocalPath(path string) (string, error)
}

//go:embed index.js
var indexJS string

func init() {
	Hooks.Register.Onload(func() {
		blacklist_format()
		video_encoder()
		if !plugin_enable() || !isActive() {
			return
		}

		cachePath := GetAbsolutePath(VIDEO_CACHE_PATH)
		os.RemoveAll(cachePath)
		os.MkdirAll(cachePath, os.ModePerm)

		Hooks.Register.ProcessFileContentBeforeSend(createPlaylist)
		Hooks.Register.HttpEndpoint(func(r *mux.Router) error {
			r.HandleFunc(
				WithBase("/assets/"+BUILD_REF+"/pages/viewerpage/application_video/sources.js"),
				createVideoMap,
			)
			serveHLSChunks(r)
			return nil
		})
	})
}

func createPlaylist(reader io.ReadCloser, ctx *App, res *http.ResponseWriter, req *http.Request) (io.ReadCloser, bool, error) {
	query := req.URL.Query()
	if query.Get("transcode") != "hls" {
		return reader, false, nil
	}
	path := query.Get("path")
	if strings.HasPrefix(GetMimeType(path), "video/") == false {
		return reader, false, nil
	}

	// resolve the quality preset against the server-side whitelist before any
	// transcode is set up; a bogus value is a 400 that never reaches ffmpeg.
	quality, ok := preset.FromRequest(query.Get("preset"), default_preset())
	if ok == false {
		reader.Close()
		return nil, false, NewError("invalid preset", http.StatusBadRequest)
	}

	cacheName := "vid_" + GenerateID(ctx.Session) + "_" + QuickHash(path, 10) + ".dat"
	if sourcePath, ok := localSourcePath(ctx, path); ok {
		// the file is already on a disk the transcoder can seek into: copying it
		// would only delay the first segment by the length of the copy
		if err := mediaref.Register(cacheName, sourcePath, CLEAR_CACHE_AFTER*time.Hour); err == nil {
			reader.Close()
			(*res).Header().Set("Content-Type", "application/x-mpegURL")
			return NewReadCloserFromBytes([]byte(servePlaylist(cacheName, quality))), true, nil
		}
	}

	cachePath := GetAbsolutePath(
		VIDEO_CACHE_PATH,
		cacheName,
	)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		f, err := os.OpenFile(cachePath, os.O_CREATE|os.O_RDWR, os.ModePerm)
		if err != nil {
			return reader, false, err
		}
		io.Copy(f, reader)
		f.Close()
		time.AfterFunc(CLEAR_CACHE_AFTER*time.Hour, func() {
			mediaref.Forget(cacheName)
			os.Remove(cachePath)
		})
	}
	if err := mediaref.Register(cacheName, cachePath, CLEAR_CACHE_AFTER*time.Hour); err != nil {
		return reader, false, err
	}
	reader.Close()

	(*res).Header().Set("Content-Type", "application/x-mpegURL")
	return NewReadCloserFromBytes([]byte(servePlaylist(cacheName, quality))), true, nil
}

// defaultPresetName is the configured default, coerced to a whitelisted name.
func defaultPresetName() string {
	p, _ := preset.FromRequest("", default_preset())
	return p.Name
}

// localSourcePath is the on-disk location of the file this request is for, when
// the session's backend can name one. The path comes from the session's own
// chroot via PathBuilder, never from anything the HLS routes are handed later.
func localSourcePath(ctx *App, path string) (string, bool) {
	backend, ok := ctx.Backend.(localBackend)
	if ok == false {
		return "", false
	}
	fullpath, err := PathBuilder(ctx, path)
	if err != nil {
		return "", false
	}
	sourcePath, err := backend.LocalPath(fullpath)
	if err != nil {
		return "", false
	}
	return sourcePath, true
}

func createVideoMap(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", GetMimeType(req.URL.String()))
	presetsJSON, _ := json.Marshal(preset.Names())
	quality, _ := preset.FromRequest("", default_preset())
	out, err := TmplExec(indexJS, map[string]string{
		"enabled":               fmt.Sprintf("%v", plugin_enable()),
		"blacklist":             blacklist_format(),
		"presets":               string(presetsJSON),
		"defaultPreset":         quality.Name,
		"forceTranscodeDefault": fmt.Sprintf("%v", force_transcode_default()),
	})
	if err != nil {
		return
	}
	res.Write([]byte(out))
}
