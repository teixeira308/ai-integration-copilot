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
			line := fmt.Sprintf("- %s %s: %s\n", ep.Method, ep.Path, ep.Summary)
			if len(line) > 100 {
				line = line[:100] + "..."
			}
			b.WriteString(line)
		}
	}

	return strings.TrimSpace(b.String())
}
