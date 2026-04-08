package ai

import (
	"fmt"
	"strings"

	"github.com/guilhermeteixeira/ai-integration-copilot/backend/internal/parser"
)

// BuildPrompt assembles a human-readable description of the parsed spec that can be
// consumed by downstream generators or LLMs.
func BuildPrompt(doc *parser.SpecDocument) string {
	if doc == nil {
		return "No spec document provided."
	}

	var b strings.Builder

	b.WriteString("You are generating a production-leaning Go SDK from an OpenAPI summary.\n")
	b.WriteString("Return exactly one JSON object and nothing else.\n")
	b.WriteString("Do not include markdown fences, comments outside JSON, explanations, XML tags, or prose before or after the JSON object.\n")
	b.WriteString("Do not output diff markers, patch format, leading '+' characters, placeholders like TODO, or truncated code.\n")
	b.WriteString("The response must be valid JSON that can be decoded with encoding/json.\n")
	b.WriteString("Use this schema exactly: {\"files\":[{\"path\":\"relative/path.go\",\"content\":\"full file contents\"}]}.\n")
	b.WriteString("Every file entry must contain only path and content.\n")
	b.WriteString("All paths must be relative and use forward slashes.\n")
	b.WriteString("Generate complete file contents, not snippets.\n")
	b.WriteString("The generated project must compile after go mod tidy and go test ./....\n")
	b.WriteString("Target Go 1.21+ idiomatic code.\n\n")

	b.WriteString("Required output contract:\n")
	b.WriteString("- Must include go.mod.\n")
	b.WriteString("- Must include client/client.go for shared HTTP infrastructure.\n")
	b.WriteString("- Must include client/models.go or multiple schema files under client/.\n")
	b.WriteString("- Must include client/auth.go for explicit authentication handling.\n")
	b.WriteString("- Must include client/errors.go for structured HTTP and decode errors.\n")
	b.WriteString("- Must include one or more operation files under client/ implementing the endpoints.\n")
	b.WriteString("- Must include cmd/example/main.go with a compilable example.\n")
	b.WriteString("- Must include README.md with setup, auth, BaseURL override, usage, and known simplifications.\n")
	b.WriteString("- Must include at least one *_test.go file with httptest-based coverage.\n\n")

	b.WriteString("SDK quality requirements:\n")
	b.WriteString("- Every public endpoint method must accept context.Context as the first parameter.\n")
	b.WriteString("- Generate operations for all endpoints listed below, not just one example endpoint.\n")
	b.WriteString("- Centralize request building, response handling, and error handling in shared infrastructure.\n")
	b.WriteString("- Support BaseURL, custom HTTPClient, timeout, default headers, and user agent through explicit client configuration.\n")
	b.WriteString("- Model authentication explicitly from securitySchemes. Support apiKey header, bearer token, and basic auth when applicable.\n")
	b.WriteString("- Preserve JSON names with struct tags and avoid inventing fields that are not grounded in the spec summary.\n")
	b.WriteString("- Use deterministic Go typing. If a field is optional and absence matters, prefer pointers.\n")
	b.WriteString("- Handle response decoding by status code. Do not assume every 2xx has the same schema.\n")
	b.WriteString("- Treat 204 or empty bodies without JSON decoding.\n")
	b.WriteString("- Return structured errors containing status code, method, path, and a limited raw response body preview.\n")
	b.WriteString("- Validate required input before sending requests.\n")
	b.WriteString("- Omit optional query parameters when unset.\n")
	b.WriteString("- If the summary lacks some details, choose the safest minimal assumption and document the limitation in README.md.\n\n")

	b.WriteString("Testing requirements:\n")
	b.WriteString("- Include at least one httptest.Server integration-style test for a successful endpoint call.\n")
	b.WriteString("- Include at least one test covering HTTP error handling.\n")
	b.WriteString("- Include at least one test covering authentication header application when auth is present.\n")
	b.WriteString("- Include at least one serialization round-trip or decode test for a primary model.\n\n")

	b.WriteString("Project constraints:\n")
	b.WriteString("- Use package client for SDK files under client/.\n")
	b.WriteString("- Use package main for cmd/example/main.go.\n")
	b.WriteString("- Keep the code reasonably small, but do not sacrifice correctness for brevity.\n")
	b.WriteString("- Prefer standard library imports unless a dependency is clearly justified.\n")
	b.WriteString("- README.md is documentation only.\n\n")

	b.WriteString(fmt.Sprintf("Generate a Go SDK for %s (%s)\n", doc.Title, doc.Version))
	if strings.TrimSpace(doc.Description) != "" {
		b.WriteString("API description:\n")
		b.WriteString(strings.TrimSpace(doc.Description))
		b.WriteString("\n")
	}
	if doc.BaseURL != "" {
		b.WriteString(fmt.Sprintf("Base URL: %s\n", doc.BaseURL))
	}

	if len(doc.Security) > 0 {
		b.WriteString("\nGlobal security schemes:\n")
		for _, scheme := range doc.Security {
			b.WriteString(fmt.Sprintf("- name=%s | type=%s | scheme=%s | in=%s | bearerFormat=%s\n",
				emptyAsUnknown(scheme.Name),
				emptyAsUnknown(scheme.Type),
				emptyAsUnknown(scheme.Scheme),
				emptyAsUnknown(scheme.In),
				emptyAsUnknown(scheme.BearerFormat),
			))
		}
	}

	if len(doc.Endpoints) > 0 {
		b.WriteString("\nEndpoints to implement:\n")
		for _, ep := range doc.Endpoints {
			b.WriteString(fmt.Sprintf("- %s %s\n", ep.Method, ep.Path))
			b.WriteString(fmt.Sprintf("  operationId: %s\n", emptyAsUnknown(ep.OperationID)))
			b.WriteString(fmt.Sprintf("  summary: %s\n", emptyAsUnknown(ep.Summary)))
			b.WriteString(fmt.Sprintf("  description: %s\n", emptyAsUnknown(ep.Description)))
			b.WriteString(fmt.Sprintf("  authRequired: %t\n", ep.AuthRequired))
			if len(ep.Security) > 0 {
				b.WriteString("  endpointSecurity:\n")
				for _, scheme := range ep.Security {
					b.WriteString(fmt.Sprintf("  - name=%s | type=%s | scheme=%s | in=%s\n",
						emptyAsUnknown(scheme.Name),
						emptyAsUnknown(scheme.Type),
						emptyAsUnknown(scheme.Scheme),
						emptyAsUnknown(scheme.In),
					))
				}
			}
			if len(ep.RequestBody) > 0 {
				b.WriteString("  requestBody:\n")
				for _, body := range ep.RequestBody {
					b.WriteString(fmt.Sprintf("  - mediaType=%s | schemaRef=%s | schemaType=%s | schemaFormat=%s\n",
						emptyAsUnknown(body.MediaType),
						emptyAsUnknown(body.SchemaRef),
						emptyAsUnknown(body.SchemaType),
						emptyAsUnknown(body.SchemaFormat),
					))
				}
			}
			if len(ep.Responses) > 0 {
				b.WriteString("  responses:\n")
				for _, resp := range ep.Responses {
					b.WriteString(fmt.Sprintf("  - status=%s | mediaType=%s | schemaRef=%s | schemaType=%s | schemaFormat=%s\n",
						emptyAsUnknown(resp.Status),
						emptyAsUnknown(resp.MediaType),
						emptyAsUnknown(resp.SchemaRef),
						emptyAsUnknown(resp.SchemaType),
						emptyAsUnknown(resp.SchemaFormat),
					))
				}
			}
		}
	}

	b.WriteString("\nFinal output rules:\n")
	b.WriteString("- Escape newlines and quotes correctly so the JSON remains valid.\n")
	b.WriteString("- Do not wrap the JSON in ```json fences.\n")
	b.WriteString("- Do not omit any required file category.\n")
	b.WriteString("- Do not invent extra infrastructure like Docker, CI, database, or web server code.\n")
	b.WriteString("- Only generate the SDK project files.\n")
	b.WriteString("- Favor correctness and debuggability over superficial brevity.\n")

	return strings.TrimSpace(b.String())
}

func emptyAsUnknown(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}
