# AI Integration Copilot — Implementation Plan

This document defines the step-by-step implementation plan.

The implementation is designed to work well with AI coding agents operating under limited quota.

The project must be generated incrementally.

---

# Phase 1 — Integration Generator (MVP)

Goal:

Generate a Go API client from an OpenAPI specification.

Estimated time: 4–5 days

---

# Step 1 — Project Initialization

Create repository structure.

ai-integration-copilot/

backend/
frontend/
generator/
docs/

Backend stack:

Go  
Gin

Frontend stack:

React  
Vite  
Tailwind

Deliverable:

Initial repository with empty modules.

---

# Step 2 — Backend Skeleton (completed)

Create backend service.

backend/

cmd/server/main.go  
internal/api/router.go  
internal/config/config.go

The server must expose:

- `POST /generate` (JSON or multipart upload) that stores the spec and returns job metadata/status.
- `GET /generate/{jobId}` for polling prompt/result previews in addition to the status endpoint.

---

# Step 3 — OpenAPI Parser (completed)

Implement parser module.

backend/internal/parser/

parser.go  
types.go

# Step 4 — LLM Prompt Builder (completed)

Recommended library:

github.com/getkin/kin-openapi

Deliverable:

Parser returning structured API representation.

---

# Step 4 — LLM Prompt Builder

# Step 5 — Code Generator (in progress)

---

# Step 5 — Code Generator

Module:

backend/internal/generator/

job.go  
runner.go

Responsibilities:

- Track generation jobs (status, prompt, result, timestamps).
- Invoke local Ollama (`ollama run`) via `runner.go` to execute prompts.
- Return prompt/result previews in `POST /generate` and `GET /generate/{jobId}` responses.

Next deliverables:

1. Implement actual Go client/model/auth code emission once prompt/result data is ready.
2. Persist generated files (e.g., under `integration/`) and expose download links.

---

# Phase 2 — Architecture Diagram Generator (next)

Goal:

Generate Mermaid diagrams describing integration architecture.

Estimated time: 3–4 days

---

# Step 6 — Integration Flow Builder (pending)

Module:

backend/internal/architecture/

flow_builder.go

Responsibilities:

Analyze endpoints and determine:

- API flows
- webhook events

---

# Step 7 — Diagram Generator (pending)

Generate Mermaid diagrams.

Files:

architecture/

sequence.mmd  
events.mmd  

Example output:

sequenceDiagram

Client->>API: POST /payments  
API-->>Client: payment_id  
API->>Webhook: payment.success  

---

# Step 8 — Frontend UI (pending)

Frontend responsibilities:

- upload OpenAPI spec
- trigger generation
- display results
- render diagrams

Libraries:

React  
Tailwind  
Mermaid.js

---

# Step 9 — Integration Package Builder

Combine generated files into a downloadable package.

Structure:

integration/

client/
webhook/
retry/
architecture/
README.md

Package format:

ZIP

---

# AI Generation Strategy (Important)

Because the project may be generated using an AI coding agent with limited quota:

The implementation must be generated in the following order:

1. Repository structure
2. Backend skeleton
3. Parser
4. Generator
5. Frontend
6. Diagram generator

After each step the agent should stop and wait for the next instruction.

Never generate more than one module at a time.

This prevents the generation from stopping due to quota limits.
