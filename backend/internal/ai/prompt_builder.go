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

	b.WriteString("Return only valid JSON.\n")
	b.WriteString("Do not include markdown fences, explanations, or prose outside JSON.\n")
	b.WriteString("Generate exactly these files: client/client.go, client/models.go, client/auth.go, README.md.\n")
	b.WriteString("The JSON schema must be: {\"files\":[{\"path\":\"client/client.go\",\"content\":\"...\"}]}.\n")
	b.WriteString("Go files must compile together as package client, use only standard library imports when possible, and keep the implementation minimal but valid.\n")
	b.WriteString("README.md must summarize the generated client and how auth should be configured.\n\n")
	b.WriteString(fmt.Sprintf("Generate a Go integration client for %s (%s)\n", doc.Title, doc.Version))
	if doc.BaseURL != "" {
		b.WriteString(fmt.Sprintf("Base URL: %s\n", doc.BaseURL))
	}

	if len(doc.Security) > 0 {
		b.WriteString("Authentication\ntype | scheme | in\n")
		for _, scheme := range doc.Security {
			b.WriteString(fmt.Sprintf("- %s | %s | %s\n", scheme.Type, scheme.Scheme, scheme.In))
		}
	}

	if len(doc.Endpoints) > 0 {
		b.WriteString("\nEndpoints:\n")
		for _, ep := range doc.Endpoints {
			line := fmt.Sprintf("- %s %s | operationId=%s | authRequired=%t | summary=%s\n", ep.Method, ep.Path, ep.OperationID, ep.AuthRequired, ep.Summary)
			b.WriteString(line)
		}
	}

	return strings.TrimSpace(b.String())
}
