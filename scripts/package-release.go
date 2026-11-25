//go:build ignore

// package-release builds and packages one standalone Mavis release artifact.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	version := flag.String("version", "", "release version, for example v1.0.0")
	goos := flag.String("goos", runtime.GOOS, "target operating system")
	goarch := flag.String("goarch", runtime.GOARCH, "target CPU architecture")
	output := flag.String("output", "dist", "output directory")
	flag.Parse()
	if err := packageRelease(*version, *goos, *goarch, *output); err != nil {
		fmt.Fprintln(os.Stderr, "package release:", err)
		os.Exit(1)
	}
}

func packageRelease(version, goos, goarch, output string) error {
	if version == "" || version == "." || version == ".." || strings.ContainsAny(version, `/\\`) {
		return fmt.Errorf("invalid version %q", version)
	}
	base := fmt.Sprintf("mavis-%s-%s-%s", version, goos, goarch)
	stage := filepath.Join(output, base)
	if err := os.RemoveAll(stage); err != nil {
		return fmt.Errorf("clean staging directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(stage, "docs"), 0o755); err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}

	binaryName := "mavis"
	archiveName := base + ".tar.gz"
	if goos == "windows" {
		binaryName += ".exe"
		archiveName = base + ".zip"
	}
	binaryPath := filepath.Join(stage, binaryName)
	command := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w", "-o", binaryPath, "./cmd/mavis")
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("build %s/%s binary: %w", goos, goarch, err)
	}

	if err := copyFile("README.md", filepath.Join(stage, "README.md"), 0o644); err != nil {
		return err
	}
	// Preserve README links, inter-guide links, and the example used in the quick start.
	for _, tree := range []string{"docs/content", "testdata/docs"} {
		if err := os.CopyFS(filepath.Join(stage, tree), os.DirFS(tree)); err != nil {
			return fmt.Errorf("stage %s: %w", tree, err)
		}
	}

	archivePath := filepath.Join(output, archiveName)
	if goos == "windows" {
		err := writeZip(archivePath, stage, base, binaryName)
		if err != nil {
			return err
		}
	} else if err := writeTarGz(archivePath, stage, base); err != nil {
		return err
	}
	if err := writeChecksum(archivePath); err != nil {
		return err
	}
	fmt.Println("Packaged:", archivePath)
	fmt.Println("Binary:", binaryPath)
	return nil
}

func copyFile(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open packaged document %s: %w", source, err)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create packaged document %s: %w", destination, err)
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return fmt.Errorf("copy packaged document %s: %w", source, err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close packaged document %s: %w", destination, err)
	}
	return nil
}

func writeTarGz(archivePath, stage, base string) error {
	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	compressed := gzip.NewWriter(archive)
	writer := tar.NewWriter(compressed)
	err = filepath.Walk(stage, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filepath.Join(base, relative))
		if info.IsDir() {
			header.Name += "/"
		}
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		archive.Close()
		return fmt.Errorf("write tar archive: %w", err)
	}
	if err := writer.Close(); err != nil {
		archive.Close()
		return fmt.Errorf("finish tar archive: %w", err)
	}
	if err := compressed.Close(); err != nil {
		archive.Close()
		return fmt.Errorf("finish gzip archive: %w", err)
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}
	return nil
}

func writeZip(archivePath, stage, base, binaryName string) error {
	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	writer := zip.NewWriter(archive)
	err = filepath.Walk(stage, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filepath.Join(base, relative))
		header.Method = zip.Deflate
		if filepath.Base(path) == binaryName {
			header.SetMode(0o755)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(entry, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		archive.Close()
		return fmt.Errorf("write zip archive: %w", err)
	}
	if err := writer.Close(); err != nil {
		archive.Close()
		return fmt.Errorf("finish zip archive: %w", err)
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}
	return nil
}

func writeChecksum(archivePath string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive for checksum: %w", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, archive); err != nil {
		archive.Close()
		return fmt.Errorf("hash archive: %w", err)
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close archive after checksum: %w", err)
	}
	checksum := fmt.Sprintf("%x  %s\n", hash.Sum(nil), filepath.Base(archivePath))
	if err := os.WriteFile(archivePath+".sha256", []byte(checksum), 0o644); err != nil {
		return fmt.Errorf("write checksum: %w", err)
	}
	return nil
}
