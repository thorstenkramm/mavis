package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
)

func TestSearchRouteMethodsEmptyQueryHEADAndSecurityHeaders(t *testing.T) {
	root := t.TempDir()
	calls := 0
	handler, _ := newSearchTestHandler(t, root, func(context.Context, string) (content.SearchResults, error) {
		calls++
		return content.SearchResults{}, nil
	})
	response := requestWeb(handler, "/_mavis/search?q=%20%09%20")
	if response.Code != http.StatusOK || calls != 0 || !strings.Contains(response.Body.String(), "Enter a word or phrase") {
		t.Fatalf("empty search = %d, calls %d, body %s", response.Code, calls, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(response.Header().Get("Content-Security-Policy"), "form-action 'self'") {
		t.Fatalf("search headers = %+v", response.Header())
	}

	headRequest := httptest.NewRequest(http.MethodHead, "/_mavis/search", nil)
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, headRequest)
	if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("HEAD search = %d %q, headers %+v", head.Code, head.Body.String(), head.Header())
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		request := httptest.NewRequest(method, "/_mavis/search", nil)
		rejected := httptest.NewRecorder()
		handler.ServeHTTP(rejected, request)
		if rejected.Code != http.StatusMethodNotAllowed || rejected.Header().Get("Allow") != "GET, HEAD" {
			t.Errorf("%s search = %d, Allow %q", method, rejected.Code, rejected.Header().Get("Allow"))
		}
	}
}

func TestShadowedFilesDoNotCountTowardResultLimit(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"_mavis/search", "_mavis/navigation", "_mavis/assets/manual.txt"} {
		writeWebFile(t, root, name, "needle")
	}
	for i := range 100 {
		writeWebFile(t, root, fmt.Sprintf("page%02d.md", i), "needle")
	}
	body := requestWeb(newWebHandler(t, root, render.NewRenderer(), nil), "/_mavis/search?q=needle").Body.String()
	if strings.Count(body, `class="search-result-link"`) != 100 || !strings.Contains(body, `class="search-result-link" href="/page99.md"`) || strings.Contains(body, `class="search-notice"`) {
		t.Fatal("shadowed paths displaced eligible results or triggered a notice")
	}
}

func TestSearchQueryValidationAndCharacterBoundary(t *testing.T) {
	root := t.TempDir()
	calls := 0
	var searched string
	handler, _ := newSearchTestHandler(t, root, func(_ context.Context, query string) (content.SearchResults, error) {
		calls++
		searched = query
		return content.SearchResults{}, nil
	})
	for _, requestPath := range []string{"/_mavis/search?q=%FF", "/_mavis/search?q=" + url.QueryEscape(strings.Repeat("界", 201))} {
		response := requestWeb(handler, requestPath)
		if response.Code != http.StatusBadRequest || strings.Contains(response.Body.String(), root) {
			t.Errorf("GET %q = %d %q", requestPath, response.Code, response.Body.String())
		}
	}
	if calls != 0 {
		t.Fatalf("invalid queries started %d searches", calls)
	}
	response := requestWeb(handler, "/_mavis/search?q="+url.QueryEscape("  "+strings.Repeat("界", 200)+"  "))
	if response.Code != http.StatusOK || calls != 1 || searched != strings.Repeat("界", 200) {
		t.Fatalf("200-character query = %d, calls %d, query length %d", response.Code, calls, len([]rune(searched)))
	}
}

func TestSearchErrorsAreGeneric(t *testing.T) {
	var logs bytes.Buffer
	handler, state := newSearchTestHandler(t, t.TempDir(), func(context.Context, string) (content.SearchResults, error) {
		return content.SearchResults{}, errors.New("private /filesystem/root failure")
	})
	state.logger = log.New(&logs, "", 0)
	response := requestWeb(handler, "/_mavis/search?q=private")
	if response.Code != http.StatusInternalServerError || response.Body.String() != "Search failed\n" ||
		strings.Contains(response.Body.String(), "/filesystem/root") || !strings.Contains(logs.String(), "/filesystem/root") {
		t.Fatalf("root failure = %d %q, logs %q", response.Code, response.Body.String(), logs.String())
	}
}

