package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/thorstenkramm/mavis/internal/content"
)

const (
	searchPath         = "/_mavis/search"
	searchTimeout      = 10 * time.Second
	concurrentSearches = 2
	maxQueryCharacters = 200
)

type searchModel struct {
	Query      string
	Active     bool
	HasQuery   bool
	Results    []searchResultModel
	Truncated  bool
	Incomplete bool
}

type searchResultModel struct {
	Name         string
	RelativePath string
	URL          string
	Excerpt      []content.SearchSegment
}

func (handler *webHandler) serveSearch(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	rawQuery := request.URL.Query().Get("q")
	if !utf8.ValidString(rawQuery) {
		http.Error(writer, "Invalid search query", http.StatusBadRequest)
		return
	}
	query := strings.TrimSpace(rawQuery)
	if utf8.RuneCountInString(query) > maxQueryCharacters {
		http.Error(writer, "Search query is too long", http.StatusBadRequest)
		return
	}
	model := searchModel{Query: query, Active: true, HasQuery: query != ""}
	if query != "" {
		results, err := handler.runSearch(request.Context(), query)
		if err != nil {
			handler.writeSearchError(writer, request, err)
			return
		}
		model.Results = make([]searchResultModel, 0, len(results.Items))
		for _, result := range results.Items {
			model.Results = append(model.Results, searchResultModel{
				Name:         result.Name,
				RelativePath: result.RelativePath,
				URL:          content.URLFor(result.RelativePath, false),
				Excerpt:      result.Excerpt,
			})
		}
		model.Truncated = results.Truncated
		model.Incomplete = results.Incomplete
	}

	navigation, err := handler.resolver.Navigation("")
	if err != nil {
		handler.logger.Printf("build search navigation: %v", err)
		http.Error(writer, "Search failed", http.StatusInternalServerError)
		return
	}
	if request.Method == http.MethodHead {
		writer = &headResponseWriter{ResponseWriter: writer}
	}
	handler.writePage(writer, http.StatusOK, pageModel{
		Title:      "Search documentation",
		Navigation: navigation,
		Search:     model,
	})
}

func (handler *webHandler) runSearch(parent context.Context, query string) (content.SearchResults, error) {
	if err := parent.Err(); err != nil {
		return content.SearchResults{}, err
	}
	ctx, cancel := context.WithTimeout(parent, handler.searchTimeout)
	defer cancel()
	select {
	case handler.searches <- struct{}{}:
		defer func() { <-handler.searches }()
	case <-ctx.Done():
		return content.SearchResults{}, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return content.SearchResults{}, err
	}
	results, err := handler.search(ctx, query)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return content.SearchResults{}, ctxErr
	}
	return results, err
}

func (handler *webHandler) writeSearchError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, context.Canceled) || request.Context().Err() != nil && !errors.Is(err, context.DeadlineExceeded):
		return
	case errors.Is(err, context.DeadlineExceeded):
		http.Error(writer, "Search timed out", http.StatusServiceUnavailable)
	default:
		handler.logger.Printf("search content: %v", err)
		http.Error(writer, "Search failed", http.StatusInternalServerError)
	}
}

type headResponseWriter struct {
	http.ResponseWriter
}

func (writer *headResponseWriter) Write(body []byte) (int, error) {
	return len(body), nil
}
