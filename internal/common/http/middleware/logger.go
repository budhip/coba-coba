package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"
	"golang.org/x/exp/slices"

	"github.com/labstack/echo/v4"
)

type bodyDumpResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bodyDumpResponseWriter) Flush() {
	err := http.NewResponseController(w.ResponseWriter).Flush()
	if err != nil && errors.Is(err, http.ErrNotSupported) {
		panic(errors.New("response writer flushing is not supported"))
	}
}

func (w *bodyDumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func (w *bodyDumpResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (m *AppMiddleware) parseRequestBody(c echo.Context) []byte {
	var body []byte
	if c.Request().Body != nil {
		body, _ = io.ReadAll(c.Request().Body)
	}
	c.Request().Body = io.NopCloser(bytes.NewBuffer(body))
	return body
}

var sensitiveHeaders = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"set-cookie":    {},
	"x-secret-key":  {},
	// Add other sensitive headers here
}

func (m *AppMiddleware) parseRequestHeader(c echo.Context) []byte {
	headers := make(map[string][]string)
	for k, vals := range c.Request().Header {
		if _, ok := sensitiveHeaders[strings.ToLower(k)]; ok {
			headers[k] = []string{"*****"}
		} else {
			headers[k] = vals
		}
	}

	b, _ := json.Marshal(headers)
	return b
}

func (m *AppMiddleware) getResponseBodyBuffer(c echo.Context) *bytes.Buffer {
	resBody := new(bytes.Buffer)
	mw := io.MultiWriter(c.Response().Writer, resBody)
	writer := &bodyDumpResponseWriter{
		mw,
		c.Response().Writer,
	}
	c.Response().Writer = writer
	return resBody
}

var excludedLogs = []string{
	"/api/health",
	"/metrics",
}

func (m *AppMiddleware) Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if slices.Contains(excludedLogs, c.Path()) {
				return next(c)
			}

			start := time.Now()
			ctx := c.Request().Context()
			req := c.Request()
			res := c.Response()
			reqBody := m.parseRequestBody(c)
			reqHeader := m.parseRequestHeader(c)
			resBodyBuff := m.getResponseBodyBuffer(c)

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			latency := time.Since(start)

			fields := []xlog.Field{
				xlog.String("timestamp", start.String()),
				xlog.String("end_time", start.Add(latency).String()),
				xlog.String("method", req.Method),
				xlog.String("url_path", req.URL.String()),
				xlog.String("request_body", string(reqBody)),
				xlog.String("request_header", string(reqHeader)),
				xlog.Int("status", res.Status),
				xlog.String("response", string(resBodyBuff.Bytes())),
				xlog.String("latency", latency.String()),
				xlog.String("idempotency_key", req.Header.Get("x-idempotency-key")),
			}

			message := fmt.Sprintf("%v %v %v %v", res.Status, req.Method, req.URL.String(), latency)

			switch {
			case res.Status >= 500:
				xlog.Error(ctx, message, fields...)
			case res.Status >= 400:
				xlog.Warn(ctx, message, fields...)
			case res.Status >= 300:
				xlog.Warn(ctx, message, fields...)
			default:
				xlog.Info(ctx, message, fields...)
			}

			return nil
		}
	}
}
