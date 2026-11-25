//go:build ignore

// smoke-release verifies a native Mavis binary through its public CLI and HTTP interface.
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	if strings.HasPrefix(filepath.Base(os.Args[0]), "pandoc") {
		if err := fakePandoc(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	binary := flag.String("binary", "", "path to the native Mavis binary")
	flag.Parse()
	if err := smokeTest(*binary); err != nil {
		fmt.Fprintln(os.Stderr, "release smoke test:", err)
		os.Exit(1)
	}
	fmt.Println("Release smoke test passed:", *binary)
}

func smokeTest(binary string) (result error) {
	if binary == "" {
		return fmt.Errorf("-binary is required")
	}
	root, err := os.MkdirTemp("", "mavis-release-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	writeFixture(root, "README.md", "---\ntitle: <Release Metadata>\n---\n# Release Smoke\n\n- [x] Packaged\n\n| Feature | Status |\n| --- | --- |\n| Rendering | Works |\n")
	writeFixture(root, "public.txt", "public content\n")
	writeFixture(root, "search/guide.md", "Before context Informationssicherheit after <script>alert(1)</script> ending\n")
	writeFixture(root, ".secret.md", "secret Informationssicherheit hidden secret\n")
	writeFixture(root, "drafts/private.md", "private Informationssicherheit excluded secret\n")
	writeFixture(root, "drafts/private.txt", "private text Informationssicherheit excluded secret\n")
	writeFixture(root, ".mavis.conf", "# Informationssicherheit private configuration\nexclude = [\"drafts/**\"]\ninclude_hidden = false\n\n[server]\nhost = \"127.0.0.1\"\nport = 1\n")
	writeFixture(root, "Über uns.md", "# Unicode filename\n")

	fakePandocDir, err := installFakePandoc()
	if err != nil {
		return err
	}
	defer os.RemoveAll(fakePandocDir)
	if err := checkDOCXOptOut(binary, root, fakePandocDir); err != nil {
		return err
	}

	port, err := availablePort()
	if err != nil {
		return err
	}
	command := exec.Command(binary, "serve", root, "--port", strconv.Itoa(port))
	command.Env = replaceEnvironment(os.Environ(), "PATH", fakePandocDir)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return fmt.Errorf("start binary: %w", err)
	}
	done, lifetime := trackServer(command)
	defer func() {
		if lifetime.Err() == nil {
			_ = command.Process.Kill()
		}
		<-lifetime.Done() // Wait has joined the os/exec output-copy goroutines.
		if result != nil {
			result = fmt.Errorf("%w; stdout=%q stderr=%q", result, stdout.String(), stderr.String())
		} else if !strings.Contains(stdout.String(), "Serving:") || !strings.Contains(stdout.String(), "Root: "+root) {
			result = fmt.Errorf("startup output is incomplete: %q", stdout.String())
		}
	}()

	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	if err := waitForServer(lifetime, baseURL); err != nil {
		return fmt.Errorf("startup failed: %w", err)
	}
	body, err := request(lifetime, baseURL+"/", http.StatusOK)
	if err != nil {
		return err
	}
	for _, expected := range []string{
		`<title>Release Smoke · Mavis</title>`, `<svg class="brand-mark"`, `id="theme-toggle"`,
		`<h1 id="release-smoke">Release Smoke</h1>`, "<table>", `aria-label="Packaged"`,
		`<details class="front-matter"><summary><span class="front-matter-title">Front matter</span>`,
		`<span class="front-matter-key">title</span><span class="front-matter-value">&lt;Release Metadata&gt;</span>`,
		`<code class="language-yaml">title: &lt;Release Metadata&gt;`,
		`<details class="download-menu">`, `href="/README.md?download=docx"`,
	} {
		if !strings.Contains(body, expected) {
			return fmt.Errorf("rendered README does not contain %q", expected)
		}
	}
	for _, unexpected := range []string{`<details class="front-matter" open>`, `href="#title-release-metadata"`, "drafts/private"} {
		if strings.Contains(body, unexpected) {
			return fmt.Errorf("rendered README unexpectedly contains %q", unexpected)
		}
	}
	if err := checkSearch(lifetime, baseURL, root, body); err != nil {
		return err
	}
	body, err = request(lifetime, baseURL+"/public.txt", http.StatusOK)
	if err != nil {
		return err
	}
	for _, expected := range []string{
		`<pre class="text-viewer"><code>public content`, `href="/public.txt?raw=1">View raw file</a>`,
		`aria-label="Documentation files"`,
	} {
		if !strings.Contains(body, expected) {
			return fmt.Errorf("text viewer does not contain %q", expected)
		}
	}
	body, err = request(lifetime, baseURL+"/public.txt?raw=1", http.StatusOK)
	if err != nil || body != "public content\n" {
		return fmt.Errorf("raw public file response = %q, %v", body, err)
	}

	original, header, err := requestResponse(lifetime, baseURL+"/README.md?download=markdown", http.StatusOK)
	expectedSource := "---\ntitle: <Release Metadata>\n---\n# Release Smoke\n\n- [x] Packaged\n\n| Feature | Status |\n| --- | --- |\n| Rendering | Works |\n"
	if err != nil || original != expectedSource {
		return fmt.Errorf("original Markdown download differs: %v", err)
	}
	if err := checkAttachment(header, "README.md", "text/markdown; charset=utf-8"); err != nil {
		return err
	}
	attachment, header, err := requestResponse(lifetime, baseURL+"/public.txt?raw=1&download=1", http.StatusOK)
	if err != nil || attachment != "public content\n" {
		return fmt.Errorf("raw attachment response = %q, %v", attachment, err)
	}
	if err := checkAttachment(header, "public.txt", "text/plain; charset=utf-8"); err != nil {
		return err
	}
	pdf, header, err := requestResponse(lifetime, baseURL+"/README.md?download=pdf", http.StatusOK)
	if err != nil || !strings.HasPrefix(pdf, "%PDF-") {
		return fmt.Errorf("generated PDF is invalid: %v", err)
	}
	if err := checkAttachment(header, "README.pdf", "application/pdf"); err != nil {
		return err
	}
	docx, header, err := requestResponse(lifetime, baseURL+"/README.md?download=docx", http.StatusOK)
	if err != nil || !validDOCX([]byte(docx)) {
		return fmt.Errorf("generated DOCX is invalid: %v", err)
	}
	if err := checkAttachment(header, "README.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"); err != nil {
		return err
	}
	unicodePath := "/" + url.PathEscape("Über uns.md") + "?download=markdown"
	_, header, err = requestResponse(lifetime, baseURL+unicodePath, http.StatusOK)
	if err != nil {
		return err
	}
	if err := checkAttachment(header, "Über uns.md", "text/markdown; charset=utf-8"); err != nil {
		return err
	}

	for _, path := range []string{"/.mavis.conf", "/.secret.md", "/drafts/private.md", "/drafts/private.txt", "/%2e%2e/outside.txt"} {
		for _, suffix := range []string{"", "?raw=1", "?download=markdown", "?download=pdf", "?download=docx", "?raw=1&download=1"} {
			if _, err := request(lifetime, baseURL+path+suffix, http.StatusNotFound); err != nil {
				return fmt.Errorf("content boundary %s%s: %w", path, suffix, err)
			}
		}
	}
	if runtime.GOOS == "windows" {
		if err := checkWindowsSocket(binary); err != nil {
			return err
		}
	}
	return stopServer(command, done)
}

func checkSearch(ctx context.Context, baseURL, root, home string) error {
	searchPosition := strings.Index(home, `id="search-region"`)
	headerPosition := strings.Index(home, `<header class="topbar">`)
	layoutPosition := strings.Index(home, `<div class="layout">`)
	if headerPosition < 0 || searchPosition < headerPosition || layoutPosition < searchPosition {
		return fmt.Errorf("search field is not in the header above the layout")
	}
	navigationStart := strings.Index(home, `<nav class="sidebar"`)
	navigationEnd := strings.Index(home, `</nav>`)
	if navigationStart < 0 || navigationEnd < navigationStart || strings.Contains(home[navigationStart:navigationEnd], "documentation-search") {
		return fmt.Errorf("search input is missing or appears inside the file browser")
	}

	body, err := request(ctx, baseURL+"/_mavis/search?q=sicherheit", http.StatusOK)
	if err != nil {
		return fmt.Errorf("compound search: %w", err)
	}
	for _, expected := range []string{
		`value="sicherheit"`,
		`<a class="search-result-link" href="/search/guide.md">guide.md</a>`,
		`<p class="search-result-path">search/guide.md</p>`,
		`Before context Informations<mark>sicherheit</mark> after &lt;script&gt;alert(1)&lt;/script&gt; ending`,
	} {
		if !strings.Contains(body, expected) {
			return fmt.Errorf("search results do not contain %q", expected)
		}
	}
	for _, unexpected := range []string{
		`<script>alert(1)</script>`, ".secret.md", "drafts/private", ".mavis.conf",
		"hidden secret", "excluded secret", "private configuration",
	} {
		if strings.Contains(body, unexpected) {
			return fmt.Errorf("search results disclose %q", unexpected)
		}
	}

	writeFixture(root, "search/guide.md", "After edit Vertrauenswürdig current source\n")
	body, err = request(ctx, baseURL+"/_mavis/search?q=sicherheit", http.StatusOK)
	if err != nil || !strings.Contains(body, "No results found") || strings.Contains(body, `class="search-result-link"`) {
		return fmt.Errorf("search did not reflect the current edit: %v", err)
	}
	body, err = request(ctx, baseURL+"/_mavis/search?q="+url.QueryEscape("vertrauenswürdig"), http.StatusOK)
	if err != nil || !strings.Contains(body, `After edit <mark>Vertrauenswürdig</mark> current source`) {
		return fmt.Errorf("search did not find the edited source: %v", err)
	}
	return nil
}

func checkDOCXOptOut(binary, root, executableDirectory string) (result error) {
	configurationPath := filepath.Join(root, ".mavis.conf")
	configuration, err := os.ReadFile(configurationPath)
	if err != nil {
		return err
	}
	disabled := append(configuration, []byte("\n[downloads]\nexport_docx = false\n")...)
	if err := os.WriteFile(configurationPath, disabled, 0o600); err != nil {
		return err
	}
	defer func() {
		if err := os.WriteFile(configurationPath, configuration, 0o600); err != nil && result == nil {
			result = err
		}
	}()

	port, err := availablePort()
	if err != nil {
		return err
	}
	command := exec.Command(binary, "serve", root, "--port", strconv.Itoa(port))
	command.Env = replaceEnvironment(os.Environ(), "PATH", executableDirectory)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return err
	}
	done, lifetime := trackServer(command)
	defer func() {
		if lifetime.Err() == nil {
			_ = command.Process.Kill()
		}
		<-lifetime.Done()
		if result != nil {
			result = fmt.Errorf("%w; opt-out stdout=%q stderr=%q", result, stdout.String(), stderr.String())
		}
	}()
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	if err := waitForServer(lifetime, baseURL); err != nil {
		return fmt.Errorf("DOCX opt-out startup: %w", err)
	}
	body, err := request(lifetime, baseURL+"/", http.StatusOK)
	if err != nil {
		return err
	}
	if strings.Contains(body, "download=docx") {
		return fmt.Errorf("DOCX menu remains visible with export_docx = false")
	}
	if _, err := request(lifetime, baseURL+"/README.md?download=docx", http.StatusNotFound); err != nil {
		return fmt.Errorf("DOCX opt-out request: %w", err)
	}
	return stopServer(command, done)
}

