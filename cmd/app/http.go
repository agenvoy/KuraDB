package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/agenvoy/kuradb/internal/api"
	"github.com/agenvoy/kuradb/internal/database"
	"github.com/agenvoy/kuradb/internal/openai"
)

const (
	httpPortMin   = 10000
	httpPortMax   = 65535
	httpBindTries = 10

	httpReadHeaderTimeout = 5 * time.Second
	httpShutdownTimeout   = 5 * time.Second
)

func runHTTP(ctx context.Context, configDir string, reg *database.Registry, perDBs map[string]*database.DB, embedder openai.Embedder, qcache *openai.Cache) {
	ln, port, err := pickListener()
	if err != nil {
		slog.Error("http: pickListener",
			slog.String("error", err.Error()))
		return
	}

	url := fmt.Sprintf("http://%s:%d", "localhost", port)

	endpointPath := filepath.Join(configDir, "endpoint")
	if err := go_pkg_filesystem.WriteFile(endpointPath, url, 0644); err != nil {
		slog.Warn("endpoint: write",
			slog.String("path", endpointPath),
			slog.String("error", err.Error()))
	}

	dbNames := make([]string, 0, len(perDBs))
	for name := range perDBs {
		dbNames = append(dbNames, name)
	}
	// if err := agenvoy.Register(url, dbNames); err != nil {
	// 	slog.Warn("agenvoy.Register",
	// 		slog.String("error", err.Error()))
	// }

	srv := &http.Server{
		Handler:           api.Router(reg, perDBs, embedder, qcache),
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}

	go func() {
		<-ctx.Done()
		// if err := agenvoy.Unregister(); err != nil {
		// 	slog.Warn("agenvoy.Unregister",
		// 		slog.String("error", err.Error()))
		// }
		if err := os.Remove(endpointPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("endpoint: remove",
				slog.String("path", endpointPath),
				slog.String("error", err.Error()))
		}
		shutCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			slog.Warn("http: Shutdown",
				slog.String("error", err.Error()))
		}
	}()

	slog.Info("http server",
		slog.String("url", url))

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http: Serve",
			slog.String("error", err.Error()))
	}
}

func pickListener() (net.Listener, int, error) {
	for range httpBindTries {
		port := httpPortMin + rand.IntN(httpPortMax-httpPortMin+1)
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", port))
		if err == nil {
			return ln, port, nil
		}
	}
	return nil, 0, fmt.Errorf("no free port in [%d, %d] after %d tries", httpPortMin, httpPortMax, httpBindTries)
}
