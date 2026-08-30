package kite

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

/**
 * RouteDoc holds the OpenAPI metadata for a specific route.
 */
type RouteDoc struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
	Hidden      bool
	RequestBody *BodyDoc
	Responses   map[int]*ResponseDoc
	Parameters  []ParamDoc
}

/**
 * BodyDoc defines metadata for an HTTP request body.
 */
type BodyDoc struct {
	Model       any
	Description string
	Required    bool
	ContentType string
}

/**
 * ResponseDoc defines metadata for an HTTP response.
 */
type ResponseDoc struct {
	Model       any
	Description string
	ContentType string
}

/**
 * ParamDoc defines metadata for a path, query, or header parameter.
 */
type ParamDoc struct {
	In          string // "path", "query", "header", "cookie"
	Name        string
	Model       any
	Description string
	Required    bool
	Example     any
}

/**
 * OpenAPISpec represents the root OpenAPI 3.1.0 document.
 */
type OpenAPISpec struct {
	OpenAPI    string                            `json:"openapi"`
	Info       OpenAPIInfo                       `json:"info"`
	Servers    []OpenAPIServer                   `json:"servers,omitempty"`
	Paths      map[string]map[string]OpenAPIOp   `json:"paths"`
	Components OpenAPIComponents                 `json:"components,omitempty"`
}

type OpenAPIInfo struct {
	Title          string `json:"title"`
	Version        string `json:"version"`
	Description    string `json:"description,omitempty"`
	TermsOfService string `json:"termsOfService,omitempty"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type OpenAPIOp struct {
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	OperationID string                     `json:"operationId,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Parameters  []OpenAPIParam             `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
	Deprecated  bool                       `json:"deprecated,omitempty"`
}

type OpenAPIParam struct {
	Name        string         `json:"name"`
	In          string         `json:"in"`
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Schema      map[string]any `json:"schema,omitempty"`
	Example     any            `json:"example,omitempty"`
}

type OpenAPIRequestBody struct {
	Description string                      `json:"description,omitempty"`
	Required    bool                        `json:"required,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content"`
}

type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

type OpenAPIMediaType struct {
	Schema  map[string]any `json:"schema,omitempty"`
	Example any            `json:"example,omitempty"`
}

type OpenAPIComponents struct {
	Schemas map[string]map[string]any `json:"schemas,omitempty"`
}

/**
 * OpenAPIGenerator builds an OpenAPI 3.1 specification from Kite App and routes.
 */
type OpenAPIGenerator struct {
	config  Config
	schemas map[string]map[string]any
}

/**
 * NewOpenAPIGenerator creates an instance of the OpenAPI generator.
 */
func NewOpenAPIGenerator(config Config) *OpenAPIGenerator {
	return &OpenAPIGenerator{
		config:  config,
		schemas: make(map[string]map[string]any),
	}
}

/**
 * Generate builds the full OpenAPI specification from a list of routes.
 */
func (g *OpenAPIGenerator) Generate(routes []*node) *OpenAPISpec {
	spec := &OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:       g.config.Title,
			Version:     g.config.Version,
			Description: g.config.Description,
		},
		Paths: make(map[string]map[string]OpenAPIOp),
	}

	if g.config.Title == "" {
		spec.Info.Title = "Kite API"
	}
	if g.config.Version == "" {
		spec.Info.Version = "1.0.0"
	}

	for _, s := range g.config.Servers {
		spec.Servers = append(spec.Servers, OpenAPIServer{
			URL:         s.URL,
			Description: s.Description,
		})
	}

	for _, n := range routes {
		if n.doc != nil && n.doc.Hidden {
			continue
		}
		// Ignore internal OPTIONS preflight routes
		if n.method == http.MethodOptions {
			continue
		}

		// Convert Kite route format (/users/:id/*file) to OpenAPI format (/users/{id}/{file})
		openAPIPath, pathParams := convertPathToOpenAPI(n.fullPath)

		if spec.Paths[openAPIPath] == nil {
			spec.Paths[openAPIPath] = make(map[string]OpenAPIOp)
		}

		method := strings.ToLower(n.method)
		if method == "" {
			method = "get"
		}

		op := OpenAPIOp{
			Responses: make(map[string]OpenAPIResponse),
		}

		doc := n.doc
		if doc != nil {
			op.Summary = doc.Summary
			op.Description = doc.Description
			op.Tags = doc.Tags
			op.Deprecated = doc.Deprecated
		}

		// Path parameters from route path
		for _, paramName := range pathParams {
			op.Parameters = append(op.Parameters, OpenAPIParam{
				Name:        paramName,
				In:          "path",
				Required:    true,
				Description: fmt.Sprintf("Parameter '%s'", paramName),
				Schema:      map[string]any{"type": "string"},
			})
		}

		// Explicit parameters (query, headers, etc.)
		if doc != nil {
			for _, p := range doc.Parameters {
				schema := g.generateSchema(p.Model)
				param := OpenAPIParam{
					Name:        p.Name,
					In:          p.In,
					Description: p.Description,
					Required:    p.Required || p.In == "path",
					Schema:      schema,
					Example:     p.Example,
				}
				op.Parameters = append(op.Parameters, param)
			}

			// Request Body
			if doc.RequestBody != nil && doc.RequestBody.Model != nil {
				contentType := doc.RequestBody.ContentType
				if contentType == "" {
					contentType = "application/json"
				}
				schema := g.generateSchema(doc.RequestBody.Model)
				op.RequestBody = &OpenAPIRequestBody{
					Description: doc.RequestBody.Description,
					Required:    doc.RequestBody.Required,
					Content: map[string]OpenAPIMediaType{
						contentType: {
							Schema: schema,
						},
					},
				}
			}

			// Responses
			for code, respDoc := range doc.Responses {
				codeStr := strconv.Itoa(code)
				description := respDoc.Description
				if description == "" {
					description = http.StatusText(code)
				}

				resp := OpenAPIResponse{
					Description: description,
				}

				if respDoc.Model != nil {
					contentType := respDoc.ContentType
					if contentType == "" {
						contentType = "application/json"
					}
					schema := g.generateSchema(respDoc.Model)
					resp.Content = map[string]OpenAPIMediaType{
						contentType: {
							Schema: schema,
						},
					}
				}

				op.Responses[codeStr] = resp
			}
		}

		// Fallback default response if none registered
		if len(op.Responses) == 0 {
			op.Responses["200"] = OpenAPIResponse{
				Description: "Successful operation",
			}
		}

		spec.Paths[openAPIPath][method] = op
	}

	if len(g.schemas) > 0 {
		spec.Components = OpenAPIComponents{
			Schemas: g.schemas,
		}
	}

	return spec
}

