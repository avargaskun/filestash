#ifndef FILESTASH_TRANSCODE_H
#define FILESTASH_TRANSCODE_H

#include <stdint.h>

typedef struct {
	const char *path;
	const char *encoder;
	int start_sec;
	int end_sec;
	int segment_len;
	int max_height;
	int video_bitrate;
	int audio_bitrate;
	char *errbuf;
	int errlen;
	uintptr_t interrupt;
} FFRequest;

// ff_transcode_segment returns 0 on success, a negative AVERROR on failure, and
// FF_SEGMENT_NO_FRAMES (a non-negative sentinel, so the Go negative=error
// contract is untouched) when the requested window held no video frame — a
// legitimate outcome for a final sliver / out-of-range segment, not an error.
#define FF_SEGMENT_NO_FRAMES 1

int ff_transcode_segment(const FFRequest *req, uintptr_t writer);

int ff_probe_media(const char *path, double *duration, int *has_audio, char *errbuf, int errlen);

void ff_set_log_quiet(void);

#endif
