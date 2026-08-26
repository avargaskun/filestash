package ffmpeg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/mickael-kerjean/filestash/server/common"
	"github.com/mickael-kerjean/filestash/server/plugin/plg_video_transcoder/preset"
)

func transcodeVideoSegment(ctx context.Context, cachePath string, segmentNumber int, w io.Writer, quality preset.Preset) error {
	start := segmentNumber * HLS_VIDEO_SEGMENT_LENGTH
	height := quality.Height
	bitrate := fmt.Sprintf("%d", quality.Bitrate)
	maxrate := bitrate
	bufsize := fmt.Sprintf("%d", quality.Bitrate*2)
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-timelimit", "30",
		"-ss", fmt.Sprintf("%d.00", start),
		"-i", cachePath,
		"-t", fmt.Sprintf("%d.00", HLS_VIDEO_SEGMENT_LENGTH),
		"-an", "-sn",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d.000)", HLS_VIDEO_SEGMENT_LENGTH),
		"-fps_mode", "cfr",
		"-output_ts_offset", fmt.Sprintf("%d.00", start),
	}

	switch ENCODER {
	case "h264_vaapi":
		args = append([]string{
			"-init_hw_device", "vaapi=va:/dev/dri/renderD128",
			"-filter_hw_device", "va",
		}, args...)
		args = append(args,
			"-vf", fmt.Sprintf("format=nv12,hwupload,scale_vaapi=w=-2:h=%d", height),
			"-c:v", "h264_vaapi",
			"-b:v", bitrate, "-maxrate", maxrate, "-bufsize", bufsize,
		)
	case "h264_nvenc":
		args = append([]string{
			"-init_hw_device", "cuda=hw",
			"-filter_hw_device", "hw",
		}, args...)
		args = append(args,
			"-vf", fmt.Sprintf("format=nv12,hwupload_cuda,scale_cuda=w=-2:h=%d", height),
			"-c:v", "h264_nvenc",
			"-b:v", bitrate, "-maxrate", maxrate, "-bufsize", bufsize,
		)
	case "h264_v4l2m2m":
		args = append(args,
			"-vf", fmt.Sprintf("scale=-2:%d,format=yuv420p", height),
			"-c:v", "h264_v4l2m2m",
			"-b:v", bitrate,
			"-num_output_buffers", "32",
			"-num_capture_buffers", "32",
		)
	case "libx264":
		args = append(args,
			"-vf", fmt.Sprintf("scale=-2:%d,format=yuv420p", height),
			"-c:v", "libx264", "-preset", "veryfast",
			"-b:v", bitrate, "-maxrate", maxrate, "-bufsize", bufsize,
			"-x264opts", "subme=0:me_range=4:rc_lookahead=10:me=dia:no_chroma_me:8x8dct=0:partitions=none",
		)
	default:
		return ErrNotImplemented
	}

	args = append(args, "-f", "mpegts", "pipe:1")
	return runFFmpeg(ctx, args, w)
}

func transcodeAudioSegment(ctx context.Context, cachePath string, segmentNumber int, w io.Writer) error {
	start := segmentNumber * HLS_AUDIO_SEGMENT_LENGTH
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-timelimit", "30",
		"-ss", fmt.Sprintf("%d.00", start),
		"-i", cachePath,
		"-t", fmt.Sprintf("%d.00", HLS_AUDIO_SEGMENT_LENGTH),
		"-vn", "-sn",
		"-c:a", "aac", "-b:a", strconv.Itoa(AUDIO_BITRATE),
		"-output_ts_offset", fmt.Sprintf("%d.00", start),
		"-f", "mpegts", "pipe:1",
	}
	return runFFmpeg(ctx, args, w)
}

func runFFmpeg(ctx context.Context, args []string, w io.Writer) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = w
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}
	msg, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("ffmpeg: %w: %s", err, msg)
	}
	return nil
}

func probeMedia(path string) (float64, bool, error) {
	out, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration:stream=codec_type",
		"-of", "json",
		path,
	).Output()
	if err != nil {
		return 0, false, fmt.Errorf("ffprobe: %w", err)
	}
	var parsed struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return 0, false, fmt.Errorf("ffprobe parse: %w", err)
	}
	if parsed.Format.Duration == "" {
		return 0, false, errors.New("ffprobe: no duration")
	}
	d, err := strconv.ParseFloat(parsed.Format.Duration, 64)
	if err != nil {
		return 0, false, fmt.Errorf("ffprobe duration: %w", err)
	}
	hasAudio := false
	for _, s := range parsed.Streams {
		if s.CodecType == "audio" {
			hasAudio = true
			break
		}
	}
	return d, hasAudio, nil
}

// probeContentEnd measures where the streams that get transcoded really end,
// which the container's declared duration does not promise. It returns 0 —
// meaning "do not clamp" — on any failure at all.
//
// The type selectors match what the segment encoders do: they pass a bare -i
// with -an/-vn, so ffmpeg's own default stream selection applies, and pinning
// this to v:0/a:0 would measure a stream the encoder never touches. Taking the
// max across the type can only over-report, i.e. fail open.
func probeContentEnd(ctx context.Context, path string, duration float64) float64 {
	if !(duration > 0) || math.IsInf(duration, 0) || math.IsNaN(duration) {
		return 0
	}
	for _, back := range []float64{10, 90, 600} {
		from := duration - back
		if from < 0 {
			from = 0
		}
		best := math.Inf(-1)
		for _, selector := range []string{"V", "a"} {
			out, err := exec.CommandContext(ctx,
				"ffprobe", "-v", "error",
				"-select_streams", selector,
				"-read_intervals", fmt.Sprintf("%f%%", from),
				"-show_entries", "packet=pts_time",
				"-of", "csv=p=0",
				path,
			).Output()
			if err != nil {
				return 0
			}
			for _, line := range strings.Split(string(out), "\n") {
				line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ","))
				if line == "" {
					continue
				}
				// decode order: the running max, never the last line
				if v, err := strconv.ParseFloat(line, 64); err == nil && v > best {
					best = v
				}
			}
		}
		if !math.IsInf(best, -1) {
			return best
		}
		if from == 0 {
			break
		}
	}
	return 0
}

// HasAudio reports whether the source carries an audio stream. An unprobeable
// source is reported as audio-bearing: that keeps the historical playlist shape
// and leaves the failure to the segment encoder.
func HasAudio(path string) bool {
	_, hasAudio, err := probeMedia(path)
	if err != nil {
		return true
	}
	return hasAudio
}
