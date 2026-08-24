package plg_video_transcoder

import (
	"fmt"
	"os"

	. "github.com/mickael-kerjean/filestash/server/common"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/preset"
)

var (
	plugin_enable           func() bool
	blacklist_format        func() string
	video_encoder           func() string
	default_preset          func() string
	force_transcode_default func() bool
)

func init() {
	plugin_enable = func() bool {
		return Config.Get("features.video.enable_transcoder").Schema(func(f *FormElement) *FormElement {
			if f == nil {
				f = &FormElement{}
			}
			f.Name = "enable_transcoder"
			f.Type = "enable"
			f.Target = []string{"transcoding_blacklist_format", "transcoding_video_encoder"}
			f.Description = "Enable/Disable on demand video transcoding. The transcoder"
			f.Default = true
			return f
		}).Bool()
	}
	blacklist_format = func() string {
		return Config.Get("features.video.blacklist_format").Schema(func(f *FormElement) *FormElement {
			if f == nil {
				f = &FormElement{}
			}
			f.Id = "transcoding_blacklist_format"
			f.Name = "blacklist_format"
			f.Type = "text"
			f.Description = "Video format that won't be transcoded"
			f.Default = os.Getenv("FEATURE_TRANSCODING_VIDEO_BLACKLIST")
			if f.Default != "" {
				f.Placeholder = fmt.Sprintf("Default: '%s'", f.Default)
			}
			return f
		}).String()
	}
	video_encoder = func() string {
		return Config.Get("features.video.encoder").Schema(func(f *FormElement) *FormElement {
			if f == nil {
				f = &FormElement{}
			}
			f.Id = "transcoding_video_encoder"
			f.Name = "encoder"
			f.Type = "select"
			f.Description = "Video encoder used for on demand HLS transcoding"
			f.Default = "libx264"
			f.Opts = []string{"libx264", "h264_vaapi", "h264_nvenc", "h264_v4l2m2m"}
			return f
		}).String()
	}
	default_preset = func() string {
		return Config.Get("features.video.default_preset").Schema(func(f *FormElement) *FormElement {
			if f == nil {
				f = &FormElement{}
			}
			f.Id = "transcoding_default_preset"
			f.Name = "default_preset"
			f.Type = "select"
			f.Description = "Default quality preset for on demand HLS transcoding"
			f.Default = preset.Default
			f.Opts = preset.Names()
			return f
		}).String()
	}
	force_transcode_default = func() bool {
		return Config.Get("features.video.force_transcode_default").Schema(func(f *FormElement) *FormElement {
			if f == nil {
				f = &FormElement{}
			}
			f.Name = "force_transcode_default"
			f.Type = "boolean"
			f.Description = "Transcode by default even for videos the browser could play directly (Original stays one click away)"
			f.Default = true
			return f
		}).Bool()
	}
}
