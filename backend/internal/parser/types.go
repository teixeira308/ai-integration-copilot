package parser

type SpecDocument struct {
	Title       string
	Description string
	Version     string
	BaseURL     string
	Endpoints   []Endpoint
	Security    []SecurityScheme
}

type Endpoint struct {
	Path         string
	Method       string
	OperationID  string
	Summary      string
	Description  string
	RequestBody  []MediaTypeSchema
	Responses    []MediaTypeSchema
	Security     []SecurityScheme
	AuthRequired bool
}

type MediaTypeSchema struct {
	Status       string
	MediaType    string
	SchemaRef    string
	SchemaType   string
	SchemaFormat string
}

type SecurityScheme struct {
	Name         string
	Type         string
	In           string
	Scheme       string
	BearerFormat string
}
