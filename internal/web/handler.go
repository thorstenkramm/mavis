package web

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
	documentexport "github.com/thorstenkramm/mavis/internal/export"
	"github.com/thorstenkramm/mavis/internal/render"
)

const assetPrefix = "/_mavis/assets/"

// MarkdownRenderer renders one source document.
type MarkdownRenderer interface {
	Render(source []byte, documentPath string, authorize render.AccessFunc) (render.Document, error)
}

// Exporter generates complete document attachments without writing HTTP responses.
type Exporter interface {
	Export(ctx context.Context, resource content.Resource, format documentexport.Format) (documentexport.Result, error)
	DOCXAvailable() bool
}

type webHandler struct {
	editTemplate  config.EditTemplate
	resolver      *content.Resolver
	renderer      MarkdownRenderer
	exporter      Exporter
	files         http.Handler
	assets        fs.FS
	pages         *template.Template
	logger        *log.Logger
	readFile      func(string) ([]byte, error)
	search        func(context.Context, string) (content.SearchResults, error)
	searchTimeout time.Duration
	searches      chan struct{}
}

type pageModel struct {
	EditURL          string
	Title            string
	Navigation       []content.NavigationItem
	Breadcrumbs      []crumbModel
	Search           searchModel
	Download         *downloadModel
	Content          template.HTML
	TOC              []render.TOCItem
	Entries          []entryModel
	FrontMatter      string
	FrontMatterTree  []frontMatterNode
	FrontMatterCount string
	HasFrontMatter   bool
	Viewer           *viewerModel
	ShowListing      bool
	ShowTitle        bool
}

type entryModel struct {
	Name string
	URL  string
	Icon string
	Kind string
}

// crumbModel is one step of the path from the content root to the page.
type crumbModel struct {
	Name    string
	URL     string
	Current bool
}

type downloadModel struct {
	MarkdownURL string
	PDFURL      string
	DOCXURL     string
	RawURL      string
}

// NewHandler constructs the embedded asset and dynamic content routes.
func NewHandler(resolver *content.Resolver, logger *log.Logger) (http.Handler, error) {
	return NewHandlerWithRenderer(resolver, render.NewRenderer(), logger)
}

// NewHandlerWithRenderer constructs routes with an injectable Markdown renderer.
func NewHandlerWithRenderer(resolver *content.Resolver, renderer MarkdownRenderer, logger *log.Logger) (http.Handler, error) {
	return NewHandlerWithExporter(resolver, renderer, nil, logger, config.EditTemplate{})
}

// NewHandlerWithExporter constructs routes with injectable rendering and export services.
func NewHandlerWithExporter(resolver *content.Resolver, renderer MarkdownRenderer, exporter Exporter, logger *log.Logger, edit config.EditTemplate) (http.Handler, error) {
	handler, err := newHandler(resolver, renderer, exporter, logger)
	if err != nil {
		return nil, err
	}
	handler.editTemplate = edit
	return handler.routes(), nil
}

func newHandler(resolver *content.Resolver, renderer MarkdownRenderer, exporter Exporter, logger *log.Logger) (*webHandler, error) {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	pages, err := parsePages()
	if err != nil {
		return nil, err
	}
	assets, err := fs.Sub(embedded, "static")
	if err != nil {
		return nil, err
	}
	return &webHandler{
		resolver: resolver,
		renderer: renderer,
		exporter: exporter,
		files:    content.NewFileHandler(resolver, logger),
		assets:   assets,
		pages:    pages,
		logger:   logger,
		readFile: os.ReadFile,
		search: func(ctx context.Context, query string) (content.SearchResults, error) {
			return resolver.SearchWhere(ctx, query, hasContentRoute)
		},
		searchTimeout: searchTimeout,
		searches:      make(chan struct{}, concurrentSearches),
	}, nil
}

// parsePages loads the page template with the helpers it depends on.
func parsePages() (*template.Template, error) {
	return template.New("page.html").Funcs(template.FuncMap{"icon": iconFor}).ParseFS(embedded, "templates/page.html")
}

