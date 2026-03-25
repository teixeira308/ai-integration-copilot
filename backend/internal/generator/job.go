package generator

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/guilhermeteixeira/ai-integration-copilot/backend/internal/parser"
)

// Job represents a generation request tracked by the backend.
type Job struct {
	ID          string               `json:"id"`
	CreatedAt   time.Time            `json:"createdAt"`
	StartedAt   time.Time            `json:"startedAt,omitempty"`
	CompletedAt time.Time            `json:"completedAt,omitempty"`
	Status      string               `json:"status"`
	Prompt      string               `json:"prompt"`
	Result      string               `json:"result,omitempty"`
	Error       string               `json:"error,omitempty"`
	Metrics     *ExecutionMetrics    `json:"metrics,omitempty"`
	OutputDir   string               `json:"outputDir,omitempty"`
	ArchivePath string               `json:"archivePath,omitempty"`
	Files       []GeneratedFileMeta  `json:"files,omitempty"`
	Spec        *parser.SpecDocument `json:"-"`
}

type ExecutionMetrics struct {
	Model             string `json:"model,omitempty"`
	BaseURL           string `json:"baseUrl,omitempty"`
	PromptChars       int    `json:"promptChars,omitempty"`
	ResultChars       int    `json:"resultChars,omitempty"`
	TotalDurationMS   int64  `json:"totalDurationMs,omitempty"`
	LoadDurationMS    int64  `json:"loadDurationMs,omitempty"`
	PromptEvalCount   int    `json:"promptEvalCount,omitempty"`
	PromptEvalMS      int64  `json:"promptEvalMs,omitempty"`
	EvalCount         int    `json:"evalCount,omitempty"`
	EvalDurationMS    int64  `json:"evalDurationMs,omitempty"`
	RequestDurationMS int64  `json:"requestDurationMs,omitempty"`
	EndpointCount     int    `json:"endpointCount,omitempty"`
	SecurityCount     int    `json:"securityCount,omitempty"`
}

type ExecutionResult struct {
	Output  string
	Metrics *ExecutionMetrics
}

// Executor executes a job and returns the generated output or an error.
type Executor interface {
	Execute(job *Job) (*ExecutionResult, error)
}

// Manager holds in-memory jobs.
type Manager struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewManager returns a ready-to-use job manager.
func NewManager() *Manager {
	return &Manager{
		jobs: make(map[string]*Job),
	}
}

// Create registers a new job with the provided prompt and spec.
func (m *Manager) Create(doc *parser.SpecDocument, prompt string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("job-%d", time.Now().UnixNano())
	job := &Job{
		ID:        id,
		CreatedAt: time.Now().UTC(),
		Status:    "pending",
		Prompt:    prompt,
		Spec:      doc,
	}
	m.jobs[id] = job

	return job
}

// Get returns a job by ID.
func (m *Manager) Get(id string) (*Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[id]
	return job, ok
}

// Schedule starts executor processing for the job in the background.
func (m *Manager) Schedule(job *Job, exec Executor) {
	go func() {
		endpointCount := 0
		if job.Spec != nil {
			endpointCount = len(job.Spec.Endpoints)
		}
		log.Printf("job=%s phase=job status=running prompt_chars=%d endpoint_count=%d", job.ID, len(job.Prompt), endpointCount)
		m.update(job.ID, func(j *Job) {
			j.Status = "running"
			j.StartedAt = time.Now().UTC()
		})

		result, err := exec.Execute(job)

		m.update(job.ID, func(j *Job) {
			j.CompletedAt = time.Now().UTC()
			if err != nil {
				j.Status = "failed"
				j.Error = err.Error()
				log.Printf("job=%s phase=job status=failed total_ms=%d error=%q", j.ID, j.CompletedAt.Sub(j.CreatedAt).Milliseconds(), j.Error)
			} else {
				outputDir, files, persistErr := persistGeneratedOutput(j.ID, result.Output)
				if persistErr != nil {
					j.Status = "failed"
					j.Error = persistErr.Error()
					j.Result = result.Output
					j.Metrics = result.Metrics
					log.Printf("job=%s phase=artifacts status=failed error=%q", j.ID, j.Error)
					return
				}
				j.Status = "succeeded"
				j.Result = result.Output
				j.Metrics = result.Metrics
				j.OutputDir = outputDir
				j.Files = files
				archivePath, archiveErr := packageGeneratedOutput(j.ID, j.OutputDir, j.Files)
				if archiveErr != nil {
					j.Status = "failed"
					j.Error = archiveErr.Error()
					log.Printf("job=%s phase=archive status=failed error=%q", j.ID, j.Error)
					return
				}
				j.ArchivePath = archivePath
				log.Printf("job=%s phase=artifacts status=succeeded output_dir=%s file_count=%d", j.ID, j.OutputDir, len(j.Files))
				log.Printf("job=%s phase=archive status=succeeded archive_path=%s", j.ID, j.ArchivePath)
				log.Printf("job=%s phase=job status=succeeded total_ms=%d result_chars=%d", j.ID, j.CompletedAt.Sub(j.CreatedAt).Milliseconds(), len(j.Result))
			}
		})
	}()
}

func (m *Manager) update(id string, fn func(*Job)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job, ok := m.jobs[id]; ok {
		fn(job)
	}
}
