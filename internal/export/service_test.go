package export

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thorstenkramm/mavis/internal/content"
)

type converterFunc func(context.Context, []byte, string, io.Writer) error

func (function converterFunc) Convert(ctx context.Context, source []byte, documentPath string, destination io.Writer) error {
	return function(ctx, source, documentPath, destination)
}

func TestServiceExportsCurrentSourceAndValidatesPDF(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "Handbuch.markdown")
	writeExportFile(t, filename, []byte("before"))
	service := NewService(converterFunc(func(_ context.Context, source []byte, documentPath string, destination io.Writer) error {
		if documentPath != "Handbuch.markdown" || string(source) != "after" {
			t.Errorf("conversion input = %q, %q", documentPath, source)
		}
		_, err := destination.Write([]byte("%PDF-1.7\ncomplete"))
		return err
	}), nil)
	writeExportFile(t, filename, []byte("after"))

	result, err := service.Export(context.Background(), content.Resource{Path: filename, RelativePath: "Handbuch.markdown"}, PDF)
	if err != nil {
		t.Fatal(err)
	}
	if result.Filename != "Handbuch.pdf" || result.ContentType != "application/pdf" || string(result.Body) != "%PDF-1.7\ncomplete" {
		t.Fatalf("result = %+v", result)
	}
}

func TestServiceValidatesDOCXContainer(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "guide.md")
	writeExportFile(t, filename, []byte("# Guide"))
	service := NewService(nil, converterFunc(func(_ context.Context, _ []byte, _ string, destination io.Writer) error {
		_, err := destination.Write(validDOCX(t))
		return err
	}))
	result, err := service.Export(context.Background(), content.Resource{Path: filename, RelativePath: "guide.md"}, DOCX)
	if err != nil {
		t.Fatal(err)
	}
	if result.Filename != "guide.docx" || !bytes.HasPrefix(result.Body, []byte("PK")) {
		t.Fatalf("DOCX result = %+v", result)
	}
}

func TestServiceRejectsUnknownUnavailableAndInvalidOutput(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "guide.md")
	writeExportFile(t, filename, []byte("# Guide"))
	resource := content.Resource{Path: filename, RelativePath: "guide.md"}
	service := NewService(converterFunc(func(_ context.Context, _ []byte, _ string, destination io.Writer) error {
		_, err := destination.Write([]byte("not a PDF"))
		return err
	}), nil)
	if _, err := service.Export(context.Background(), resource, Format("html")); !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("unknown format error = %v", err)
	}
	if _, err := service.Export(context.Background(), resource, DOCX); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unavailable DOCX error = %v", err)
	}
	if _, err := service.Export(context.Background(), resource, PDF); !errors.Is(err, ErrConversion) {
		t.Fatalf("invalid PDF error = %v", err)
	}
}

func TestServiceEnforcesSourceAndOutputLimits(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "guide.md")
	writeExportFile(t, filename, []byte("12345"))
	resource := content.Resource{Path: filename, RelativePath: "guide.md"}
	service := NewService(converterFunc(func(_ context.Context, _ []byte, _ string, destination io.Writer) error {
		_, err := destination.Write([]byte("%PDF-too-large"))
		return err
	}), nil)
	service.maxSource = 4
	service.maxOutput = 8
	if _, err := service.Export(context.Background(), resource, PDF); !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("source limit error = %v", err)
	}
	service.maxSource = 10
	if _, err := service.Export(context.Background(), resource, PDF); !errors.Is(err, ErrOutputTooLarge) {
		t.Fatalf("output limit error = %v", err)
	}
}

func TestServiceRecoversPanics(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "guide.md")
	writeExportFile(t, filename, []byte("# Guide"))
	service := NewService(converterFunc(func(context.Context, []byte, string, io.Writer) error {
		panic("private converter panic")
	}), nil)
	_, err := service.Export(context.Background(), content.Resource{Path: filename, RelativePath: "guide.md"}, PDF)
	if !errors.Is(err, ErrConversion) {
		t.Fatalf("panic error = %v", err)
	}
}

