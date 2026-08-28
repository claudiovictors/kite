package kite

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ----------------------------------------------------------------------
// Input() — leitura unificada de campos (JSON, form, query)
// ----------------------------------------------------------------------

func TestInputFromJSON(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		return res.WithJson(map[string]string{"nome": req.Input("nome")})
	})

	body := `{"nome":"Ana"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if !strings.Contains(rec.Body.String(), "Ana") {
		t.Fatalf("esperava 'Ana' no body, veio: %s", rec.Body.String())
	}
}

func TestInputFromQueryString(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.WithJson(map[string]string{"nome": req.Input("nome")})
	})

	r := httptest.NewRequest(http.MethodGet, "/?nome=Carlos", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if !strings.Contains(rec.Body.String(), "Carlos") {
		t.Fatalf("esperava 'Carlos' no body, veio: %s", rec.Body.String())
	}
}

func TestInputFromForm(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		return res.WithJson(map[string]string{"email": req.Input("email")})
	})

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("email=ana%40ex.com"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if !strings.Contains(rec.Body.String(), "ana@ex.com") {
		t.Fatalf("esperava 'ana@ex.com' no body, veio: %s", rec.Body.String())
	}
}

// ----------------------------------------------------------------------
// InputDefault()
// ----------------------------------------------------------------------

func TestInputDefault(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		role := req.InputDefault("role", "user")
		return res.WithText(role)
	})

	// Sem o campo — deve usar o default
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)
	if rec.Body.String() != "user" {
		t.Fatalf("esperava default 'user', veio: %s", rec.Body.String())
	}

	// Com o campo — deve usar o valor
	r2 := httptest.NewRequest(http.MethodGet, "/?role=admin", nil)
	rec2 := httptest.NewRecorder()
	app.ServeHTTP(rec2, r2)
	if rec2.Body.String() != "admin" {
		t.Fatalf("esperava 'admin', veio: %s", rec2.Body.String())
	}
}

// ----------------------------------------------------------------------
// InputInt()
// ----------------------------------------------------------------------

func TestInputInt(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		page := req.InputInt("page", 1)
		return res.WithJson(map[string]int{"page": page})
	})

	// Com valor numérico
	r := httptest.NewRequest(http.MethodGet, "/?page=5", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)
	if !strings.Contains(rec.Body.String(), "5") {
		t.Fatalf("esperava page=5, veio: %s", rec.Body.String())
	}

	// Sem valor — default
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	app.ServeHTTP(rec2, r2)
	if !strings.Contains(rec2.Body.String(), "1") {
		t.Fatalf("esperava default page=1, veio: %s", rec2.Body.String())
	}

	// Valor inválido — default
	r3 := httptest.NewRequest(http.MethodGet, "/?page=abc", nil)
	rec3 := httptest.NewRecorder()
	app.ServeHTTP(rec3, r3)
	if !strings.Contains(rec3.Body.String(), "1") {
		t.Fatalf("esperava default page=1 para valor inválido, veio: %s", rec3.Body.String())
	}
}

// ----------------------------------------------------------------------
// InputBool()
// ----------------------------------------------------------------------

func TestInputBool(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"?active=true", true},
		{"?active=1", true},
		{"?active=yes", true},
		{"?active=y", true},
		{"?active=t", true},
		{"?active=false", false},
		{"?active=0", false},
		{"?active=no", false},
		{"?active=n", false},
		{"?active=f", false},
		{"", false}, // sem campo, usa default (false)
	}

	app := New()
	app.Get("/", func(req Request, res Response) error {
		val := req.InputBool("active", false)
		return res.WithJson(map[string]bool{"active": val})
	})

	for _, tc := range tests {
		r := httptest.NewRequest(http.MethodGet, "/"+tc.query, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, r)

		var result map[string]bool
		json.Unmarshal(rec.Body.Bytes(), &result)

		if result["active"] != tc.expected {
			t.Fatalf("query=%q: esperava active=%v, veio %v", tc.query, tc.expected, result["active"])
		}
	}
}

// ----------------------------------------------------------------------
// InputFloat64()
// ----------------------------------------------------------------------

func TestInputFloat64(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		preco := req.InputFloat64("preco", 0.0)
		return res.WithJson(map[string]float64{"preco": preco})
	})

	r := httptest.NewRequest(http.MethodGet, "/?preco=29.99", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]float64
	json.Unmarshal(rec.Body.Bytes(), &result)
	if result["preco"] != 29.99 {
		t.Fatalf("esperava preco=29.99, veio: %v", result["preco"])
	}

	// Sem valor — default
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	app.ServeHTTP(rec2, r2)

	var result2 map[string]float64
	json.Unmarshal(rec2.Body.Bytes(), &result2)
	if result2["preco"] != 0.0 {
		t.Fatalf("esperava default preco=0, veio: %v", result2["preco"])
	}

	// Valor inválido — default
	r3 := httptest.NewRequest(http.MethodGet, "/?preco=abc", nil)
	rec3 := httptest.NewRecorder()
	app.ServeHTTP(rec3, r3)

	var result3 map[string]float64
	json.Unmarshal(rec3.Body.Bytes(), &result3)
	if result3["preco"] != 0.0 {
		t.Fatalf("esperava default preco=0 para valor inválido, veio: %v", result3["preco"])
	}
}

// ----------------------------------------------------------------------
// InputSlice() — arrays de JSON, form e query string
// ----------------------------------------------------------------------

func TestInputSliceFromJSON(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		tags := req.InputSlice("tags")
		return res.WithJson(map[string][]string{"tags": tags})
	})

	body := `{"tags":["go","web","api"]}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string][]string
	json.Unmarshal(rec.Body.Bytes(), &result)

	if len(result["tags"]) != 3 {
		t.Fatalf("esperava 3 tags, veio: %v", result["tags"])
	}
	if result["tags"][0] != "go" || result["tags"][1] != "web" || result["tags"][2] != "api" {
		t.Fatalf("tags inesperadas: %v", result["tags"])
	}
}

