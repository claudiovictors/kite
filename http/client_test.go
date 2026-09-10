package http

import (
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Tests a simple GET request.
func TestGet(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	resp, err := NewRequest().Get(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Expected status %d, got %d", nethttp.StatusOK, resp.Status())
	}
	if resp.Body() != "ok" {
		t.Errorf("Expected body 'ok', got '%s'", resp.Body())
	}
}

// Tests a POST request with a JSON body.
func TestPost(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "value") {
			t.Errorf("Unexpected request body: %s", string(body))
		}
		w.WriteHeader(nethttp.StatusCreated)
	}))
	defer server.Close()

	data := map[string]string{"key": "value"}
	resp, err := NewRequest().Post(server.URL, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Status() != nethttp.StatusCreated {
		t.Errorf("Expected status %d, got %d", nethttp.StatusCreated, resp.Status())
	}
}

// Tests a PUT request with a JSON body.
func TestPut(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, err := NewRequest().Put(server.URL, map[string]string{"update": "true"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Expected status %d, got %d", nethttp.StatusOK, resp.Status())
	}
}

// Tests a PATCH request with a JSON body.
func TestPatch(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPatch {
			t.Errorf("Expected PATCH method, got %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, err := NewRequest().Patch(server.URL, map[string]string{"patch": "true"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Expected status %d, got %d", nethttp.StatusOK, resp.Status())
	}
}

// Tests DELETE request with and without a body.
func TestDelete(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusNoContent)
	}))
	defer server.Close()

	// Without body
	resp, err := NewRequest().Delete(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error on Delete without body: %v", err)
	}
	if resp.Status() != nethttp.StatusNoContent {
		t.Errorf("Expected status %d, got %d", nethttp.StatusNoContent, resp.Status())
	}

	// With body
	respBody, errBody := NewRequest().Delete(server.URL, map[string]string{"id": "1"})
	if errBody != nil {
		t.Fatalf("Unexpected error on Delete with body: %v", errBody)
	}
	if respBody.Status() != nethttp.StatusNoContent {
		t.Errorf("Expected status %d, got %d", nethttp.StatusNoContent, respBody.Status())
	}
}

// Tests a HEAD request.
func TestHead(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodHead {
			t.Errorf("Expected HEAD method, got %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, err := NewRequest().Head(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Expected status %d, got %d", nethttp.StatusOK, resp.Status())
	}
}

// Tests an OPTIONS request.
func TestOptions(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodOptions {
			t.Errorf("Expected OPTIONS method, got %s", r.Method)
		}
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		w.WriteHeader(nethttp.StatusNoContent)
	}))
	defer server.Close()

	resp, err := NewRequest().Options(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Status() != nethttp.StatusNoContent {
		t.Errorf("Expected status %d, got %d", nethttp.StatusNoContent, resp.Status())
	}
	if resp.Header("Allow") != "GET, POST, OPTIONS" {
		t.Errorf("Unexpected Allow header: %s", resp.Header("Allow"))
	}
}