func writeFixture(root, name, content string) {
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		panic(err)
	}
}

func checkAttachment(header http.Header, filename, contentType string) error {
	mediaType, parameters, err := mime.ParseMediaType(header.Get("Content-Disposition"))
	if err != nil || mediaType != "attachment" || parameters["filename"] != filename {
		return fmt.Errorf("attachment disposition = %q, parameters = %+v, error = %v", header.Get("Content-Disposition"), parameters, err)
	}
	if header.Get("Content-Type") != contentType || header.Get("Cache-Control") != "no-store" || header.Get("X-Content-Type-Options") != "nosniff" {
		return fmt.Errorf("attachment headers are incomplete: %+v", header)
	}
	return nil
}

func validDOCX(body []byte) bool {
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return false
	}
	required := map[string]bool{"[Content_Types].xml": false, "word/document.xml": false}
	for _, file := range archive.File {
		if _, ok := required[file.Name]; ok {
			required[file.Name] = true
		}
	}
	for _, present := range required {
		if !present {
			return false
		}
	}
	return true
}

func installFakePandoc() (string, error) {
	directory, err := os.MkdirTemp("", "mavis-fake-pandoc-")
	if err != nil {
		return "", err
	}
	executable, err := os.Executable()
	if err != nil {
		os.RemoveAll(directory)
		return "", err
	}
	body, err := os.ReadFile(executable)
	if err != nil {
		os.RemoveAll(directory)
		return "", err
	}
	name := "pandoc"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(directory, name), body, 0o700); err != nil {
		os.RemoveAll(directory)
		return "", err
	}
	return directory, nil
}