func TestInputSliceFromQueryString(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		tags := req.InputSlice("tag")
		return res.WithJson(map[string][]string{"tags": tags})
	})

	r := httptest.NewRequest(http.MethodGet, "/?tag=go&tag=web", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string][]string
	json.Unmarshal(rec.Body.Bytes(), &result)

	if len(result["tags"]) != 2 {
		t.Fatalf("esperava 2 tags, veio: %v", result["tags"])
	}
}

func TestInputSliceEmpty(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		tags := req.InputSlice("tags")
		if tags == nil {
			return res.WithText("nil")
		}
		return res.WithText("not nil")
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Body.String() != "nil" {
		t.Fatalf("esperava nil para InputSlice sem dados, veio: %s", rec.Body.String())
	}
}

// ----------------------------------------------------------------------
// GetBody()
// ----------------------------------------------------------------------

func TestGetBody(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		body, err := req.GetBody()
		if err != nil {
			return err
		}
		return res.WithText(string(body))
	})

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("raw body content"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Body.String() != "raw body content" {
		t.Fatalf("esperava 'raw body content', veio: %s", rec.Body.String())
	}
}

// ----------------------------------------------------------------------
// Has() / Filled() / Missing()
// ----------------------------------------------------------------------

func TestHas(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.WithJson(map[string]bool{
			"has_nome":  req.Has("nome"),
			"has_outro": req.Has("outro"),
		})
	})

	r := httptest.NewRequest(http.MethodGet, "/?nome=Ana", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]bool
	json.Unmarshal(rec.Body.Bytes(), &result)

	if !result["has_nome"] {
		t.Fatal("esperava Has('nome') = true")
	}
	if result["has_outro"] {
		t.Fatal("esperava Has('outro') = false")
	}
}

