package middleware

import (
	"bytes"
	"net/http"
	"strconv"

	. "github.com/mickael-kerjean/filestash/server/pkg/config"
	. "github.com/mickael-kerjean/filestash/server/pkg/core"
	. "github.com/mickael-kerjean/filestash/server/pkg/utils"
)

// This file used to double as a telemetry uploader: every request was recorded
// with its full URI, client IP, user agent, referer, backend, share id and a
// salted session hash, and batches were POSTed to a vendor endpoint whenever
// `log.telemetry` was on. This fork drops the uploader and the config knob with
// it, so no config change can turn it back on - see README.md. What is left is
// the local access log.
type LogEntry struct {
	Method     string
	RequestURI string
	Status     int
	Duration   int
	RequestID  string
}

func logger(ctx *App, res *ResponseWriter, req *http.Request) {
	if req.RequestURI == "/about" {
		return
	}
	if Config.Get("log.enable").Bool() == false {
		return
	}
	point := LogEntry{
		Method:     req.Method,
		RequestURI: req.RequestURI,
		Status:     res.status,
		Duration:   int((now() - res.start) / (100_000)),
		RequestID:  res.Header().Get("X-Request-Id"),
	}
	var (
		arr [512]byte
		num [32]byte
	)
	buf := bytes.NewBuffer(arr[:0])
	buf.WriteString("HTTP ")
	buf.Write(strconv.AppendInt(num[:0], int64(point.Status), 10))
	buf.WriteByte(' ')
	buf.WriteString(point.Method)
	buf.WriteByte(' ')
	buf.Write(strconv.AppendInt(num[:0], int64(point.Duration/10), 10))
	buf.WriteByte('.')
	buf.Write(strconv.AppendInt(num[:0], int64(point.Duration%10), 10))
	buf.WriteString("ms ")
	if uri := point.RequestURI; len(uri) > 200 {
		buf.WriteString(uri[:200])
		buf.WriteString("...")
	} else {
		buf.WriteString(uri)
	}
	buf.WriteByte(' ')
	if point.RequestID != "" && Config.Get("log.level").String() == "DEBUG" {
		buf.WriteString("trace=")
		buf.WriteString(point.RequestID)
	}
	Log.Raw(buf.Bytes())
}
