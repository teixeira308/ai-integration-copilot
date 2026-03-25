package generator

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type generatedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type generatedPackage struct {
	Files []generatedFile `json:"files"`
}

type GeneratedFileMeta struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

func persistGeneratedOutput(jobID string, raw string) (string, []GeneratedFileMeta, error) {
	pkg, err := parseGeneratedPackage(raw)
	if err != nil {
		return "", nil, err
	}
	if len(pkg.Files) == 0 {
		return "", nil, fmt.Errorf("generated package contained no files")
	}

	outputDir := filepath.Join(os.TempDir(), "ai-integration-output", jobID)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create output directory: %w", err)
	}

	metas := make([]GeneratedFileMeta, 0, len(pkg.Files))
	for _, file := range pkg.Files {
		if strings.TrimSpace(file.Path) == "" {
			return "", nil, fmt.Errorf("generated file path is empty")
		}

		cleanPath := filepath.Clean(file.Path)
		if cleanPath == "." || strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			return "", nil, fmt.Errorf("invalid generated file path: %s", file.Path)
		}

		content := file.Content
		if strings.HasSuffix(cleanPath, ".go") {
			formatted, err := format.Source([]byte(content))
			if err == nil {
				content = string(formatted)
			}
		}

		fullPath := filepath.Join(outputDir, cleanPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return "", nil, fmt.Errorf("create file directory: %w", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return "", nil, fmt.Errorf("write generated file %s: %w", cleanPath, err)
		}

		metas = append(metas, GeneratedFileMeta{
			Path:  cleanPath,
			Bytes: len(content),
		})
	}

	return outputDir, metas, nil
}

func packageGeneratedOutput(jobID, outputDir string, files []GeneratedFileMeta) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		return "", fmt.Errorf("output directory is empty")
	}

	zipDir := filepath.Join(os.TempDir(), "ai-integration-output", "archives")
	if err := os.MkdirAll(zipDir, 0o755); err != nil {
		return "", fmt.Errorf("create archive directory: %w", err)
	}

	zipPath := filepath.Join(zipDir, jobID+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("create zip file: %w", err)
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	for _, file := range files {
		fullPath := filepath.Join(outputDir, file.Path)
		src, err := os.Open(fullPath)
		if err != nil {
			return "", fmt.Errorf("open generated file %s: %w", file.Path, err)
		}

		entry, err := zw.Create(file.Path)
		if err != nil {
			src.Close()
			return "", fmt.Errorf("create zip entry %s: %w", file.Path, err)
		}

		if _, err := io.Copy(entry, src); err != nil {
			src.Close()
			return "", fmt.Errorf("copy zip entry %s: %w", file.Path, err)
		}

		src.Close()
	}

	return zipPath, nil
}

func parseGeneratedPackage(raw string) (*generatedPackage, error) {
	payload := strings.TrimSpace(raw)
	if payload == "" {
		return nil, fmt.Errorf("empty model output")
	}

	if strings.HasPrefix(payload, "```") {
		payload = strings.TrimPrefix(payload, "```json")
		payload = strings.TrimPrefix(payload, "```JSON")
		payload = strings.TrimPrefix(payload, "```")
		payload = strings.TrimSuffix(payload, "```")
		payload = strings.TrimSpace(payload)
	}

	payload = strings.ReplaceAll(payload, "\r\n", "\n")

	start := strings.Index(payload, "{")
	end := strings.LastIndex(payload, "}")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("model output does not contain a JSON object")
	}
	payload = payload[start : end+1]

	var pkg generatedPackage
	if err := json.Unmarshal([]byte(payload), &pkg); err != nil {
		return nil, fmt.Errorf("decode generated package: %w", err)
	}

	return &pkg, nil
}
