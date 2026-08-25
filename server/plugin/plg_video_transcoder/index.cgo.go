//go:build cgo

package plg_video_transcoder

import (
	"github.com/gorilla/mux"

	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/libav"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/preset"
)

func isActive() bool {
	return true
}

func servePlaylist(cacheName string, sourcePath string, p preset.Preset) string {
	return libav.MasterPlaylist(cacheName, p, libav.HasAudio(sourcePath))
}

func serveHLSChunks(r *mux.Router) {
	libav.RegisterRoutes(r, video_encoder(), defaultPresetName())
}