// Tests sending a Bearer token.
func TestWithToken(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer meu-token-secreto" {
			t.Errorf("Unexpected Authorization header: %s", auth)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest().WithToken("meu-token-secreto").Get(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// Tests HTTP Basic Auth.
func TestWithBasicAuth(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "1234" {
			t.Errorf("Unexpected Basic Auth: user=%s, pass=%s", user, pass)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest().WithBasicAuth("admin", "1234").Get(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// Tests sending multiple headers.
func TestWithHeaders(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Header.Get("X-Custom") != "valor1" || r.Header.Get("X-Outro") != "valor2" {
			t.Errorf("Missing or incorrect headers")
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().
		WithHeader("X-Custom", "valor1").
		WithHeaders(map[string]string{"X-Outro": "valor2"})

	_, err := req.Get(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// Tests sending query string parameters.
func TestWithQuery(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Query().Get("busca") != "termo" || r.URL.Query().Get("pagina") != "2" {
			t.Errorf("Incorrect query params: %s", r.URL.RawQuery)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().
		WithQuery("busca", "termo").
		WithQueryParams(map[string]string{"pagina": "2"})

	_, err := req.Get(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// Tests sending Cookies.
func TestWithCookies(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		c1, err1 := r.Cookie("sessao")
		c2, err2 := r.Cookie("tema")
		if err1 != nil || err2 != nil || c1.Value != "123" || c2.Value != "escuro" {
			t.Errorf("Missing or incorrect cookies")
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().
		WithCookie("sessao", "123").
		WithCookies(map[string]string{"tema": "escuro"})

	_, err := req.Get(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// Tests BaseURL prefixing.
func TestBaseURL(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path != "/api/users" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().BaseURL(server.URL + "/api/")
	_, err := req.Get("/users")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// Tests sending a POST form (application/x-www-form-urlencoded).
func TestPostForm(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Unexpected Content-Type: %s", r.Header.Get("Content-Type"))
		}
		r.ParseForm()
		if r.PostForm.Get("nome") != "teste" {
			t.Errorf("Incorrect form field: %s", r.PostForm.Get("nome"))
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest().PostForm(server.URL, map[string]string{"nome": "teste"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// Tests multipart upload with files and fields.
func TestPostMultipart(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Unexpected Content-Type: %s", r.Header.Get("Content-Type"))
		}
		r.ParseMultipartForm(10 << 20)
		if r.FormValue("descricao") != "documento" {
			t.Errorf("Incorrect multipart field: %s", r.FormValue("descricao"))
		}
		file, _, err := r.FormFile("arquivo")
		if err != nil {
			t.Fatalf("Error reading file: %v", err)
		}
		defer file.Close()
		content, _ := io.ReadAll(file)
		if string(content) != "conteudo-do-arquivo" {
			t.Errorf("Incorrect file content: %s", string(content))
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	fields := map[string]string{"descricao": "documento"}
	files := map[string][]byte{"arquivo": []byte("conteudo-do-arquivo")}
	_, err := NewRequest().PostMultipart(server.URL, fields, files)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// Tests disabling automatic redirects.
func TestWithoutRedirects(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path == "/" {
			nethttp.Redirect(w, r, "/destino", nethttp.StatusFound)
			return
		}
		if r.URL.Path == "/destino" {
			w.WriteHeader(nethttp.StatusOK)
			return
		}
	}))
	defer server.Close()

	resp, err := NewRequest().WithoutRedirects().Get(server.URL)
	if err != nil {
		// client should return response despite ErrUseLastResponse
		if !strings.Contains(err.Error(), nethttp.ErrUseLastResponse.Error()) {
			t.Logf("Warning: the returned error contains ErrUseLastResponse, which is expected in Go net/http: %v", err)
		}
	}

	if resp == nil {
		t.Fatalf("Response should not be nil even with redirect-avoided error")
	}

	if resp.Status() != nethttp.StatusFound {
		t.Errorf("Expected status %d (Redirect), got %d", nethttp.StatusFound, resp.Status())
	}
	if !resp.Redirect() {
		t.Errorf("Redirect() method should return true")
	}
}

// Tests status check helpers (Ok, Failed, ServerError, etc).
func TestResponseOk(t *testing.T) {
	cases := []struct {
		status   int
		check    func(r *ClientResponse) bool
		shouldBe bool
	}{
		{200, func(r *ClientResponse) bool { return r.Ok() }, true},
		{201, func(r *ClientResponse) bool { return r.Successful() }, true},
		{400, func(r *ClientResponse) bool { return r.Failed() }, true},
		{500, func(r *ClientResponse) bool { return r.ServerError() }, true},
		{404, func(r *ClientResponse) bool { return r.ClientError() }, true},
		{401, func(r *ClientResponse) bool { return r.Unauthorized() }, true},
		{403, func(r *ClientResponse) bool { return r.Forbidden() }, true},
		{404, func(r *ClientResponse) bool { return r.NotFound() }, true},
	}

	for _, c := range cases {
		server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			w.WriteHeader(c.status)
		}))

		resp, _ := NewRequest().Get(server.URL)
		if c.check(resp) != c.shouldBe {
			t.Errorf("Status validation failed for code %d", c.status)
		}
		server.Close()
	}
}

// Tests decoding the response as JSON.
func TestResponseJson(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"mensagem": "sucesso"}`))
	}))
	defer server.Close()

	resp, _ := NewRequest().Get(server.URL)
	var dest struct {
		Mensagem string `json:"mensagem"`
	}
	if err := resp.Json(&dest); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}
	if dest.Mensagem != "sucesso" {
		t.Errorf("Incorrect decoded value: %s", dest.Mensagem)
	}
}

// Tests decoding the response as a Map.
func TestResponseMap(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Write([]byte(`{"chave": "valor"}`))
	}))
	defer server.Close()

	resp, _ := NewRequest().Get(server.URL)
	m, err := resp.Map()
	if err != nil {
		t.Fatalf("Error converting to map: %v", err)
	}
	if m["chave"] != "valor" {
		t.Errorf("Incorrect value in map: %v", m["chave"])
	}
}

// Tests decoding the response as a slice of objects (Collect).
func TestResponseCollect(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Write([]byte(`[{"id": 1}, {"id": 2}]`))
	}))
	defer server.Close()

	resp, _ := NewRequest().Get(server.URL)
	arr, err := resp.Collect()
	if err != nil {
		t.Fatalf("Error using Collect: %v", err)
	}
	if len(arr) != 2 || arr[0]["id"].(float64) != 1 {
		t.Errorf("Array decoded incorrectly")
	}
}

// Tests reading response headers.
func TestResponseHeaders(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("X-RateLimit", "100")
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, _ := NewRequest().Get(server.URL)
	if resp.Header("X-RateLimit") != "100" {
		t.Errorf("Incorrect received header: %s", resp.Header("X-RateLimit"))
	}
	if len(resp.Headers()) == 0 {
		t.Errorf("Headers() should not be empty")
	}
}

// Tests reading response Cookies.
func TestResponseCookies(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		nethttp.SetCookie(w, &nethttp.Cookie{Name: "token", Value: "abc"})
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, _ := NewRequest().Get(server.URL)
	if resp.Cookie("token") != "abc" {
		t.Errorf("Incorrect received cookie: %s", resp.Cookie("token"))
	}
	if len(resp.Cookies()) == 0 {
		t.Errorf("Cookies() should not be empty")
	}
}

// Tests global Get shortcut.
func TestGlobalGet(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, err := Get(server.URL)
	if err != nil || resp.Status() != nethttp.StatusOK {
		t.Errorf("Error in global Get")
	}
}

// Tests global Post shortcut.
func TestGlobalPost(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			t.Errorf("Expected POST method")
		}
		w.WriteHeader(nethttp.StatusCreated)
	}))
	defer server.Close()

	resp, err := Post(server.URL, map[string]string{"a": "b"})
	if err != nil || resp.Status() != nethttp.StatusCreated {
		t.Errorf("Error in global Post")
	}
}

// Tests Timeout configuration.
func TestTimeout(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().Timeout(10 * time.Millisecond)
	_, err := req.Get(server.URL)
	if err == nil {
		t.Fatalf("Expected a timeout error, but request succeeded")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected error containing timeout/deadline, got: %v", err)
	}
}

// Tests the automatic Retry mechanism for 5xx failures.
func TestRetry(t *testing.T) {
	var tentativas int
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		tentativas++
		if tentativas < 3 {
			w.WriteHeader(nethttp.StatusInternalServerError)
			return
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().Retry(3, 10*time.Millisecond)
	resp, err := req.Get(server.URL)
	if err != nil {
		t.Fatalf("Unexpected error with retries: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Expected OK status after retries, got %d", resp.Status())
	}
	if tentativas != 3 {
		t.Errorf("Expected 3 attempts, got %d", tentativas)
	}
}