func TestSearchDeadlineAndCancellation(t *testing.T) {
	t.Run("deadline", func(t *testing.T) {
		handler, state := newSearchTestHandler(t, t.TempDir(), func(ctx context.Context, _ string) (content.SearchResults, error) {
			<-ctx.Done()
			return content.SearchResults{}, ctx.Err()
		})
		state.searchTimeout = 20 * time.Millisecond
		response := requestWeb(handler, "/_mavis/search?q=slow")
		if response.Code != http.StatusServiceUnavailable || response.Body.String() != "Search timed out\n" {
			t.Fatalf("timeout = %d %q", response.Code, response.Body.String())
		}
	})

	t.Run("already canceled", func(t *testing.T) {
		calls := 0
		handler, _ := newSearchTestHandler(t, t.TempDir(), func(context.Context, string) (content.SearchResults, error) {
			calls++
			return content.SearchResults{}, nil
		})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		request := httptest.NewRequest(http.MethodGet, "/_mavis/search?q=cancel", nil).WithContext(ctx)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if calls != 0 || response.Body.Len() != 0 {
			t.Fatalf("canceled search calls = %d, response %d %q", calls, response.Code, response.Body.String())
		}
	})

	t.Run("canceled during scan", func(t *testing.T) {
		started := make(chan struct{})
		handler, _ := newSearchTestHandler(t, t.TempDir(), func(ctx context.Context, _ string) (content.SearchResults, error) {
			close(started)
			<-ctx.Done()
			return content.SearchResults{}, ctx.Err()
		})
		ctx, cancel := context.WithCancel(context.Background())
		request := httptest.NewRequest(http.MethodGet, "/_mavis/search?q=cancel", nil).WithContext(ctx)
		response := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			handler.ServeHTTP(response, request)
			close(done)
		}()
		<-started
		cancel()
		<-done
		if response.Body.Len() != 0 {
			t.Fatalf("canceled response = %d %q", response.Code, response.Body.String())
		}
	})
}

func TestSearchConcurrencyWaitAndDeadline(t *testing.T) {
	started := make(chan struct{}, 3)
	release := make(chan struct{})
	handler, state := newSearchTestHandler(t, t.TempDir(), func(context.Context, string) (content.SearchResults, error) {
		started <- struct{}{}
		<-release
		return content.SearchResults{}, nil
	})
	state.searchTimeout = time.Second
	var wait sync.WaitGroup
	responses := make([]*httptest.ResponseRecorder, 3)
	for index := range responses {
		responses[index] = httptest.NewRecorder()
		wait.Add(1)
		go func() {
			defer wait.Done()
			request := httptest.NewRequest(http.MethodGet, "/_mavis/search?q=bounded", nil)
			handler.ServeHTTP(responses[index], request)
		}()
	}
	<-started
	<-started
	select {
	case <-started:
		t.Fatal("third search started before a worker became available")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	wait.Wait()
	for index, response := range responses {
		if response.Code != http.StatusOK {
			t.Errorf("response %d = %d %q", index, response.Code, response.Body.String())
		}
	}

	started = make(chan struct{}, 3)
	release = make(chan struct{})
	handler, state = newSearchTestHandler(t, t.TempDir(), func(context.Context, string) (content.SearchResults, error) {
		started <- struct{}{}
		<-release
		return content.SearchResults{}, nil
	})
	state.searchTimeout = 40 * time.Millisecond
	for range 2 {
		go func() {
			request := httptest.NewRequest(http.MethodGet, "/_mavis/search?q=occupied", nil)
			handler.ServeHTTP(httptest.NewRecorder(), request)
		}()
	}
	<-started
	<-started
	response := requestWeb(handler, "/_mavis/search?q=waiting")
	if response.Code != http.StatusServiceUnavailable || response.Body.String() != "Search timed out\n" {
		t.Fatalf("queued timeout = %d %q", response.Code, response.Body.String())
	}
	close(release)
}

func TestSearchRouteShadowsContentAndRawCSPRemainsRestricted(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "_mavis/search", "same-named user secret")
	writeWebFile(t, root, "public.txt", "public")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	for _, requestPath := range []string{"/_mavis/search", "/_mavis/search?raw=1"} {
		response := requestWeb(handler, requestPath)
		if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "same-named user secret") ||
			!strings.Contains(response.Body.String(), "Search results") {
			t.Fatalf("GET %q = %d %q", requestPath, response.Code, response.Body.String())
		}
	}
	raw := requestWeb(handler, "/public.txt?raw=1")
	policy := raw.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "form-action 'none'") || strings.Contains(policy, "form-action 'self'") {
		t.Fatalf("raw CSP = %q", policy)
	}
}

