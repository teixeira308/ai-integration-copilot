package parser

import (
	"fmt"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

// ParseFromFile loads and parses an OpenAPI specification from a local file and
// returns a structured representation capturing endpoints, security, and metadata.
func ParseFromFile(path string) (*SpecDocument, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	return buildSpecDocument(doc), nil
}

func buildSpecDocument(doc *openapi3.T) *SpecDocument {
	result := &SpecDocument{
		Title:       doc.Info.Title,
		Description: doc.Info.Description,
		Version:     doc.Info.Version,
		BaseURL:     resolveBaseURL(doc),
		Security:    collectSecuritySchemes(doc, &doc.Security),
	}

	var endpoints []Endpoint

	for path, item := range doc.Paths {
		for _, entry := range collectOperations(item) {
			endpoint := Endpoint{
				Path:         path,
				Method:       entry.method,
				OperationID:  entry.operation.OperationID,
				Summary:      entry.operation.Summary,
				Description:  entry.operation.Description,
				RequestBody:  buildMediaTypeSchemas(entry.operation.RequestBody),
				Responses:    buildResponses(entry.operation.Responses),
				Security:     collectSecuritySchemes(doc, entry.operation.Security),
				AuthRequired: entry.operation.Security != nil && len(*entry.operation.Security) > 0,
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path == endpoints[j].Path {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	result.Endpoints = endpoints

	return result
}

type operationEntry struct {
	method    string
	operation *openapi3.Operation
}

func collectOperations(item *openapi3.PathItem) []operationEntry {
	if item == nil {
		return nil
	}

	entries := []operationEntry{}

	add := func(method string, op *openapi3.Operation) {
		if op != nil {
			entries = append(entries, operationEntry{method: method, operation: op})
		}
	}

	add("HEAD", item.Head)
	add("DELETE", item.Delete)
	add("GET", item.Get)
	add("PATCH", item.Patch)
	add("POST", item.Post)
	add("PUT", item.Put)
	add("OPTIONS", item.Options)

	return entries
}

func buildMediaTypeSchemas(bodyRef *openapi3.RequestBodyRef) []MediaTypeSchema {
	if bodyRef == nil || bodyRef.Value == nil {
		return nil
	}

	return mapMediaTypes("", bodyRef.Value.Content)
}

func buildResponses(responses openapi3.Responses) []MediaTypeSchema {
	if responses == nil {
		return nil
	}

	var result []MediaTypeSchema

	for status, respRef := range responses {
		result = append(result, mapMediaTypes(status, respRef.Value.Content)...)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Status == result[j].Status {
			return result[i].MediaType < result[j].MediaType
		}
		if result[i].Status == "" {
			return false
		}
		if result[j].Status == "" {
			return true
		}
		return result[i].Status < result[j].Status
	})

	return result
}

func mapMediaTypes(status string, content openapi3.Content) []MediaTypeSchema {
	if len(content) == 0 {
		return nil
	}

	var mediaSchemas []MediaTypeSchema

	for mediaType, media := range content {
		if media == nil || media.Schema == nil || media.Schema.Value == nil {
			continue
		}

		schema := media.Schema.Value

		mediaSchemas = append(mediaSchemas, MediaTypeSchema{
			Status:       status,
			MediaType:    mediaType,
			SchemaRef:    media.Schema.Ref,
			SchemaType:   schema.Type,
			SchemaFormat: schema.Format,
		})
	}

	sort.Slice(mediaSchemas, func(i, j int) bool {
		return mediaSchemas[i].MediaType < mediaSchemas[j].MediaType
	})

	return mediaSchemas
}

func resolveBaseURL(doc *openapi3.T) string {
	if len(doc.Servers) == 0 {
		return ""
	}

	return doc.Servers[0].URL
}

func collectSecuritySchemes(doc *openapi3.T, reqs *openapi3.SecurityRequirements) []SecurityScheme {
	if doc == nil {
		return nil
	}

	schemes := []SecurityScheme{}
	seen := map[string]struct{}{}

	add := func(name string, ref *openapi3.SecuritySchemeRef) {
		if ref == nil || ref.Value == nil {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}

		schemes = append(schemes, SecurityScheme{
			Name:         name,
			Type:         ref.Value.Type,
			In:           ref.Value.In,
			Scheme:       ref.Value.Scheme,
			BearerFormat: ref.Value.BearerFormat,
		})
	}

	if reqs != nil && len(*reqs) > 0 {
		for _, req := range *reqs {
			for name := range req {
				add(name, doc.Components.SecuritySchemes[name])
			}
		}
		return schemes
	}

	for name, ref := range doc.Components.SecuritySchemes {
		add(name, ref)
	}

	return schemes
}
