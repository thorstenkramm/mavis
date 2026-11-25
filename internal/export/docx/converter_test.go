package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/render"
)

func TestConvertInvokesExactExecutableWithSelfContainedInput(t *testing.T) {
	requireShell(t)
	fixture := writeDOCXFixture(t)
	captureDir := t.TempDir()
	argsCapture := filepath.Join(captureDir, "args")
	inputCapture := filepath.Join(captureDir, "input.html")
	executable := writeFakePandoc(t, fmt.Sprintf(`{
  printf '%%s\n' "$#"
  for argument do printf '%%s\n' "$argument"; done
} > %s
/bin/cp "$8" %s
printf 'PATH=<%%s>\nHOME=<%%s>\nSECRET=<%%s>\n' "${PATH-unset}" "${HOME-unset}" "${MAVIS_DOCX_SECRET-unset}" > %s
printf 'PWD=<%%s>\n' "$PWD" > %s
/bin/cp %s "$7"
`, shellQuote(argsCapture), shellQuote(inputCapture), shellQuote(filepath.Join(captureDir, "environment")), shellQuote(filepath.Join(captureDir, "pwd")), shellQuote(fixture)))

	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "docs/pixel.png"), []byte("private-png-bytes"))
	converter := New(executable, newTestResolver(t, root, testConfiguration()), render.NewRenderer(), nil)
	t.Setenv("MAVIS_DOCX_SECRET", "must-not-leak")
	t.Setenv("PATH", filepath.Join(captureDir, "wrong-path"))
	var destination bytes.Buffer
	documentPath := "docs/guide.md;--extract-media=must-not-be-an-argument"
	if err := converter.Convert(context.Background(), []byte("# Heading\n\n![](pixel.png)\n"), documentPath, &destination); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(destination.Bytes(), readFixture(t, fixture)) {
		t.Fatal("converted bytes differ from fake pandoc output")
	}

	arguments := strings.Split(strings.TrimSuffix(string(readFixture(t, argsCapture)), "\n"), "\n")
	if len(arguments) != 9 || arguments[0] != "8" {
		t.Fatalf("captured arguments = %#v", arguments)
	}
	wantFlags := []string{"--sandbox", "-f", "html", "-t", "docx", "-o"}
	for index, want := range wantFlags {
		if arguments[index+1] != want {
			t.Errorf("argument %d = %q, want %q", index+1, arguments[index+1], want)
		}
	}
	outputPath, inputPath := arguments[7], arguments[8]
	if filepath.Base(outputPath) != "document.docx" || filepath.Base(inputPath) != "document.html" || filepath.Dir(outputPath) != filepath.Dir(inputPath) {
		t.Fatalf("output/input paths are not private fixed names: %q %q", outputPath, inputPath)
	}
	joinedArguments := strings.Join(arguments, "\n")
	for _, forbidden := range []string{root, documentPath, "--extract-media=must-not-be-an-argument"} {
		if strings.Contains(joinedArguments, forbidden) {
			t.Errorf("process arguments expose %q: %s", forbidden, joinedArguments)
		}
	}
	article := string(readFixture(t, inputCapture))
	if !strings.Contains(article, "data:image/png;base64,") || strings.Contains(article, root) {
		t.Fatalf("process input is not self-contained: %s", article)
	}
	environment := string(readFixture(t, filepath.Join(captureDir, "environment")))
	for _, want := range []string{"HOME=<unset>", "SECRET=<unset>"} {
		if !strings.Contains(environment, want) {
			t.Errorf("process environment does not contain %q: %s", want, environment)
		}
	}
	// A POSIX shell may supply its own default PATH after startup. The inherited
	// sentinel must nevertheless be absent because Converter supplies an empty env.
	if strings.Contains(environment, "wrong-path") {
		t.Fatalf("process inherited the caller PATH: %s", environment)
	}
	workingDirectory := strings.TrimSpace(string(readFixture(t, filepath.Join(captureDir, "pwd"))))
	workingDirectory = strings.TrimSuffix(strings.TrimPrefix(workingDirectory, "PWD=<"), ">")
	if filepath.Base(workingDirectory) != filepath.Base(filepath.Dir(inputPath)) {
		t.Fatalf("process working directory = %q, want private directory %q", workingDirectory, filepath.Dir(inputPath))
	}
	assertRemovedTree(t, outputPath)
}

