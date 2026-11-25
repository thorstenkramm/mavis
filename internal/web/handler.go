package web

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
)

const assetPrefix = "/_mavis/assets/"

// MarkdownRenderer renders one source document.
type MarkdownRenderer interface {
	Render(source []byte, documentPath string, authorize render.AccessFunc) (render.Document, error)
}

type webHandler struct {
	resolver *content.Resolver
	renderer MarkdownRenderer
	files    http.Handler
	pages    *template.Template
	logger   *log.Logger
}

type pageModel struct {
	Title       string
	Navigation  []content.NavigationItem
	Content     template.HTML
	TOC         []render.TOCItem
	Entries     []entryModel
	ShowListing bool
	ShowTitle   bool
}

type entryModel struct {
	Name string
	URL  string
}

// NewHandler constructs the embedded asset and dynamic content routes.
func NewHandler(resolver *content.Resolver, logger *log.Logger) (http.Handler, error) {
	return NewHandlerWithRenderer(resolver, render.NewRenderer(), logger)
}

// NewHandlerWithRenderer constructs routes with an injectable Markdown renderer.
func NewHandlerWithRenderer(resolver *content.Resolver, renderer MarkdownRenderer, logger *log.Logger) (http.Handler, error) {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	pages, err := template.ParseFS(embedded, "templates/page.html")
	if err != nil {
		return nil, err
	}
	assets, err := fs.Sub(embedded, "static")
	if err != nil {
		return nil, err
	}
	handler := &webHandler{
		resolver: resolver,
		renderer: renderer,
		files:    content.NewFileHandler(resolver, logger),
		pages:    pages,
		logger:   logger,
	}
	mux := http.NewServeMux()
	mux.Handle("GET "+assetPrefix, http.StripPrefix(assetPrefix, http.FileServer(http.FS(assets))))
	mux.Handle("/", handler)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		setSecurityHeaders(writer.Header())
		mux.ServeHTTP(writer, request)
	}), nil
}

func (handler *webHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	resource, err := handler.resolver.Resolve(request.URL.EscapedPath())
	if err != nil {
		handler.writeContentError(writer, request, "", err)
		return
	}
	if resource.Info.IsDir() {
		handler.serveDirectory(writer, request, resource)
		return
	}
	if render.IsMarkdown(resource.Info.Name()) {
		handler.serveMarkdown(writer, request, *resource, resource.RelativePath, displayName(resource.RelativePath))
		return
	}
	handler.files.ServeHTTP(writer, request)
}

func (handler *webHandler) serveDirectory(writer http.ResponseWriter, request *http.Request, resource *content.Resource) {
	directory, err := handler.resolver.Directory(resource.RelativePath)
	if err != nil {
		handler.writeContentError(writer, request, resource.RelativePath, err)
		return
	}
	title := displayName(resource.RelativePath)
	if directory.README != nil {
		handler.serveMarkdown(writer, request, *directory.README, resource.RelativePath, title)
		return
	}
	navigation, err := handler.resolver.Navigation(resource.RelativePath)
	if err != nil {
		handler.writeContentError(writer, request, resource.RelativePath, err)
		return
	}
	model := pageModel{Title: title, Navigation: navigation, ShowListing: true, ShowTitle: true}
	for _, entry := range directory.Entries {
		model.Entries = append(model.Entries, entryModel{
			Name: path.Base(entry.RelativePath),
			URL:  content.URLFor(entry.RelativePath, entry.Info.IsDir()),
		})
	}
	handler.writePage(writer, http.StatusOK, model)
}

func (handler *webHandler) serveMarkdown(writer http.ResponseWriter, request *http.Request, resource content.Resource, navigationPath, title string) {
	navigation, err := handler.resolver.Navigation(navigationPath)
	if err != nil {
		handler.writeContentError(writer, request, resource.RelativePath, err)
		return
	}
	source, err := os.ReadFile(resource.Path)
	if err != nil {
		handler.writeContentError(writer, request, resource.RelativePath, err)
		return
	}
	document, err := handler.renderer.Render(source, resource.RelativePath, handler.authorize)
	if err != nil {
		handler.logger.Printf("render content %q: %v", resource.RelativePath, err)
		fallback := render.FallbackDocument()
		handler.writePage(writer, http.StatusInternalServerError, pageModel{
			Title:      title,
			Navigation: navigation,
			Content:    template.HTML(fallback.HTML),
			TOC:        fallback.TOC,
		})
		return
	}
	handler.writePage(writer, http.StatusOK, pageModel{
		Title:      title,
		Navigation: navigation,
		Content:    template.HTML(document.HTML),
		TOC:        document.TOC,
	})
}

func (handler *webHandler) authorize(rootRelativePath string) bool {
	_, err := handler.resolver.ResolveLogical(rootRelativePath)
	return err == nil
}

func (handler *webHandler) writeContentError(writer http.ResponseWriter, request *http.Request, relativePath string, err error) {
	if errors.Is(err, content.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		handler.writeErrorPage(writer, http.StatusNotFound, "Not found", "<p>The requested content could not be found.</p>")
		return
	}
	handler.logger.Printf("serve content %q: %v", relativePath, err)
	handler.writeErrorPage(writer, http.StatusInternalServerError, "Server error", render.FallbackDocument().HTML)
}

func (handler *webHandler) writeErrorPage(writer http.ResponseWriter, status int, title, contentHTML string) {
	navigation, err := handler.resolver.Navigation("")
	if err != nil {
		handler.logger.Printf("build error-page navigation: %v", err)
	}
	handler.writePage(writer, status, pageModel{
		Title:      title,
		Navigation: navigation,
		Content:    template.HTML(contentHTML),
		ShowTitle:  true,
	})
}

func (handler *webHandler) writePage(writer http.ResponseWriter, status int, model pageModel) {
	var output bytes.Buffer
	if err := handler.pages.Execute(&output, model); err != nil {
		handler.logger.Printf("execute HTML template: %v", err)
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(status)
	if _, err := output.WriteTo(writer); err != nil {
		handler.logger.Printf("write HTML response: %v", err)
	}
}

func setSecurityHeaders(header http.Header) {
	header.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; script-src 'self'; img-src 'self' http: https:; connect-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
}

func displayName(relativePath string) string {
	if relativePath == "" || relativePath == "." {
		return "Documentation"
	}
	return path.Base(relativePath)
}
