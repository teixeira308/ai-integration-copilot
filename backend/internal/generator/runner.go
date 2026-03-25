package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	defaultModel   = "qwen2.5-coder"
	defaultBaseURL = "http://host.docker.internal:11434"
)

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response           string `json:"response"`
	Error              string `json:"error"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// Runner executes generation jobs via the Ollama HTTP API.
type Runner struct {
	model      string
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
}

// NewRunner returns a runner configured for the given Ollama model name, base URL, and timeout.
func NewRunner(model, baseURL, timeoutValue string) *Runner {
	if strings.TrimSpace(model) == "" {
		model = defaultModel
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	timeout := 2 * time.Minute
	if strings.TrimSpace(timeoutValue) != "" {
		if parsed, err := time.ParseDuration(timeoutValue); err == nil {
			timeout = parsed
		}
	}

	return &Runner{
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Execute calls the Ollama API and returns the generated output.
func (r *Runner) Execute(job *Job) (*ExecutionResult, error) {
	payload := ollamaGenerateRequest{
		Model:  r.model,
		Prompt: job.Prompt,
		Stream: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	requestStartedAt := time.Now()
	log.Printf("job=%s phase=ollama start model=%s base_url=%s timeout=%s prompt_chars=%d", job.ID, r.model, r.baseURL, r.timeout, len(job.Prompt))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		log.Printf("job=%s phase=ollama failed duration_ms=%d error=%q", job.ID, time.Since(requestStartedAt).Milliseconds(), err.Error())
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ollama response: %w", err)
	}

	if resp.StatusCode >= 300 {
		log.Printf("job=%s phase=ollama failed duration_ms=%d status=%d", job.ID, time.Since(requestStartedAt).Milliseconds(), resp.StatusCode)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed ollamaGenerateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	if strings.TrimSpace(parsed.Error) != "" {
		return nil, fmt.Errorf("ollama error: %s", parsed.Error)
	}

	metrics := &ExecutionMetrics{
		Model:             r.model,
		BaseURL:           r.baseURL,
		PromptChars:       len(job.Prompt),
		ResultChars:       len(parsed.Response),
		TotalDurationMS:   parsed.TotalDuration / int64(time.Millisecond),
		LoadDurationMS:    parsed.LoadDuration / int64(time.Millisecond),
		PromptEvalCount:   parsed.PromptEvalCount,
		PromptEvalMS:      parsed.PromptEvalDuration / int64(time.Millisecond),
		EvalCount:         parsed.EvalCount,
		EvalDurationMS:    parsed.EvalDuration / int64(time.Millisecond),
		RequestDurationMS: time.Since(requestStartedAt).Milliseconds(),
	}
	if job.Spec != nil {
		metrics.EndpointCount = len(job.Spec.Endpoints)
		metrics.SecurityCount = len(job.Spec.Security)
	}

	log.Printf(
		"job=%s phase=ollama succeeded duration_ms=%d total_duration_ms=%d eval_count=%d prompt_eval_count=%d result_chars=%d",
		job.ID,
		metrics.RequestDurationMS,
		metrics.TotalDurationMS,
		metrics.EvalCount,
		metrics.PromptEvalCount,
		metrics.ResultChars,
	)

	return &ExecutionResult{
		Output:  parsed.Response,
		Metrics: metrics,
	}, nil
}