func TestConvertNeverLooksUpBareExecutableInPATH(t *testing.T) {
	requireShell(t)
	directory := t.TempDir()
	capture := filepath.Join(directory, "called")
	writeExecutable(t, filepath.Join(directory, "pandoc"), "#!/bin/sh\nprintf called > "+shellQuote(capture)+"\n")
	t.Setenv("PATH", directory)
	converter := New("pandoc", nil, staticRenderer(`<p>Body</p>`), nil)
	var destination bytes.Buffer
	if err := converter.Convert(context.Background(), nil, "guide.md", &destination); err == nil {
		t.Fatal("Convert() error = nil for unresolved executable")
	}
	if _, err := os.Stat(capture); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("PATH executable was invoked; stat error = %v", err)
	}
	if destination.Len() != 0 {
		t.Fatal("destination received partial output")
	}
}

func TestConvertFailureTimeoutAndCleanup(t *testing.T) {
	requireShell(t)
	tests := []struct {
		name       string
		body       string
		timeout    time.Duration
		wantError  error
		wantInLog  string
		malformed  bool
		nonzeroLog bool
	}{
		{name: "nonzero exit", body: "printf 'stdout diagnostic'\nprintf 'stderr diagnostic' >&2\nexit 23\n", nonzeroLog: true},
		{name: "malformed output", body: "printf 'not a ZIP container' > \"$7\"\n", malformed: true},
		{name: "timeout", body: "while :; do :; done\n", timeout: 500 * time.Millisecond, wantError: context.DeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capture := filepath.Join(t.TempDir(), "output-path")
			executable := writeFakePandoc(t, "printf '%s\\n' \"$7\" > "+shellQuote(capture)+"\n"+test.body)
			var logs bytes.Buffer
			converter := New(executable, nil, staticRenderer(`<p>Body</p>`), log.New(&logs, "", 0))
			ctx := context.Background()
			cancel := func() {}
			if test.timeout != 0 {
				ctx, cancel = context.WithTimeout(ctx, test.timeout)
			}
			defer cancel()
			var destination bytes.Buffer
			err := converter.Convert(ctx, nil, "guide.md", &destination)
			if err == nil {
				t.Fatal("Convert() error = nil")
			}
			if test.wantError != nil && !errors.Is(err, test.wantError) {
				t.Fatalf("Convert() error = %v, want errors.Is(%v)", err, test.wantError)
			}
			if destination.Len() != 0 {
				t.Fatal("destination received partial or malformed output")
			}
			if test.nonzeroLog && (!strings.Contains(logs.String(), "stdout diagnostic") || !strings.Contains(logs.String(), "stderr diagnostic")) {
				t.Fatalf("process diagnostics not logged: %s", logs.String())
			}
			outputPath := strings.TrimSpace(string(readFixture(t, capture)))
			assertRemovedTree(t, outputPath)
		})
	}
}

func TestConvertRejectsSymlinkOutput(t *testing.T) {
	requireShell(t)
	fixture := writeDOCXFixture(t)
	capture := filepath.Join(t.TempDir(), "output-path")
	executable := writeFakePandoc(t, "printf '%s\\n' \"$7\" > "+shellQuote(capture)+"\n/bin/ln -s "+shellQuote(fixture)+" \"$7\"\n")
	converter := New(executable, nil, staticRenderer(`<p>Body</p>`), nil)
	var destination bytes.Buffer
	if err := converter.Convert(context.Background(), nil, "guide.md", &destination); err == nil {
		t.Fatal("Convert() error = nil for symlink output")
	}
	if destination.Len() != 0 {
		t.Fatal("destination received symlink target bytes")
	}
	assertRemovedTree(t, strings.TrimSpace(string(readFixture(t, capture))))
}

