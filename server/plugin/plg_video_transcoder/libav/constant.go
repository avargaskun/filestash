package libav

import (
	"runtime"
)

const (
	AUDIO_BITRATE = 128000
)

var (
	HLS_SEGMENT_LENGTH        = 5
	ENCODER            string = ""
	DEFAULT_PRESET     string = ""
)

func init() {
	if runtime.NumCPU() <= 4 && runtime.GOARCH == "arm" { // eg: a raspberry pi
		HLS_SEGMENT_LENGTH = 10
	}
}