/**
 * generateSchema reflects upon a Go value or type and creates a JSON Schema structure.
 */
func (g *OpenAPIGenerator) generateSchema(v any) map[string]any {
	if v == nil {
		return map[string]any{"type": "object"}
	}

	t := reflect.TypeOf(v)
	return g.typeToSchema(t)
}

func (g *OpenAPIGenerator) typeToSchema(t reflect.Type) map[string]any {
	// Unwrap pointers
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Handle time.Time
	if t == reflect.TypeOf(time.Time{}) {
		return map[string]any{
			"type":   "string",
			"format": "date-time",
		}
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return map[string]any{"type": "integer", "format": "int32"}
	case reflect.Int64:
		return map[string]any{"type": "integer", "format": "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32:
		return map[string]any{"type": "number", "format": "float"}
	case reflect.Float64:
		return map[string]any{"type": "number", "format": "double"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "format": "binary"}
		}
		return map[string]any{
			"type":  "array",
			"items": g.typeToSchema(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": g.typeToSchema(t.Elem()),
		}
	case reflect.Struct:
		typeName := t.Name()
		if typeName != "" {
			// If it's a named struct, register in components.schemas
			if _, exists := g.schemas[typeName]; !exists {
				// Prevent infinite recursion on self-referencing types
				g.schemas[typeName] = map[string]any{"type": "object"}
				g.schemas[typeName] = g.structToSchema(t)
			}
			return map[string]any{
				"$ref": "#/components/schemas/" + typeName,
			}
		}
		return g.structToSchema(t)
	case reflect.Interface:
		return map[string]any{"type": "object"}
	default:
		return map[string]any{"type": "string"}
	}
}

func (g *OpenAPIGenerator) structToSchema(t reflect.Type) map[string]any {
	properties := make(map[string]map[string]any)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		isOmitEmpty := false

		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
			for _, p := range parts[1:] {
				if p == "omitempty" {
					isOmitEmpty = true
				}
			}
		}

		fieldSchema := g.typeToSchema(field.Type)

		// Read field tags: doc, description, example, format, validate, binding
		doc := field.Tag.Get("doc")
		if doc == "" {
			doc = field.Tag.Get("description")
		}
		if doc != "" {
			fieldSchema["description"] = doc
		}

		example := field.Tag.Get("example")
		if example != "" {
			fieldSchema["example"] = parseExampleValue(example, field.Type)
		}

		format := field.Tag.Get("format")
		if format != "" {
			fieldSchema["format"] = format
		}

		defaultVal := field.Tag.Get("default")
		if defaultVal != "" {
			fieldSchema["default"] = parseExampleValue(defaultVal, field.Type)
		}

		validate := field.Tag.Get("validate")
		binding := field.Tag.Get("binding")
		isRequired := strings.Contains(validate, "required") || strings.Contains(binding, "required")

		if isRequired || (!isOmitEmpty && field.Type.Kind() != reflect.Ptr) {
			if isRequired {
				required = append(required, fieldName)
			}
		}

		properties[fieldName] = fieldSchema
	}

	sort.Strings(required)

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func parseExampleValue(val string, t reflect.Type) any {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	case reflect.Bool:
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return val
}

/**
 * convertPathToOpenAPI converts "/users/:id/*file" to "/users/{id}/{file}"
 * and returns the list of extracted parameter names.
 */
func convertPathToOpenAPI(path string) (string, []string) {
	if path == "" {
		return "/", nil
	}

	parts := strings.Split(path, "/")
	var params []string
	var converted []string

	for _, p := range parts {
		if strings.HasPrefix(p, ":") {
			param := p[1:]
			params = append(params, param)
			converted = append(converted, "{"+param+"}")
		} else if p == "*" {
			params = append(params, "wildcard")
			converted = append(converted, "{wildcard}")
		} else {
			converted = append(converted, p)
		}
	}

	return strings.Join(converted, "/"), params
}

/**
 * ToJSON serializes the OpenAPISpec into indented JSON bytes.
 */
func (s *OpenAPISpec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}