func TestServiceTimeoutInvalidatesLateResultAndRetainsWorker(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "guide.md")
	writeExportFile(t, filename, []byte("# Guide"))
	started := make(chan struct{})
	release := make(chan struct{})
	service := NewService(converterFunc(func(_ context.Context, _ []byte, _ string, destination io.Writer) error {
		close(started)
		<-release
		_, err := destination.Write([]byte("%PDF-late"))
		return err
	}), nil)
	service.timeout = 20 * time.Millisecond
	done := make(chan error, 1)
	go func() {
		_, err := service.Export(context.Background(), content.Resource{Path: filename, RelativePath: "guide.md"}, PDF)
		done <- err
	}()
	<-started
	if err := <-done; !errors.Is(err, ErrTimeout) {
		t.Fatalf("timeout error = %v", err)
	}
	if len(service.workers) != 1 {
		t.Fatalf("worker released before conversion returned: %d", len(service.workers))
	}
	close(release)
	waitForWorkers(t, service, 0)
}

func TestServiceCancellation(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "guide.md")
	writeExportFile(t, filename, []byte("# Guide"))
	service := NewService(converterFunc(func(context.Context, []byte, string, io.Writer) error { return nil }), nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.Export(ctx, content.Resource{Path: filename, RelativePath: "guide.md"}, PDF)
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestServiceMapsConverterDeadline(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "guide.md")
	writeExportFile(t, filename, []byte("# Guide"))
	service := NewService(converterFunc(func(context.Context, []byte, string, io.Writer) error {
		return context.DeadlineExceeded
	}), nil)
	_, err := service.Export(context.Background(), content.Resource{Path: filename, RelativePath: "guide.md"}, PDF)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("converter deadline error = %v", err)
	}
}

func TestServiceBoundsQueuedSourceReads(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "guide.md")
	writeExportFile(t, filename, []byte("# Guide"))
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	service := NewService(converterFunc(func(_ context.Context, _ []byte, _ string, _ io.Writer) error {
		started <- struct{}{}
		<-release
		return context.DeadlineExceeded
	}), nil)
	service.timeout = 100 * time.Millisecond
	resource := content.Resource{Path: filename, RelativePath: "guide.md"}
	completed := make(chan error, 2)
	for index := 0; index < 2; index++ {
		go func() {
			_, err := service.Export(context.Background(), resource, PDF)
			completed <- err
		}()
	}
	<-started
	<-started
	_, err := service.Export(context.Background(), content.Resource{Path: filepath.Join(t.TempDir(), "missing.md"), RelativePath: "missing.md"}, PDF)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("queued missing source error = %v, want deadline while waiting", err)
	}
	close(release)
	waitForWorkers(t, service, 0)
	for index := 0; index < 2; index++ {
		if err := <-completed; !errors.Is(err, ErrTimeout) {
			t.Errorf("blocked conversion error = %v", err)
		}
	}
}

func TestServiceBoundsSharedConcurrency(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "guide.md")
	writeExportFile(t, filename, []byte("# Guide"))
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	var active atomic.Int32
	var maximum atomic.Int32
	converter := converterFunc(func(_ context.Context, _ []byte, _ string, destination io.Writer) error {
		current := active.Add(1)
		for {
			previous := maximum.Load()
			if current <= previous || maximum.CompareAndSwap(previous, current) {
				break
			}
		}
		started <- struct{}{}
		<-release
		active.Add(-1)
		_, err := destination.Write([]byte("%PDF-ok"))
		return err
	})
	service := NewService(converter, converter)
	service.timeout = time.Second
	var wait sync.WaitGroup
	for index := 0; index < 4; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := service.Export(context.Background(), content.Resource{Path: filename, RelativePath: "guide.md"}, PDF); err != nil {
				t.Errorf("concurrent export: %v", err)
			}
		}()
	}
	<-started
	<-started
	select {
	case <-started:
		t.Fatal("more than two conversions started")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	wait.Wait()
	if maximum.Load() != 2 || len(service.workers) != 0 {
		t.Fatalf("maximum = %d, occupied workers = %d", maximum.Load(), len(service.workers))
	}
}

func validDOCX(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for name, body := range map[string]string{
		"[Content_Types].xml": "<Types/>",
		"word/document.xml":   "<document/>",
	} {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func writeExportFile(t *testing.T, filename string, source []byte) {
	t.Helper()
	if err := os.WriteFile(filename, source, 0o600); err != nil {
		t.Fatal(err)
	}
}

func waitForWorkers(t *testing.T, service *Service, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for len(service.workers) != count && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(service.workers) != count {
		t.Fatalf("occupied workers = %d, want %d", len(service.workers), count)
	}
}
