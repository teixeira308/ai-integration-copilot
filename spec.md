# AI Integration Copilot — Specification

## Progress

- Backend service (Gin) with `/generate` POST plus `/health` probe and Dockerized runtime.
- Multipart upload handler that stores specs under `/tmp/ai-integration-specs`.
- Kin-openapi parser producing `SpecDocument` (metadata, endpoints, media types, security).
- Prompt builder summarizing spec data and Ollama runner to execute LLM prompts locally.
- In-memory job tracking with status polling (`GET /generate/{jobId}`) and preview metadata in responses.

## Next Steps

- Implement generator that writes actual Go integration files (client/models/auth) based on prompts.
- Generate architecture diagrams (Mermaid) describing API/event flows.
- Build frontend UI for uploading specs, viewing job progress, and downloading packages.

## Overview

AI Integration Copilot is a developer tool that automatically generates API integrations from documentation.

The system ingests API documentation (OpenAPI/Swagger or URL), analyzes endpoints using an LLM, and produces a ready-to-use integration package including:

- API client
- data models
- webhook handlers
- retry strategy
- idempotency helpers
- architecture diagrams

The goal is to reduce the time required to integrate third-party APIs.

Target users are backend engineers and integration engineers.

---

# Core Concept

Input:

- OpenAPI specification
- Swagger JSON/YAML
- Documentation URL

Output:

integration-package/

client/
models/
webhook/
retry/
tests/
architecture/
README.md

---

# Phase 1 — Integration Generator (MVP)

The system generates a Go client based on an OpenAPI specification.

## Features

### OpenAPI ingestion

The system must accept:

- OpenAPI JSON file
- OpenAPI YAML file
- URL pointing to OpenAPI spec

### Endpoint parsing

Extract:

- base URL
- endpoints
- HTTP methods
- request schemas
- response schemas
- authentication method

### Client generation

The generated client must include:

client/

client.go  
models.go  
auth.go  

Example generated method:

```go
func (c *Client) CreatePayment(req CreatePaymentRequest) (*PaymentResponse, error)
````

### Authentication support

Initial supported auth types:

* API Key
* Bearer Token

The system should detect auth method from OpenAPI security definitions.

---

# Phase 2 — Architecture Diagram Generator

The system generates architecture diagrams for the integration.

These diagrams describe:

* API call flow
* webhook callbacks
* retry logic

Diagrams should be generated using Mermaid syntax.

Example output:

sequenceDiagram

Client->>PaymentAPI: POST /payments
PaymentAPI-->>Client: payment_id
PaymentAPI->>WebhookEndpoint: payment.succeeded

---

# Architecture Diagram Types

### Sequence diagram

Shows request/response flow.

### Event flow diagram

Shows event based communication (webhooks).

### Component diagram

Shows system components:

* Your Service
* Integration Client
* External API
* Webhook Handler

---

# System Components

## Frontend

Responsibilities:

* Accept OpenAPI input
* Trigger generation
* Display results
* Render architecture diagrams

Tech stack:

React
Vite
Tailwind

---

## Backend

Responsibilities:

* Parse OpenAPI
* Generate prompts for LLM
* Generate code files
* Generate diagrams

Tech stack:

Go
Gin
LLM provider API

---

# AI Integration Engine

The AI engine receives:

* OpenAPI structure
* Endpoint list
* Auth configuration

And generates:

* Go client
* Webhook handler
* Retry strategy

Prompt structure must include:

* endpoint list
* request models
* response models
* auth strategy

---

# Generated Integration Structure

Example output:

integration-stripe/

client/

client.go
models.go
auth.go

webhook/

handler.go

retry/

retry.go

architecture/

sequence.mmd
events.mmd

README.md

---

# Codex Execution Constraints

This project may be generated using an AI coding agent with limited quota (such as Codex in a free or promotional tier).

To avoid exceeding quota limits:

The system must follow these rules during code generation:

1. Generate the project **incrementally**
2. Never attempt to generate the entire repository in a single response
3. Generate **one module at a time**
4. Prefer **short files over large files**
5. Stop after completing each module and wait for the next instruction

Suggested generation order:

1. Project structure
2. Backend skeleton
3. OpenAPI parser
4. Code generator
5. Frontend skeleton
6. Diagram generator

This ensures the project can be completed even under strict token or usage limits.

---

# Non-Functional Requirements

Generated code must:

* compile
* follow Go conventions
* avoid unused imports
* be formatted with gofmt

Performance:

Integration generation should complete within 10 seconds for typical APIs.

Security:

The system must not:

* store API keys
* execute generated code automatically

---

# Success Criteria

The MVP is successful if:

1. User uploads an OpenAPI spec
2. System generates a working Go client
3. Architecture diagrams are generated
4. Integration package can be downloaded
