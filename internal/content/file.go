package content

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
)

type openedFile interface {
	io.ReadSeeker
	Stat() (os.FileInfo, error)
	Close() error
}

type fileHandler struct {
	resolver *Resolver
	open     func(string) (openedFile, error)
	logger   *log.Logger
}

// NewFileHandler serves eligible non-directory resources from their resolved paths.
func NewFileHandler(resolver *Resolver, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &fileHandler{
		resolver: resolver,
		open: func(path string) (openedFile, error) {
			return os.Open(path)
		},
		logger: logger,
	}
}

func (handler *fileHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; script-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	resource, err := handler.resolver.Resolve(request.URL.EscapedPath())
	if err != nil {
		handler.writeError(writer, "", err)
		return
	}
	if resource.Info.IsDir() {
		http.NotFound(writer, request)
		return
	}

	file, err := handler.open(resource.Path)
	if err != nil {
		handler.writeError(writer, resource.RelativePath, err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			handler.logger.Printf("close content %q: %v", resource.RelativePath, err)
		}
	}()

	info, err := file.Stat()
	if err != nil {
		handler.writeError(writer, resource.RelativePath, err)
		return
	}
	if !info.Mode().IsRegular() {
		http.NotFound(writer, request)
		return
	}
	pdf, err := hasPDFSignature(file)
	if err != nil {
		handler.writeError(writer, resource.RelativePath, err)
		return
	}
	if pdf {
		writer.Header().Set("Content-Type", "application/pdf")
		writer.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; script-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'self'")
		// Agree with frame-ancestors for clients that only honor the older header.
		writer.Header().Set("X-Frame-Options", "SAMEORIGIN")
	}

	http.ServeContent(writer, request, path.Base(resource.RelativePath), info.ModTime(), file)
}

func hasPDFSignature(file openedFile) (bool, error) {
	prefix := make([]byte, 5)
	read, err := io.ReadFull(file, prefix)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return false, err
	}
	return bytes.Equal(prefix[:read], []byte("%PDF-")), nil
}

func (handler *fileHandler) writeError(writer http.ResponseWriter, relativePath string, err error) {
	if errors.Is(err, ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	handler.logger.Printf("serve content %q: %v", relativePath, err)
	http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
