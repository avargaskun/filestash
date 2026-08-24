package ffmpeg

const (
	HLS_VIDEO_SEGMENT_LENGTH = 5
	HLS_AUDIO_SEGMENT_LENGTH = 8 * 10
	AUDIO_BITRATE            = 128000
)

var (
	ENCODER        string = ""
	DEFAULT_PRESET string = ""
)
