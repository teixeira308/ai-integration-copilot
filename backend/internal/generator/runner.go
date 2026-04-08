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
	defaultModel   = "gemini-3.1-flash-lite-preview"
	defaultBaseURL = "https://generativelanguage.googleapis.com"
)

type geminiGenerateRequest struct {
	Contents []geminiContent `json:"contents"`
	Config   geminiConfig    `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiConfig struct {
	ResponseMIMEType string `json:"responseMimeType,omitempty"`
}

type geminiGenerateResponse struct {
	Candidates     []geminiCandidate     `json:"candidates"`
	PromptFeedback *geminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *geminiUsageMetadata  `json:"usageMetadata,omitempty"`
	Error          *geminiError          `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content       *geminiContent   `json:"content,omitempty"`
	FinishReason  string           `json:"finishReason,omitempty"`
	SafetyRatings []map[string]any `json:"safetyRatings,omitempty"`
}

type geminiPromptFeedback struct {
	BlockReason string `json:"blockReason,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}

type geminiError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}

// Runner executes generation jobs via the Gemini generateContent API.
type Runner struct {
	model      string
	baseURL    string
	apiKey     string
	timeout    time.Duration
	httpClient *http.Client
}

// NewRunner returns a runner configured for the given Gemini model, base URL, API key, and timeout.
func NewRunner(model, baseURL, apiKey, timeoutValue string) *Runner {
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
		apiKey:  strings.TrimSpace(apiKey),
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Execute calls the Gemini API and returns the generated output.
func (r *Runner) Execute(job *Job) (*ExecutionResult, error) {
	if r.apiKey == "" {
		return nil, fmt.Errorf("gemini api key is not configured")
	}

	payload := geminiGenerateRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: job.Prompt},
				},
			},
		},
		Config: geminiConfig{
			ResponseMIMEType: "application/json",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal gemini request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	requestStartedAt := time.Now()
	log.Printf("job=%s phase=gemini start model=%s base_url=%s timeout=%s prompt_chars=%d", job.ID, r.model, r.baseURL, r.timeout, len(job.Prompt))

	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent", r.baseURL, r.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", r.apiKey)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		log.Printf("job=%s phase=gemini failed duration_ms=%d error=%q", job.ID, time.Since(requestStartedAt).Milliseconds(), err.Error())
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read gemini response: %w", err)
	}

	if resp.StatusCode >= 300 {
		log.Printf("job=%s phase=gemini failed duration_ms=%d status=%d", job.ID, time.Since(requestStartedAt).Milliseconds(), resp.StatusCode)
		return nil, fmt.Errorf("gemini returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode gemini response: %w", err)
	}

	if parsed.Error != nil {
		return nil, fmt.Errorf("gemini error: %s", strings.TrimSpace(parsed.Error.Message))
	}

	if parsed.PromptFeedback != nil && strings.TrimSpace(parsed.PromptFeedback.BlockReason) != "" {
		return nil, fmt.Errorf("gemini blocked prompt: %s", parsed.PromptFeedback.BlockReason)
	}

	output := extractGeminiText(parsed)
	if strings.TrimSpace(output) == "" {
		return nil, fmt.Errorf("gemini returned an empty response")
	}

	metrics := &ExecutionMetrics{
		Model:             r.model,
		BaseURL:           r.baseURL,
		PromptChars:       len(job.Prompt),
		ResultChars:       len(output),
		RequestDurationMS: time.Since(requestStartedAt).Milliseconds(),
	}
	if parsed.UsageMetadata != nil {
		metrics.PromptEvalCount = parsed.UsageMetadata.PromptTokenCount
		metrics.EvalCount = parsed.UsageMetadata.CandidatesTokenCount
	}
	if job.Spec != nil {
		metrics.EndpointCount = len(job.Spec.Endpoints)
		metrics.SecurityCount = len(job.Spec.Security)
	}

	log.Printf(
		"job=%s phase=gemini succeeded duration_ms=%d eval_count=%d prompt_eval_count=%d result_chars=%d",
		job.ID,
		metrics.RequestDurationMS,
		metrics.EvalCount,
		metrics.PromptEvalCount,
		metrics.ResultChars,
	)

	return &ExecutionResult{
		Output:  output,
		Metrics: metrics,
	}, nil
}

func extractGeminiText(resp geminiGenerateResponse) string {
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return ""
	}

	var builder strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		builder.WriteString(part.Text)
	}

	return builder.String()
}