// breadcrumbs lists the ancestors of a root-relative path followed by the
// page itself. The content root has no trail.
func breadcrumbs(relativePath string) []crumbModel {
	if relativePath == "" || relativePath == "." {
		return nil
	}
	parts := strings.Split(relativePath, "/")
	crumbs := make([]crumbModel, 0, len(parts))
	for index, part := range parts {
		if index == len(parts)-1 {
			crumbs = append(crumbs, crumbModel{Name: part, Current: true})
			break
		}
		crumbs = append(crumbs, crumbModel{Name: part, URL: content.URLFor(strings.Join(parts[:index+1], "/"), true)})
	}
	return crumbs
}

func (handler *webHandler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET "+assetPrefix, http.StripPrefix(assetPrefix, http.FileServer(http.FS(handler.assets))))
	mux.Handle(navigationPath, http.HandlerFunc(handler.serveNavigation))
	mux.Handle("GET "+searchPath, http.HandlerFunc(handler.serveSearch))
	mux.Handle("/", handler)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		setSecurityHeaders(writer.Header())
		mux.ServeHTTP(writer, request)
	})
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
	download := request.URL.Query().Get("download")
	if download != "" && (download != "1" || request.URL.Query().Get("raw") != "1") {
		http.Error(writer, "Unsupported download format", http.StatusBadRequest)
		return
	}
	if request.URL.Query().Get("raw") != "1" && handler.serveViewer(writer, request, *resource) {
		return
	}
	handler.files.ServeHTTP(writer, request)
}

// serveViewer reports whether it wrote a response. Only text viewers read the
// complete file; other formats are classified from their leading bytes.
func (handler *webHandler) serveViewer(writer http.ResponseWriter, request *http.Request, resource content.Resource) bool {
	sample, err := readViewerSample(resource.Path)
	if err != nil {
		handler.writeContentError(writer, request, resource.RelativePath, err)
		return true
	}
	kind := classifyViewer(resource.Info.Name(), sample)
	if kind == viewerNone {
		return false
	}
	viewer := &viewerModel{
		Kind:   kind,
		Name:   displayName(resource.RelativePath),
		RawURL: content.RawURLFor(resource.RelativePath, request.URL.RawQuery),
	}
	var editURL string
	if kind == viewerText {
		source, err := handler.readFile(resource.Path)
		if err != nil {
			handler.writeContentError(writer, request, resource.RelativePath, err)
			return true
		}
		if !utf8.Valid(source) {
			return false
		}
		viewer.Text = string(source)
		editURL = handler.editTemplate.URL(resource.RelativePath)
	}
	navigation, err := handler.resolver.Navigation(resource.RelativePath)
	if err != nil {
		handler.writeContentError(writer, request, resource.RelativePath, err)
		return true
	}
	handler.writePage(writer, http.StatusOK, pageModel{
		Title:       viewer.Name,
		EditURL:     editURL,
		Navigation:  navigation,
		Breadcrumbs: breadcrumbs(resource.RelativePath),
		Download:    &downloadModel{RawURL: content.URLFor(resource.RelativePath, false) + "?download=1&raw=1"},
		Viewer:      viewer,
	})
	return true
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
	if request.URL.Query().Get("download") != "" {
		http.Error(writer, "Unsupported download format", http.StatusBadRequest)
		return
	}
	navigation, err := handler.resolver.Navigation(resource.RelativePath)
	if err != nil {
		handler.writeContentError(writer, request, resource.RelativePath, err)
		return
	}
	model := pageModel{
		Title:       title,
		Navigation:  navigation,
		Breadcrumbs: breadcrumbs(resource.RelativePath),
		ShowListing: true,
		ShowTitle:   true,
	}
	for _, entry := range directory.Entries {
		name := path.Base(entry.RelativePath)
		model.Entries = append(model.Entries, entryModel{
			Name: name,
			URL:  content.URLFor(entry.RelativePath, entry.Info.IsDir()),
			Icon: iconFor(name, entry.Info.IsDir()),
			Kind: kindFor(name, entry.Info.IsDir()),
		})
	}
	handler.writePage(writer, http.StatusOK, model)
}

