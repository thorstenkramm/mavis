// Package server provides Mavis listeners and HTTP serving.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
	documentexport "github.com/thorstenkramm/mavis/internal/export"
	"github.com/thorstenkramm/mavis/internal/export/docx"
	pdfexport "github.com/thorstenkramm/mavis/internal/export/pdf"
	"github.com/thorstenkramm/mavis/internal/render"
	"github.com/thorstenkramm/mavis/internal/web"
)

// Serve starts a serving session and blocks until it stops.
func Serve(ctx context.Context, root string, configuration config.Config, stdout, stderr io.Writer) (returnErr error) {
	exportDependencies, err := ResolveExportDependencies(configuration.Downloads)
	if err != nil {
		return err
	}
	resolver, err := content.NewResolver(root, configuration)
	if err != nil {
		return err
	}
	handler, err := newHTTPHandler(resolver, log.New(stderr, "", log.LstdFlags), exportDependencies)
	if err != nil {
		return fmt.Errorf("initialize HTTP handler: %w", err)
	}

	loopback := true
	var listener net.Listener
	if configuration.Server.Socket != "" {
		listener, err = ListenUnix(configuration.Server.Socket)
	} else {
		loopback, err = IsLoopbackHost(configuration.Server.Host)
		if err == nil {
			listener, err = ListenTCP(configuration.Server.Host, configuration.Server.Port)
		}
	}
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) && returnErr == nil {
			returnErr = fmt.Errorf("close listener: %w", closeErr)
		}
	}()

	if configuration.Server.Socket != "" {
		if _, err := fmt.Fprintf(stdout, "Socket: %s\nRoot: %s\n", configuration.Server.Socket, root); err != nil {
			return fmt.Errorf("write startup output: %w", err)
		}
	} else {
		handler = guardTCPAuthority(handler, configuration.Server.Host, listener.Addr())
		if !loopback {
			if _, err := fmt.Fprintln(stderr, "WARNING: Anyone with network access may read the served files; authentication is not enabled."); err != nil {
				return fmt.Errorf("write network warning: %w", err)
			}
		}
		if _, err := fmt.Fprintf(stdout, "Serving: %s\nRoot: %s\n", TCPURL(configuration.Server, listener.Addr()), root); err != nil {
			return fmt.Errorf("write startup output: %w", err)
		}
	}

	return serveHTTP(ctx, listener, handler, 5*time.Second)
}

func newHTTPHandler(resolver *content.Resolver, logger *log.Logger, dependencies ExportDependencies) (http.Handler, error) {
	renderer := render.NewRenderer()
	pdfConverter, err := pdfexport.New(resolver, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize PDF export: %w", err)
	}
	var docxConverter documentexport.Converter
	if dependencies.PandocExecutable != "" {
		docxConverter = docx.New(dependencies.PandocExecutable, resolver, renderer, logger)
	}
	exporter := documentexport.NewService(pdfConverter, docxConverter)
	return web.NewHandlerWithExporter(resolver, renderer, exporter, logger)
}

func serveHTTP(ctx context.Context, listener net.Listener, handler http.Handler, shutdownTimeout time.Duration) error {
	if ctx.Err() != nil {
		return nil
	}
	httpServer := &http.Server{Handler: handler}
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- httpServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			closeErr := httpServer.Close()
			serveErr := <-serveErrors
			if !errors.Is(serveErr, http.ErrServerClosed) {
				closeErr = errors.Join(closeErr, serveErr)
			}
			return errors.Join(fmt.Errorf("graceful HTTP shutdown: %w", err), closeErr)
		}
		if err := <-serveErrors; !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
}
