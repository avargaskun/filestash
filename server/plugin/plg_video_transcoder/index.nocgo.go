//go:build !cgo

package plg_video_transcoder

import (
	"os/exec"

	. "github.com/mickael-kerjean/filestash/server/common"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/ffmpeg"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/preset"

	"github.com/gorilla/mux"
)

func isActive() bool {
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			Log.Warning("plg_video_transcoder: %q not found in $PATH, transcoding disabled", bin)
			return false
		}
	}
	return true
}

func servePlaylist(cacheName string, p preset.Preset) string {
	return ffmpeg.MasterPlaylist(cacheName, p)
}

func serveHLSChunks(r *mux.Router) {
	ffmpeg.RegisterRoutes(r, video_encoder(), defaultPresetName())
}