func TestSearchResultsAreFilenameLedEscapedAndOutsideNavigation(t *testing.T) {
	root := t.TempDir()
	results := content.SearchResults{Items: []content.SearchResult{
		{
			Name:         "guide.md",
			RelativePath: "space dir/guide & more.md",
			Excerpt: []content.SearchSegment{
				{Text: "before "},
				{Text: "<script>alert(1)</script>", Match: true},
				{Text: " after"},
			},
		},
		{Name: "guide.md", RelativePath: "other/guide.md", Excerpt: []content.SearchSegment{{Text: "second match"}}},
	}}
	handler, _ := newSearchTestHandler(t, root, func(context.Context, string) (content.SearchResults, error) {
		return results, nil
	})
	response := requestWeb(handler, "/_mavis/search?q="+url.QueryEscape("<script>alert(1)</script>"))
	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("search results status = %d: %s", response.Code, body)
	}
	for _, want := range []string{
		`<form class="search-form" action="/_mavis/search" method="get" role="search">`,
		`<label for="documentation-search">Search documentation</label>`,
		`type="search" value="&lt;script&gt;alert(1)&lt;/script&gt;"`,
		`<a class="search-result-link" href="/other/guide.md">guide.md</a>`,
		`<p class="search-result-path">space dir/guide &amp; more.md</p>`,
		`<p class="search-result-path">other/guide.md</p>`,
		`before <mark>&lt;script&gt;alert(1)&lt;/script&gt;</mark> after`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("results do not contain %q: %s", want, body)
		}
	}
	specialURL := template.HTMLEscapeString(content.URLFor("space dir/guide & more.md", false))
	if !strings.Contains(body, `href="`+specialURL+`"`) {
		t.Errorf("special-character result URL %q missing: %s", specialURL, body)
	}
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatalf("source or query created active markup: %s", body)
	}
	searchPosition := strings.Index(body, `id="search-region"`)
	headerPosition := strings.Index(body, `<header class="topbar">`)
	layoutPosition := strings.Index(body, `<div class="layout">`)
	if searchPosition < 0 || headerPosition < searchPosition || layoutPosition < headerPosition {
		t.Fatalf("search region is not above header and layout: %s", body)
	}
	navigation := between(body, `<nav class="sidebar"`, `</nav>`)
	if strings.Contains(navigation, "documentation-search") || strings.Contains(navigation, "search-results") {
		t.Fatalf("search input or results are inside file navigation: %s", navigation)
	}
	main := between(body, `<main class="content"`, `</main>`)
	if !strings.Contains(main, `class="search-results"`) {
		t.Fatalf("results are not in main content: %s", main)
	}
}

func TestSearchResultStatesAndResponsiveAccessibleStyles(t *testing.T) {
	handler, _ := newSearchTestHandler(t, t.TempDir(), func(context.Context, string) (content.SearchResults, error) {
		return content.SearchResults{Truncated: true, Incomplete: true}, nil
	})
	body := requestWeb(handler, "/_mavis/search?q=none").Body.String()
	for _, want := range []string{
		`No results found for <strong>none</strong>.`,
		`Only the first 100 path-ordered results are shown.`,
		`Some files could not be searched, so the results may be incomplete.`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("state message %q missing: %s", want, body)
		}
	}
	stylesheet := readEmbeddedText(t, "static/app.css")
	for _, want := range []string{
		".search-region", "width: 100%", ".search-controls input", "min-width: 0", "min-height: 44px",
		".search-result-link", "overflow-wrap: anywhere", ".search-result-excerpt mark", ".search-notice",
	} {
		if !strings.Contains(stylesheet, want) {
			t.Errorf("search stylesheet does not contain %q", want)
		}
	}
	script := readEmbeddedText(t, "static/app.js")
	if !strings.Contains(script, "searchRegion.inert = mobile.matches && open") {
		t.Fatal("mobile drawer does not make the search region inert")
	}
}

func TestSearchRouteFindsCurrentCompoundSource(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "guide.md", "Vorher Informationssicherheit nachher\n")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	body := requestWeb(handler, "/_mavis/search?q=sicherheit").Body.String()
	if !strings.Contains(body, `href="/guide.md">guide.md</a>`) || !strings.Contains(body, `Informations<mark>sicherheit</mark> nachher`) {
		t.Fatalf("compound result = %s", body)
	}
	writeWebFile(t, root, "guide.md", "Vorher geändert nachher\n")
	body = requestWeb(handler, "/_mavis/search?q=sicherheit").Body.String()
	if strings.Contains(body, `class="search-result-link"`) || !strings.Contains(body, "No results found") {
		t.Fatalf("edited current result = %s", body)
	}
}

func newSearchTestHandler(t *testing.T, root string, search func(context.Context, string) (content.SearchResults, error)) (http.Handler, *webHandler) {
	t.Helper()
	resolver, err := content.NewResolver(root, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler, err := newHandler(resolver, render.NewRenderer(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.search = search
	return handler.routes(), handler
}

func between(source, start, end string) string {
	startPosition := strings.Index(source, start)
	if startPosition < 0 {
		return ""
	}
	remainder := source[startPosition:]
	endPosition := strings.Index(remainder, end)
	if endPosition < 0 {
		return ""
	}
	return remainder[:endPosition+len(end)]
}
