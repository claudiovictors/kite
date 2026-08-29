package kite

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type TestCreateUserRequest struct {
	Name     string    `json:"name" doc:"Nome do usuário" example:"Carlos Silva" validate:"required"`
	Email    string    `json:"email" doc:"E-mail do usuário" format:"email" example:"carlos@email.com" validate:"required"`
	Age      int       `json:"age,omitempty" doc:"Idade do usuário" example:"28"`
	IsActive bool      `json:"is_active" default:"true"`
	JoinedAt time.Time `json:"joined_at"`
}

type TestUserResponse struct {
	ID    string `json:"id" example:"usr_123"`
	Name  string `json:"name" example:"Carlos Silva"`
	Email string `json:"email" example:"carlos@email.com"`
}

func TestOpenAPIGeneration(t *testing.T) {
	app := New(Config{
		Title:       "Test Kite API",
		Version:     "2.0.0",
		Description: "API de Teste com OpenAPI e Scalar",
	})

	app.Get("/users", func(req Request, res Response) error {
		return res.Json([]TestUserResponse{})
	}).
		Summary("Listar usuários").
		Description("Retorna todos os usuários cadastrados").
		Tags("Users").
		Response(200, []TestUserResponse{})

	app.Post("/users", func(req Request, res Response) error {
		return res.Status(201).Json(TestUserResponse{})
	}).
		Summary("Criar usuário").
		Tags("Users").
		Body(TestCreateUserRequest{}).
		Response(201, TestUserResponse{})

	app.Get("/users/:id", func(req Request, res Response) error {
		return res.Json(TestUserResponse{ID: req.Param("id")})
	}).
		Summary("Buscar usuário por ID").
		Tags("Users").
		Response(200, TestUserResponse{}).
		Response(404, ErrorResponse{}, "Usuário não encontrado")

	// Test GET /openapi.json
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Esperado status 200 em /openapi.json, recebido %d", rec.Code)
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("Falha ao deserializar JSON da spec: %v", err)
	}

	if spec.Info.Title != "Test Kite API" {
		t.Errorf("Esperado Title 'Test Kite API', recebido %q", spec.Info.Title)
	}
	if spec.Info.Version != "2.0.0" {
		t.Errorf("Esperado Version '2.0.0', recebido %q", spec.Info.Version)
	}

	// Verify paths
	if _, ok := spec.Paths["/users"]; !ok {
		t.Errorf("Rota '/users' não encontrada no OpenAPI spec")
	}
	if _, ok := spec.Paths["/users/{id}"]; !ok {
		t.Errorf("Rota '/users/{id}' não encontrada no OpenAPI spec (conversão de :id falhou)")
	}

	// Verify components schemas
	if spec.Components.Schemas == nil {
		t.Fatalf("Components.Schemas está nil")
	}
	if _, ok := spec.Components.Schemas["TestCreateUserRequest"]; !ok {
		t.Errorf("Schema 'TestCreateUserRequest' não encontrado nos components")
	}
	if _, ok := spec.Components.Schemas["TestUserResponse"]; !ok {
		t.Errorf("Schema 'TestUserResponse' não encontrado nos components")
	}

	userReqSchema := spec.Components.Schemas["TestCreateUserRequest"]
	props, ok := userReqSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("Propriedades de TestCreateUserRequest inválidas")
	}
	if _, ok := props["name"]; !ok {
		t.Errorf("Campo 'name' não encontrado em TestCreateUserRequest")
	}
	if _, ok := props["email"]; !ok {
		t.Errorf("Campo 'email' não encontrado em TestCreateUserRequest")
	}

	// Test GET /docs (Scalar)
	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	app.ServeHTTP(docsRec, docsReq)

	if docsRec.Code != http.StatusOK {
		t.Fatalf("Esperado status 200 em /docs, recebido %d", docsRec.Code)
	}

	bodyStr := docsRec.Body.String()
	if !strings.Contains(bodyStr, "@scalar/api-reference") {
		t.Errorf("HTML do /docs não contém o script do Scalar")
	}
	if !strings.Contains(bodyStr, "/openapi.json") {
		t.Errorf("HTML do /docs não aponta para /openapi.json")
	}
}

func TestDisableDocs(t *testing.T) {
	app := New(Config{
		DisableDocs: true,
	})

	app.Get("/hello", func(req Request, res Response) error {
		return res.Send("hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Esperado 404 para /docs quando DisableDocs=true, recebido %d", rec.Code)
	}
}
