package api

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/guilhermeteixeira/ai-integration-copilot/backend/internal/ai"
	"github.com/guilhermeteixeira/ai-integration-copilot/backend/internal/config"
	"github.com/guilhermeteixeira/ai-integration-copilot/backend/internal/generator"
	"github.com/guilhermeteixeira/ai-integration-copilot/backend/internal/parser"
)

var (
	jobManager = generator.NewManager()
	jobRunner  = generator.NewRunner(os.Getenv("OLLAMA_MODEL"), os.Getenv("OLLAMA_BASE_URL"), os.Getenv("OLLAMA_TIMEOUT"))
)

// GenerateRequest describes the bare minimum information required to start generation.
type GenerateRequest struct {
	SpecSource string `json:"specSource"`
	SpecURL    string `json:"specUrl"`
	SpecPath   string `json:"specPath"`
}

// GenerateResponse is a lightweight acknowledgement for the POST /generate endpoint.
type GenerateResponse struct {
	JobID         string                      `json:"jobId"`
	Status        string                      `json:"status"`
	Title         string                      `json:"title,omitempty"`
	Version       string                      `json:"version,omitempty"`
	BaseURL       string                      `json:"baseUrl,omitempty"`
	EndpointCount int                         `json:"endpointCount"`
	Security      []parser.SecurityScheme     `json:"security,omitempty"`
	PromptPreview string                      `json:"promptPreview,omitempty"`
	ResultPreview string                      `json:"resultPreview,omitempty"`
	Metrics       *generator.ExecutionMetrics `json:"metrics,omitempty"`
}

// RegisterRoutes wires the API endpoints to the router.
func RegisterRoutes(router *gin.Engine, cfg *config.Config) {
	router.POST("/generate", handleGenerate)
	router.GET("/generate/:id", handleJobStatus)
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"server_port": cfg.Server.Port,
		})
	})
}

func handleGenerate(ctx *gin.Context) {
	requestStartedAt := time.Now()
	var specPath string
	contentType := ctx.ContentType()

	if strings.HasPrefix(contentType, "multipart/form-data") {
		file, err := ctx.FormFile("specFile")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "specFile must be supplied"})
			return
		}

		specPath, err = saveUploadedSpec(file)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to store upload: %v", err)})
			return
		}
	} else {
		var payload GenerateRequest
		if err := ctx.ShouldBindJSON(&payload); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if payload.SpecSource != "" && payload.SpecSource != "file" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "unsupported specSource; only \"file\" is implemented"})
			return
		}

		specPath = strings.TrimSpace(payload.SpecPath)
	}

	if specPath == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "specPath is required"})
		return
	}

	parseStartedAt := time.Now()
	doc, err := parser.ParseFromFile(specPath)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("phase=parse spec_path=%s endpoint_count=%d security_count=%d duration_ms=%d", specPath, len(doc.Endpoints), len(doc.Security), time.Since(parseStartedAt).Milliseconds())

	promptStartedAt := time.Now()
	prompt := ai.BuildPrompt(doc)
	log.Printf("phase=prompt prompt_chars=%d duration_ms=%d", len(prompt), time.Since(promptStartedAt).Milliseconds())

	job := jobManager.Create(doc, prompt)
	log.Printf("job=%s phase=job status=created endpoint_count=%d security_count=%d total_setup_ms=%d", job.ID, len(doc.Endpoints), len(doc.Security), time.Since(requestStartedAt).Milliseconds())
	jobManager.Schedule(job, jobRunner)

	resp := GenerateResponse{
		JobID:         job.ID,
		Status:        job.Status,
		Title:         doc.Title,
		Version:       doc.Version,
		BaseURL:       doc.BaseURL,
		EndpointCount: len(doc.Endpoints),
		Security:      doc.Security,
		PromptPreview: previewPrompt(prompt),
		ResultPreview: previewResult(job.Result),
		Metrics:       job.Metrics,
	}

	ctx.JSON(http.StatusAccepted, resp)
}

func handleJobStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	job, ok := jobManager.Get(id)
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	ctx.JSON(http.StatusOK, JobStatusResponse{
		JobID:         job.ID,
		Status:        job.Status,
		CreatedAt:     job.CreatedAt,
		StartedAt:     job.StartedAt,
		CompletedAt:   job.CompletedAt,
		PromptPreview: previewPrompt(job.Prompt),
		ResultPreview: previewResult(job.Result),
		Error:         job.Error,
		Metrics:       job.Metrics,
	})
}

type JobStatusResponse struct {
	JobID         string                      `json:"jobId"`
	Status        string                      `json:"status"`
	CreatedAt     time.Time                   `json:"createdAt"`
	StartedAt     time.Time                   `json:"startedAt,omitempty"`
	CompletedAt   time.Time                   `json:"completedAt,omitempty"`
	PromptPreview string                      `json:"promptPreview,omitempty"`
	ResultPreview string                      `json:"resultPreview,omitempty"`
	Error         string                      `json:"error,omitempty"`
	Metrics       *generator.ExecutionMetrics `json:"metrics,omitempty"`
}

func saveUploadedSpec(file *multipart.FileHeader) (string, error) {
	startedAt := time.Now()
	tempDir := filepath.Join(os.TempDir(), "ai-integration-specs")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", err
	}

	dest := filepath.Join(tempDir, fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(file.Filename)))

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	log.Printf("phase=upload saved_path=%s filename=%s size_bytes=%d duration_ms=%d", dest, file.Filename, file.Size, time.Since(startedAt).Milliseconds())
	return dest, nil
}

func previewPrompt(full string) string {
	if len(full) <= 160 {
		return full
	}
	return fmt.Sprintf("%s...", strings.TrimSpace(full[:157]))
}

func previewResult(full string) string {
	if len(full) <= 160 {
		return full
	}
	return fmt.Sprintf("%s...", strings.TrimSpace(full[:157]))
}