func TestHasWithJSON(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		return res.WithJson(map[string]bool{
			"has_nome":  req.Has("nome"),
			"has_outro": req.Has("outro"),
		})
	})

	body := `{"nome":"Ana"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]bool
	json.Unmarshal(rec.Body.Bytes(), &result)

	if !result["has_nome"] {
		t.Fatal("esperava Has('nome') = true com JSON")
	}
	if result["has_outro"] {
		t.Fatal("esperava Has('outro') = false com JSON")
	}
}

func TestFilled(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.WithJson(map[string]bool{
			"filled_nome":  req.Filled("nome"),
			"filled_vazio": req.Filled("vazio"),
			"filled_outro": req.Filled("outro"),
		})
	})

	r := httptest.NewRequest(http.MethodGet, "/?nome=Ana&vazio=", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]bool
	json.Unmarshal(rec.Body.Bytes(), &result)

	if !result["filled_nome"] {
		t.Fatal("esperava Filled('nome') = true")
	}
	if result["filled_vazio"] {
		t.Fatal("esperava Filled('vazio') = false para campo vazio")
	}
	if result["filled_outro"] {
		t.Fatal("esperava Filled('outro') = false para campo ausente")
	}
}

func TestMissing(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.WithJson(map[string]bool{
			"missing_nome":  req.Missing("nome"),
			"missing_outro": req.Missing("outro"),
		})
	})

	r := httptest.NewRequest(http.MethodGet, "/?nome=Ana", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]bool
	json.Unmarshal(rec.Body.Bytes(), &result)

	if result["missing_nome"] {
		t.Fatal("esperava Missing('nome') = false")
	}
	if !result["missing_outro"] {
		t.Fatal("esperava Missing('outro') = true")
	}
}

// ----------------------------------------------------------------------
// All() / Only() / Except()
// ----------------------------------------------------------------------

func TestAll(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		return res.WithJson(req.All())
	})

	body := `{"nome":"Ana","email":"ana@ex.com"}`
	r := httptest.NewRequest(http.MethodPost, "/?page=1", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result)

	if result["nome"] != "Ana" {
		t.Fatalf("esperava nome=Ana, veio: %v", result["nome"])
	}
	if result["email"] != "ana@ex.com" {
		t.Fatalf("esperava email=ana@ex.com, veio: %v", result["email"])
	}
	if result["page"] != "1" {
		t.Fatalf("esperava page=1 da query, veio: %v", result["page"])
	}
}

func TestOnly(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		return res.WithJson(req.Only("nome", "email"))
	})

	body := `{"nome":"Ana","email":"ana@ex.com","senha":"123456"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result)

	if result["nome"] != "Ana" {
		t.Fatal("esperava nome em Only()")
	}
	if result["email"] != "ana@ex.com" {
		t.Fatal("esperava email em Only()")
	}
	if _, ok := result["senha"]; ok {
		t.Fatal("não esperava senha em Only('nome', 'email')")
	}
}

func TestExcept(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		return res.WithJson(req.Except("senha", "token"))
	})

	body := `{"nome":"Ana","email":"ana@ex.com","senha":"123456","token":"abc"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result)

	if result["nome"] != "Ana" {
		t.Fatal("esperava nome em Except()")
	}
	if _, ok := result["senha"]; ok {
		t.Fatal("não esperava senha em Except('senha', 'token')")
	}
	if _, ok := result["token"]; ok {
		t.Fatal("não esperava token em Except('senha', 'token')")
	}
}

// ----------------------------------------------------------------------
// Merge()
// ----------------------------------------------------------------------

func TestMerge(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		dados := req.Merge(map[string]interface{}{
			"status": "ativo",
			"role":   "user",
		})
		return res.WithJson(dados)
	})

	body := `{"nome":"Ana"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result)

	if result["nome"] != "Ana" {
		t.Fatal("esperava nome original em Merge()")
	}
	if result["status"] != "ativo" {
		t.Fatal("esperava status=ativo do merge")
	}
	if result["role"] != "user" {
		t.Fatal("esperava role=user do merge")
	}
}

func TestMergeOverwrite(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		dados := req.Merge(map[string]interface{}{
			"nome": "Sobrescrito",
		})
		return res.WithJson(dados)
	})

	body := `{"nome":"Ana"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result)

	if result["nome"] != "Sobrescrito" {
		t.Fatalf("esperava nome sobrescrito pelo Merge, veio: %v", result["nome"])
	}
}

// ----------------------------------------------------------------------
// InputJSON()
// ----------------------------------------------------------------------

func TestInputJSON(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		dados, err := req.InputJSON()
		if err != nil {
			return res.Status(400).WithText(err.Error())
		}
		return res.WithJson(dados)
	})

	body := `{"nome":"Ana","idade":25}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result)

	if result["nome"] != "Ana" {
		t.Fatalf("esperava nome=Ana, veio: %v", result["nome"])
	}
}

