# Parser

`backend/internal/parser` provides a thin wrapper around `kin-openapi` that extracts the
key metadata needed for generation: service title/version, base URL, endpoint definitions,
and authentication requirements.

## Usage

```go
doc, err := parser.ParseFromFile("specs/payments.yaml")
if err != nil {
    return fmt.Errorf("parse spec: %w", err)
}

fmt.Printf("Discover %d endpoints for %s\n", len(doc.Endpoints), doc.Title)
```

The returned `SpecDocument` already resolves request/response media types and security
schemes so the generator can focus on prompt construction.
