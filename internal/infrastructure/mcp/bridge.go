// Package mcp provides the zero-overhead bridge between Fiber and the mcp-go server.
package mcp

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v3"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/velar/velar-fiber/internal/domain/port"
	"go.uber.org/zap"
)

// Bridge adapts the mcp-go StreamableHTTPServer to Fiber's request/response model.
type Bridge struct {
	server *mcpserver.MCPServer
	http   *mcpserver.StreamableHTTPServer
	log    *zap.Logger
	audit  port.AuditLogger
}

// NewBridge creates the MCP server and registers all toolsets.
func NewBridge(
	name, version string,
	log *zap.Logger,
	audit port.AuditLogger,
	registrars ...func(*mcpserver.MCPServer),
) *Bridge {
	srv := mcpserver.NewMCPServer(
		name,
		version,
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(true, false),
	)

	for _, r := range registrars {
		r(srv)
	}

	httpSrv := mcpserver.NewStreamableHTTPServer(srv)

	return &Bridge{
		server: srv,
		http:   httpSrv,
		log:    log,
		audit:  audit,
	}
}

// FiberHandler returns a fiber.Handler that proxies MCP requests.
func (b *Bridge) FiberHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Observability: Log incoming MCP request
		b.log.Debug("incoming mcp request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
		)

		reqURL := &url.URL{
			Scheme:   "http",
			Host:     c.Hostname(),
			Path:     c.Path(),
			RawQuery: string(c.Request().URI().QueryString()),
		}

		stdReq := &http.Request{
			Method: c.Method(),
			URL:    reqURL,
			Header: make(http.Header),
			Body:   io.NopCloser(bytes.NewReader(c.Body())),
		}

		// Copy headers from Fiber to standard request.
		c.Request().Header.VisitAll(func(k, v []byte) {
			stdReq.Header.Set(string(k), string(v))
		})

		// Fiber v3 uses c.Context() for the fasthttp request context.
		stdReq = stdReq.WithContext(c.Context())

		w := newFiberResponseWriter(c)
		b.http.ServeHTTP(w, stdReq)
		w.flush()

		return nil
	}
}

// fiberResponseWriter implements http.ResponseWriter backed by a Fiber context.
type fiberResponseWriter struct {
	ctx     fiber.Ctx
	headers http.Header
	status  int
	buf     bytes.Buffer
}

func newFiberResponseWriter(c fiber.Ctx) *fiberResponseWriter {
	return &fiberResponseWriter{ctx: c, headers: make(http.Header), status: http.StatusOK}
}

func (w *fiberResponseWriter) Header() http.Header {
	return w.headers
}

func (w *fiberResponseWriter) WriteHeader(status int) {
	w.status = status
	for k, vs := range w.headers {
		for _, v := range vs {
			w.ctx.Set(k, v)
		}
	}
	w.ctx.Status(status)
}

func (w *fiberResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *fiberResponseWriter) flush() {
	_, _ = w.ctx.Write(w.buf.Bytes())
}