func TestInputJSONEmpty(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		dados, err := req.InputJSON()
		if err != nil {
			return res.Status(400).WithText(err.Error())
		}
		return res.WithJson(map[string]int{"total": len(dados)})
	})

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]int
	json.Unmarshal(rec.Body.Bytes(), &result)

	if result["total"] != 0 {
		t.Fatalf("esperava map vazio para body vazio, veio: %d campos", result["total"])
	}
}

// ----------------------------------------------------------------------
// BindJson()
// ----------------------------------------------------------------------

func TestBindJson(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		var payload struct {
			Nome  string `json:"nome"`
			Email string `json:"email"`
		}
		if err := req.BindJson(&payload); err != nil {
			return res.Status(400).WithText("JSON inválido")
		}
		return res.WithJson(payload)
	})

	body := `{"nome":"Ana","email":"ana@ex.com"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Code != 200 {
		t.Fatalf("esperava 200, veio: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Ana") {
		t.Fatalf("esperava 'Ana' no body, veio: %s", rec.Body.String())
	}
}

// ----------------------------------------------------------------------
// CORS Middleware
// ----------------------------------------------------------------------

func TestCORSPreflight(t *testing.T) {
	app := New()
	app.Use(CORS())
	app.Get("/api/data", func(req Request, res Response) error {
		return res.WithText("ok")
	})

	// Pedido OPTIONS (preflight)
	r := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	r.Header.Set("Origin", "https://frontend.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("esperava 204 no preflight, veio: %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("esperava Allow-Origin=*, veio: %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("esperava cabeçalho Allow-Methods no preflight")
	}
}

func TestCORSNormalRequest(t *testing.T) {
	app := New()
	app.Use(CORS())
	app.Get("/api/data", func(req Request, res Response) error {
		return res.WithText("ok")
	})

	r := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	r.Header.Set("Origin", "https://frontend.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio: %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("esperava Allow-Origin=* no pedido normal com Origin, veio: %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSCustomConfig(t *testing.T) {
	app := New()
	app.Use(CORS(CORSConfig{
		AllowOrigins:     []string{"https://meusite.com"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))
	app.Get("/api/data", func(req Request, res Response) error {
		return res.WithText("ok")
	})

	// Origem permitida
	r := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	r.Header.Set("Origin", "https://meusite.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://meusite.com" {
		t.Fatalf("esperava Allow-Origin=https://meusite.com, veio: %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("esperava Allow-Credentials=true")
	}

	// Origem não permitida
	r2 := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	r2.Header.Set("Origin", "https://outro.com")
	rec2 := httptest.NewRecorder()
	app.ServeHTTP(rec2, r2)

	if rec2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("não esperava Allow-Origin para origem não permitida, veio: %s", rec2.Header().Get("Access-Control-Allow-Origin"))
	}
}

// ----------------------------------------------------------------------
// Middleware chain
// ----------------------------------------------------------------------

func TestMultipleMiddlewares(t *testing.T) {
	app := New()
	var order []string

	app.Use(func(next HandlerFunc) HandlerFunc {
		return func(req Request, res Response) error {
			order = append(order, "mw1")
			return next(req, res)
		}
	})
	app.Use(func(next HandlerFunc) HandlerFunc {
		return func(req Request, res Response) error {
			order = append(order, "mw2")
			return next(req, res)
		}
	})
	app.Get("/", func(req Request, res Response) error {
		order = append(order, "handler")
		return res.WithText("done")
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	expected := "mw1,mw2,handler"
	if strings.Join(order, ",") != expected {
		t.Fatalf("ordem inesperada: %v, esperava: %s", order, expected)
	}
}

// ----------------------------------------------------------------------
// Response helpers
// ----------------------------------------------------------------------

func TestResponseStatus(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.Status(201).WithJson(map[string]string{"ok": "true"})
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Code != 201 {
		t.Fatalf("esperava 201, veio: %d", rec.Code)
	}
}

func TestResponseRedirect(t *testing.T) {
	app := New()
	app.Get("/old", func(req Request, res Response) error {
		return res.Redirect("/new")
	})

	r := httptest.NewRequest(http.MethodGet, "/old", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Code != http.StatusFound {
		t.Fatalf("esperava 302, veio: %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/new" {
		t.Fatalf("esperava Location=/new, veio: %s", rec.Header().Get("Location"))
	}
}

func TestResponseNoContent(t *testing.T) {
	app := New()
	app.Delete("/items/:id", func(req Request, res Response) error {
		return res.NoContent()
	})

	r := httptest.NewRequest(http.MethodDelete, "/items/1", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("esperava 204, veio: %d", rec.Code)
	}
}

func TestResponseSendStatus(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.SendStatus(404)
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Code != 404 {
		t.Fatalf("esperava 404, veio: %d", rec.Code)
	}
	if rec.Body.String() != "Not Found" {
		t.Fatalf("esperava 'Not Found', veio: %s", rec.Body.String())
	}
}

func TestResponseSetHeader(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.SetHeader("X-Custom", "valor123").WithText("ok")
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Header().Get("X-Custom") != "valor123" {
		t.Fatalf("esperava X-Custom=valor123, veio: %s", rec.Header().Get("X-Custom"))
	}
}

func TestResponseCookie(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.Cookie(&http.Cookie{
			Name:  "session",
			Value: "abc123",
		}).WithText("ok")
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session" && c.Value == "abc123" {
			found = true
		}
	}
	if !found {
		t.Fatal("esperava cookie 'session=abc123' na resposta")
	}
}

func TestResponseClearCookie(t *testing.T) {
	app := New()
	app.Get("/logout", func(req Request, res Response) error {
		return res.ClearCookie("session").WithText("ok")
	})

	r := httptest.NewRequest(http.MethodGet, "/logout", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("esperava cookie 'session' com MaxAge negativo para limpeza")
	}
}

// ----------------------------------------------------------------------
// Request helpers (cabeçalhos, IP, etc.)
// ----------------------------------------------------------------------

func TestRequestIP(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.WithText(req.IP())
	})

	// Com X-Forwarded-For
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Body.String() != "1.2.3.4" {
		t.Fatalf("esperava IP 1.2.3.4, veio: %s", rec.Body.String())
	}
}

func TestRequestContentType(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		return res.WithText(req.ContentType())
	})

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Body.String() != "application/json" {
		t.Fatalf("esperava 'application/json', veio: %s", rec.Body.String())
	}
}

func TestRequestIs(t *testing.T) {
	app := New()
	app.Post("/", func(req Request, res Response) error {
		return res.WithJson(map[string]bool{
			"is_json": req.Is("application/json"),
			"is_html": req.Is("text/html"),
		})
	})

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]bool
	json.Unmarshal(rec.Body.Bytes(), &result)

	if !result["is_json"] {
		t.Fatal("esperava Is('application/json') = true")
	}
	if result["is_html"] {
		t.Fatal("esperava Is('text/html') = false")
	}
}

func TestRequestXHR(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.WithJson(map[string]bool{"xhr": req.XHR()})
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Requested-With", "XMLHttpRequest")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	var result map[string]bool
	json.Unmarshal(rec.Body.Bytes(), &result)

	if !result["xhr"] {
		t.Fatal("esperava XHR() = true")
	}
}

func TestRequestUserAgent(t *testing.T) {
	app := New()
	app.Get("/", func(req Request, res Response) error {
		return res.WithText(req.UserAgent())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("User-Agent", "KiteBot/1.0")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Body.String() != "KiteBot/1.0" {
		t.Fatalf("esperava User-Agent 'KiteBot/1.0', veio: %s", rec.Body.String())
	}
}

func TestNotFoundHandler(t *testing.T) {
	app := New()

	r := httptest.NewRequest(http.MethodGet, "/nao-existe", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperava 404 para rota inexistente, veio: %d", rec.Code)
	}
}
