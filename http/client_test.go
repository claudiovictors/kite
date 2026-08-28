package http

import (
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Testa uma requisição GET simples.
func TestGet(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodGet {
			t.Errorf("Esperado método GET, recebido %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	resp, err := NewRequest().Get(server.URL)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Esperado status %d, recebido %d", nethttp.StatusOK, resp.Status())
	}
	if resp.Body() != "ok" {
		t.Errorf("Esperado corpo 'ok', recebido '%s'", resp.Body())
	}
}

// Testa requisição POST com corpo JSON.
func TestPost(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			t.Errorf("Esperado método POST, recebido %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "value") {
			t.Errorf("Corpo da requisição inesperado: %s", string(body))
		}
		w.WriteHeader(nethttp.StatusCreated)
	}))
	defer server.Close()

	data := map[string]string{"key": "value"}
	resp, err := NewRequest().Post(server.URL, data)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
	if resp.Status() != nethttp.StatusCreated {
		t.Errorf("Esperado status %d, recebido %d", nethttp.StatusCreated, resp.Status())
	}
}

// Testa requisição PUT com corpo JSON.
func TestPut(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPut {
			t.Errorf("Esperado método PUT, recebido %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, err := NewRequest().Put(server.URL, map[string]string{"update": "true"})
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Esperado status %d, recebido %d", nethttp.StatusOK, resp.Status())
	}
}

// Testa requisição PATCH com corpo JSON.
func TestPatch(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPatch {
			t.Errorf("Esperado método PATCH, recebido %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, err := NewRequest().Patch(server.URL, map[string]string{"patch": "true"})
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Esperado status %d, recebido %d", nethttp.StatusOK, resp.Status())
	}
}

// Testa requisição DELETE com e sem corpo.
func TestDelete(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodDelete {
			t.Errorf("Esperado método DELETE, recebido %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusNoContent)
	}))
	defer server.Close()

	// Sem corpo
	resp, err := NewRequest().Delete(server.URL)
	if err != nil {
		t.Fatalf("Erro inesperado no Delete sem corpo: %v", err)
	}
	if resp.Status() != nethttp.StatusNoContent {
		t.Errorf("Esperado status %d, recebido %d", nethttp.StatusNoContent, resp.Status())
	}

	// Com corpo
	respBody, errBody := NewRequest().Delete(server.URL, map[string]string{"id": "1"})
	if errBody != nil {
		t.Fatalf("Erro inesperado no Delete com corpo: %v", errBody)
	}
	if respBody.Status() != nethttp.StatusNoContent {
		t.Errorf("Esperado status %d, recebido %d", nethttp.StatusNoContent, respBody.Status())
	}
}

// Testa requisição HEAD.
func TestHead(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodHead {
			t.Errorf("Esperado método HEAD, recebido %s", r.Method)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, err := NewRequest().Head(server.URL)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Esperado status %d, recebido %d", nethttp.StatusOK, resp.Status())
	}
}

// Testa requisição OPTIONS.
func TestOptions(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodOptions {
			t.Errorf("Esperado método OPTIONS, recebido %s", r.Method)
		}
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		w.WriteHeader(nethttp.StatusNoContent)
	}))
	defer server.Close()

	resp, err := NewRequest().Options(server.URL)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
	if resp.Status() != nethttp.StatusNoContent {
		t.Errorf("Esperado status %d, recebido %d", nethttp.StatusNoContent, resp.Status())
	}
	if resp.Header("Allow") != "GET, POST, OPTIONS" {
		t.Errorf("Cabeçalho Allow inesperado: %s", resp.Header("Allow"))
	}
}

// Testa o envio de Token Bearer.
func TestWithToken(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer meu-token-secreto" {
			t.Errorf("Header Authorization inesperado: %s", auth)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest().WithToken("meu-token-secreto").Get(server.URL)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
}