func fakePandoc(arguments []string) error {
	if len(arguments) != 8 || arguments[0] != "--sandbox" || arguments[1] != "-f" || arguments[2] != "html" ||
		arguments[3] != "-t" || arguments[4] != "docx" || arguments[5] != "-o" {
		return fmt.Errorf("unexpected pandoc arguments: %q", arguments)
	}
	if _, err := os.ReadFile(arguments[7]); err != nil {
		return err
	}
	output, err := os.Create(arguments[6])
	if err != nil {
		return err
	}
	archive := zip.NewWriter(output)
	for name, body := range map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": "<document/>"} {
		file, err := archive.Create(name)
		if err != nil {
			output.Close()
			return err
		}
		if _, err := file.Write([]byte(body)); err != nil {
			output.Close()
			return err
		}
	}
	if err := archive.Close(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func replaceEnvironment(environment []string, key, value string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(name, key) {
			result = append(result, entry)
		}
	}
	return append(result, key+"="+value)
}

func availablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("allocate smoke-test port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("release smoke-test port: %w", err)
	}
	return port, nil
}

func waitForServer(parent context.Context, baseURL string) error {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	for {
		if _, err := request(ctx, baseURL+"/", http.StatusOK); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func request(parent context.Context, url string, expectedStatus int) (string, error) {
	body, _, err := requestResponse(parent, url, expectedStatus)
	return body, err
}

func requestResponse(parent context.Context, url string, expectedStatus int) (string, http.Header, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil, err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	closeErr := response.Body.Close()
	if readErr != nil {
		return "", nil, readErr
	}
	if closeErr != nil {
		return "", nil, closeErr
	}
	if response.StatusCode != expectedStatus {
		return string(body), response.Header, fmt.Errorf("GET %s returned %d, want %d", url, response.StatusCode, expectedStatus)
	}
	return string(body), response.Header, nil
}

func trackServer(command *exec.Cmd) (<-chan error, context.Context) {
	done := make(chan error, 1)
	lifetime, exited := context.WithCancel(context.Background())
	go func() {
		done <- command.Wait()
		exited()
	}()
	return done, lifetime
}

func stopServer(command *exec.Cmd, done <-chan error) error {
	if runtime.GOOS == "windows" {
		if err := command.Process.Kill(); err != nil {
			return fmt.Errorf("stop Windows smoke server: %w", err)
		}
		<-done
		return nil
	}
	if err := command.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("interrupt smoke server: %w", err)
	}
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	case <-time.After(10 * time.Second):
		_ = command.Process.Kill()
		return fmt.Errorf("graceful shutdown timed out")
	}
}

func checkWindowsSocket(binary string) error {
	root, err := os.MkdirTemp("", "mavis-windows-socket-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, "serve", root, "--socket", filepath.Join(root, "mavis.sock")).CombinedOutput()
	if err == nil {
		return fmt.Errorf("Windows socket mode unexpectedly started")
	}
	if !strings.Contains(string(output), "unix-socket listeners are not supported on Windows") {
		return fmt.Errorf("Windows socket error is not actionable: %q", output)
	}
	return nil
}
