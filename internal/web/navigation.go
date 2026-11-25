package web

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"path"

	"github.com/thorstenkramm/mavis/internal/content"
)

const navigationPath = "/_mavis/navigation"

type navigationEntry struct {
	Name         string `json:"name"`
	RelativePath string `json:"path"`
	URL          string `json:"url"`
	Directory    bool   `json:"directory"`
	Icon         string `json:"icon"`
}

type navigationResponse struct {
	Entries []navigationEntry `json:"entries"`
}

func (handler *webHandler) serveNavigation(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	entries, err := handler.resolver.List(request.URL.Query().Get("path"))
	if err != nil {
		if errors.Is(err, content.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		handler.logger.Printf("list navigation: %v", err)
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	response := navigationResponse{Entries: make([]navigationEntry, 0, len(entries))}
	for _, entry := range entries {
		name := path.Base(entry.RelativePath)
		response.Entries = append(response.Entries, navigationEntry{
			Name:         name,
			RelativePath: entry.RelativePath,
			URL:          content.URLFor(entry.RelativePath, entry.Info.IsDir()),
			Directory:    entry.Info.IsDir(),
			Icon:         iconFor(name, entry.Info.IsDir()),
		})
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		handler.logger.Printf("encode navigation: %v", err)
	}
}