func (handler *webHandler) serveMarkdown(writer http.ResponseWriter, request *http.Request, resource content.Resource, navigationPath, title string) {
	switch documentexport.Format(request.URL.Query().Get("download")) {
	case "":
	case "markdown":
		handler.serveOriginalDownload(writer, request, resource)
		return
	case documentexport.PDF, documentexport.DOCX:
		handler.serveGeneratedDownload(writer, request, resource)
		return
	default:
		http.Error(writer, "Unsupported download format", http.StatusBadRequest)
		return
	}
	navigation, err := handler.resolver.Navigation(navigationPath)
	if err != nil {
		handler.writeContentError(writer, request, resource.RelativePath, err)
		return
	}
	source, err := handler.readFile(resource.Path)
	if err != nil {
		handler.writeContentError(writer, request, resource.RelativePath, err)
		return
	}
	document, err := handler.renderer.Render(source, resource.RelativePath, handler.authorize)
	if err != nil {
		handler.logger.Printf("render content %q: %v", resource.RelativePath, err)
		fallback := render.FallbackDocument()
		handler.writePage(writer, http.StatusInternalServerError, pageModel{
			Title:       title,
			Navigation:  navigation,
			Breadcrumbs: breadcrumbs(navigationPath),
			Content:     template.HTML(fallback.HTML),
			TOC:         fallback.TOC,
		})
		return
	}
	if document.Title != "" {
		title = document.Title
	}
	download := &downloadModel{
		MarkdownURL: content.URLFor(resource.RelativePath, false) + "?download=markdown",
		PDFURL:      content.URLFor(resource.RelativePath, false) + "?download=pdf",
	}
	if handler.exporter != nil && handler.exporter.DOCXAvailable() {
		download.DOCXURL = content.URLFor(resource.RelativePath, false) + "?download=docx"
	}
	model := pageModel{
		Title:          title,
		Navigation:     navigation,
		Breadcrumbs:    breadcrumbs(navigationPath),
		Download:       download,
		EditURL:        handler.editTemplate.URL(resource.RelativePath),
		Content:        template.HTML(document.HTML),
		TOC:            document.TOC,
		FrontMatter:    document.FrontMatter,
		HasFrontMatter: document.HasFrontMatter,
	}
	if document.HasFrontMatter {
		if tree, count := parseFrontMatter(document.FrontMatter); count > 0 {
			model.FrontMatterTree = tree
			model.FrontMatterCount = frontMatterCount(count)
		}
	}
	handler.writePage(writer, http.StatusOK, model)
}

func (handler *webHandler) serveGeneratedDownload(writer http.ResponseWriter, request *http.Request, resource content.Resource) {
	if handler.exporter == nil {
		http.Error(writer, "Download format unavailable", http.StatusNotFound)
		return
	}
	format := documentexport.Format(request.URL.Query().Get("download"))
	result, err := handler.exporter.Export(request.Context(), resource, format)
	if err != nil {
		handler.writeExportError(writer, resource.RelativePath, err)
		return
	}
	writer.Header().Set("Content-Type", result.ContentType)
	writer.Header().Set("Content-Disposition", content.AttachmentDisposition(result.Filename))
	writer.Header().Set("Content-Length", strconv.Itoa(len(result.Body)))
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(http.StatusOK)
	if request.Method == http.MethodHead {
		return
	}
	if _, err := writer.Write(result.Body); err != nil {
		handler.logger.Printf("write generated download %q: %v", resource.RelativePath, err)
	}
}

func (handler *webHandler) writeExportError(writer http.ResponseWriter, relativePath string, err error) {
	switch {
	case errors.Is(err, documentexport.ErrUnknownFormat):
		http.Error(writer, "Unsupported download format", http.StatusBadRequest)
	case errors.Is(err, documentexport.ErrUnavailable):
		http.Error(writer, "Download format unavailable", http.StatusNotFound)
	case errors.Is(err, documentexport.ErrSourceTooLarge):
		http.Error(writer, "Export source is too large", http.StatusRequestEntityTooLarge)
	case errors.Is(err, documentexport.ErrTimeout):
		http.Error(writer, "Export timed out", http.StatusServiceUnavailable)
	default:
		handler.logger.Printf("export content %q: %v", relativePath, err)
		http.Error(writer, "Export failed", http.StatusInternalServerError)
	}
}

func (handler *webHandler) serveOriginalDownload(writer http.ResponseWriter, request *http.Request, resource content.Resource) {
	download := request.Clone(request.Context())
	download.URL.Path = "/" + resource.RelativePath
	download.URL.RawPath = content.URLFor(resource.RelativePath, false)
	download.URL.RawQuery = "download=1&raw=1"
	handler.files.ServeHTTP(writer, download)
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
	header.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; script-src 'self'; img-src 'self' http: https:; connect-src 'self'; frame-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
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