func TestReadOutputEnforcesRegularFileAndSizeCap(t *testing.T) {
	directory := t.TempDir()
	oversized := filepath.Join(directory, "oversized.docx")
	file, err := os.Create(oversized)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxOutputSize + 1); err != nil {
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readOutput(oversized); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("readOutput(oversized) error = %v", err)
	}
	if _, err := readOutput(directory); err == nil {
		t.Fatal("readOutput(directory) error = nil")
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(directory, "link.docx")
		if err := os.Symlink(oversized, link); err != nil {
			t.Fatal(err)
		}
		if _, err := readOutput(link); err == nil {
			t.Fatal("readOutput(symlink) error = nil")
		}
	}
}

func TestValidateDOCXRequiresContainerEntries(t *testing.T) {
	valid := makeDOCX(t, map[string]string{
		"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`,
		"word/document.xml":   `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body/></w:document>`,
	})
	if err := validateDOCX(valid); err != nil {
		t.Fatalf("validateDOCX(valid) error = %v", err)
	}
	for _, test := range []struct {
		name string
		data []byte
	}{
		{name: "not ZIP", data: []byte("not ZIP")},
		{name: "missing content types", data: makeDOCX(t, map[string]string{"word/document.xml": "document"})},
		{name: "missing document", data: makeDOCX(t, map[string]string{"[Content_Types].xml": "types"})},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateDOCX(test.data); err == nil {
				t.Fatal("validateDOCX() error = nil")
			}
		})
	}
}

func TestBoundedLogDiscardsExcessWithoutShortWrites(t *testing.T) {
	output := &boundedLog{}
	input := bytes.Repeat([]byte("x"), maxLogSize+100)
	written, err := output.Write(input)
	if err != nil || written != len(input) {
		t.Fatalf("Write() = %d, %v; want %d, nil", written, err, len(input))
	}
	if output.Len() != maxLogSize || !output.truncated {
		t.Fatalf("bounded log length = %d, truncated = %v", output.Len(), output.truncated)
	}
	written, err = output.Write([]byte("more"))
	if err != nil || written != 4 || output.Len() != maxLogSize {
		t.Fatalf("second Write() = %d, %v, length %d", written, err, output.Len())
	}
}

func TestConvertDoesNotWriteValidatedOutputWhenDestinationFails(t *testing.T) {
	requireShell(t)
	fixture := writeDOCXFixture(t)
	executable := writeFakePandoc(t, "/bin/cp "+shellQuote(fixture)+" \"$7\"\n")
	converter := New(executable, nil, staticRenderer(`<p>Body</p>`), nil)
	writer := &failingWriter{}
	if err := converter.Convert(context.Background(), nil, "guide.md", writer); err == nil || !strings.Contains(err.Error(), "write destination") {
		t.Fatalf("Convert() error = %v, want destination failure", err)
	}
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func staticRenderer(article string) Renderer {
	return rendererFunc(func([]byte, string, render.AccessFunc) (render.Document, error) {
		return render.Document{Title: "Title", HTML: article}, nil
	})
}

func testConfiguration() (configuration config.Config) {
	return config.Default()
}

func requireShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake pandoc scripts require a POSIX shell")
	}
}

func writeFakePandoc(t *testing.T, body string) string {
	t.Helper()
	return writeExecutable(t, filepath.Join(t.TempDir(), "fake-pandoc"), "#!/bin/sh\nset -eu\n"+body)
}

func writeExecutable(t *testing.T, name, body string) string {
	t.Helper()
	if err := os.WriteFile(name, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return name
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func assertRemovedTree(t *testing.T, outputPath string) {
	t.Helper()
	if outputPath == "" {
		t.Fatal("fake pandoc did not capture its output path")
	}
	if _, err := os.Stat(filepath.Dir(outputPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("private conversion directory still exists; stat error = %v", err)
	}
}

func makeDOCX(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for name, contents := range entries {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func writeDOCXFixture(t *testing.T) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "valid.docx")
	data := makeDOCX(t, map[string]string{
		"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`,
		"word/document.xml":   `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body/></w:document>`,
	})
	writeFixture(t, name, data)
	return name
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