// Testa autenticação básica (Basic Auth).
func TestWithBasicAuth(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "1234" {
			t.Errorf("Basic Auth inesperado: user=%s, pass=%s", user, pass)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest().WithBasicAuth("admin", "1234").Get(server.URL)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
}

// Testa envio de múltiplos cabeçalhos.
func TestWithHeaders(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Header.Get("X-Custom") != "valor1" || r.Header.Get("X-Outro") != "valor2" {
			t.Errorf("Cabeçalhos ausentes ou incorretos")
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().
		WithHeader("X-Custom", "valor1").
		WithHeaders(map[string]string{"X-Outro": "valor2"})
	
	_, err := req.Get(server.URL)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
}

// Testa envio de parâmetros na Query String.
func TestWithQuery(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Query().Get("busca") != "termo" || r.URL.Query().Get("pagina") != "2" {
			t.Errorf("Query params incorretos: %s", r.URL.RawQuery)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().
		WithQuery("busca", "termo").
		WithQueryParams(map[string]string{"pagina": "2"})

	_, err := req.Get(server.URL)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
}

// Testa o envio de Cookies.
func TestWithCookies(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		c1, err1 := r.Cookie("sessao")
		c2, err2 := r.Cookie("tema")
		if err1 != nil || err2 != nil || c1.Value != "123" || c2.Value != "escuro" {
			t.Errorf("Cookies ausentes ou incorretos")
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().
		WithCookie("sessao", "123").
		WithCookies(map[string]string{"tema": "escuro"})

	_, err := req.Get(server.URL)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
}

// Testa prefixação de BaseURL.
func TestBaseURL(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path != "/api/users" {
			t.Errorf("Path inesperado: %s", r.URL.Path)
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().BaseURL(server.URL + "/api/")
	_, err := req.Get("/users")
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
}

// Testa envio de formulário POST (application/x-www-form-urlencoded).
func TestPostForm(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type inesperado: %s", r.Header.Get("Content-Type"))
		}
		r.ParseForm()
		if r.PostForm.Get("nome") != "teste" {
			t.Errorf("Campo formulário incorreto: %s", r.PostForm.Get("nome"))
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest().PostForm(server.URL, map[string]string{"nome": "teste"})
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
}

// Testa upload multipart com arquivos e campos.
func TestPostMultipart(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Content-Type inesperado: %s", r.Header.Get("Content-Type"))
		}
		r.ParseMultipartForm(10 << 20)
		if r.FormValue("descricao") != "documento" {
			t.Errorf("Campo multipart incorreto: %s", r.FormValue("descricao"))
		}
		file, _, err := r.FormFile("arquivo")
		if err != nil {
			t.Fatalf("Erro ao ler arquivo: %v", err)
		}
		defer file.Close()
		content, _ := io.ReadAll(file)
		if string(content) != "conteudo-do-arquivo" {
			t.Errorf("Conteúdo do arquivo incorreto: %s", string(content))
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	fields := map[string]string{"descricao": "documento"}
	files := map[string][]byte{"arquivo": []byte("conteudo-do-arquivo")}
	_, err := NewRequest().PostMultipart(server.URL, fields, files)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}
}

// Testa desativar redirecionamentos automáticos.
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
			t.Logf("Aviso: o erro retornado contém ErrUseLastResponse, o que é esperado no Go net/http: %v", err)
		}
	}
	
	if resp == nil {
		t.Fatalf("A resposta não deveria ser nula mesmo com erro de redirecionamento evitado")
	}

	if resp.Status() != nethttp.StatusFound {
		t.Errorf("Esperado status %d (Redirect), recebido %d", nethttp.StatusFound, resp.Status())
	}
	if !resp.Redirect() {
		t.Errorf("O método Redirect() deveria retornar true")
	}
}

// Testa helpers de verificação de status (Ok, Failed, ServerError, etc).
func TestResponseOk(t *testing.T) {
	cases := []struct {
		status    int
		check     func(r *ClientResponse) bool
		shouldBe  bool
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
			t.Errorf("Falha na validação de status para código %d", c.status)
		}
		server.Close()
	}
}

// Testa conversão da resposta para JSON.
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
		t.Fatalf("Erro ao decodificar JSON: %v", err)
	}
	if dest.Mensagem != "sucesso" {
		t.Errorf("Valor decodificado incorreto: %s", dest.Mensagem)
	}
}

// Testa conversão da resposta para Map.
func TestResponseMap(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Write([]byte(`{"chave": "valor"}`))
	}))
	defer server.Close()

	resp, _ := NewRequest().Get(server.URL)
	m, err := resp.Map()
	if err != nil {
		t.Fatalf("Erro ao converter para mapa: %v", err)
	}
	if m["chave"] != "valor" {
		t.Errorf("Valor no mapa incorreto: %v", m["chave"])
	}
}

// Testa conversão da resposta para array de objetos (Collect).
func TestResponseCollect(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Write([]byte(`[{"id": 1}, {"id": 2}]`))
	}))
	defer server.Close()

	resp, _ := NewRequest().Get(server.URL)
	arr, err := resp.Collect()
	if err != nil {
		t.Fatalf("Erro ao usar Collect: %v", err)
	}
	if len(arr) != 2 || arr[0]["id"].(float64) != 1 {
		t.Errorf("Array decodificado incorretamente")
	}
}

// Testa leitura dos cabeçalhos da resposta.
func TestResponseHeaders(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("X-RateLimit", "100")
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, _ := NewRequest().Get(server.URL)
	if resp.Header("X-RateLimit") != "100" {
		t.Errorf("Cabeçalho recebido incorreto: %s", resp.Header("X-RateLimit"))
	}
	if len(resp.Headers()) == 0 {
		t.Errorf("Headers() não deveria estar vazio")
	}
}

// Testa leitura de Cookies da resposta.
func TestResponseCookies(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		nethttp.SetCookie(w, &nethttp.Cookie{Name: "token", Value: "abc"})
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, _ := NewRequest().Get(server.URL)
	if resp.Cookie("token") != "abc" {
		t.Errorf("Cookie recebido incorreto: %s", resp.Cookie("token"))
	}
	if len(resp.Cookies()) == 0 {
		t.Errorf("Cookies() não deveria estar vazio")
	}
}

// Testa atalho global Get.
func TestGlobalGet(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	resp, err := Get(server.URL)
	if err != nil || resp.Status() != nethttp.StatusOK {
		t.Errorf("Erro no Get global")
	}
}

// Testa atalho global Post.
func TestGlobalPost(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			t.Errorf("Esperado método POST")
		}
		w.WriteHeader(nethttp.StatusCreated)
	}))
	defer server.Close()

	resp, err := Post(server.URL, map[string]string{"a": "b"})
	if err != nil || resp.Status() != nethttp.StatusCreated {
		t.Errorf("Erro no Post global")
	}
}

// Testa configuração de Timeout.
func TestTimeout(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer server.Close()

	req := NewRequest().Timeout(10 * time.Millisecond)
	_, err := req.Get(server.URL)
	if err == nil {
		t.Fatalf("Esperava um erro de timeout, mas a requisição obteve sucesso")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Esperado erro contendo timeout/deadline, recebido: %v", err)
	}
}

// Testa a mecânica de Retry automático para falhas 5xx.
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
		t.Fatalf("Erro inesperado com retries: %v", err)
	}
	if resp.Status() != nethttp.StatusOK {
		t.Errorf("Esperado status OK após retries, recebido %d", resp.Status())
	}
	if tentativas != 3 {
		t.Errorf("Esperadas 3 tentativas, feitas %d", tentativas)
	}
}
