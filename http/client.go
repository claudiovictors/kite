// Package http fornece um cliente HTTP fluente para consumir APIs
// externas, inspirado na facade Http:: do Laravel. Permite construir
// pedidos de forma encadeável e devolver respostas com helpers para
// JSON, texto, status, etc.
//
// Exemplo rápido:
//
//  resp, err := http.Get("https://api.exemplo.com/users")
//  if err != nil {
//      log.Fatal(err)
//  }
//  fmt.Println(resp.Status())   // 200
//  fmt.Println(resp.Body())     // corpo como string
//
//  var users []User
//  resp.Json(&users)            // decode JSON
//
// Exemplo com builder:
//
//  resp, err := http.NewRequest().
//      WithToken("meu-jwt-token").
//      WithHeader("X-Custom", "valor").
//      Timeout(10 * time.Second).
//      Post("https://api.exemplo.com/users", map[string]interface{}{
//          "nome":  "Ana",
//          "email": "ana@exemplo.com",
//      })
package http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ----------------------------------------------------------------------
// PendingRequest: construtor fluente de pedidos HTTP
// ----------------------------------------------------------------------

/**
 * PendingRequest acumula a configuração de um pedido HTTP antes de o
 * enviar. Use NewRequest() para criar um, ou os atalhos globais Get(),
 * Post(), etc., para pedidos simples.
 */
type PendingRequest struct {
	client  *http.Client
	headers map[string]string
	query   url.Values
	timeout time.Duration
	baseURL string
	cookies []*http.Cookie

	// retry
	retries int
	retryMs time.Duration
}

/**
 * NewRequest cria um PendingRequest com valores sane defaults (Timeout padrão de 30s).
 *
 * Exemplo:
 *  req := http.NewRequest()
 *
 * @return *PendingRequest
 */
func NewRequest() *PendingRequest {
	return &PendingRequest{
		client:  &http.Client{Timeout: 30 * time.Second},
		headers: map[string]string{},
		query:   url.Values{},
		timeout: 30 * time.Second,
	}
}

/**
 * BaseURL define uma URL base que será prefixada a cada pedido,
 * útil quando todos os endpoints compartilham o mesmo domínio.
 *
 * Exemplo:
 *  client := http.NewRequest().BaseURL("https://api.exemplo.com")
 *  resp, _ := client.Get("/users")       // GET https://api.exemplo.com/users
 *
 * @param base string
 * @return *PendingRequest
 */
func (pr *PendingRequest) BaseURL(base string) *PendingRequest {
	pr.baseURL = strings.TrimRight(base, "/")
	return pr
}

/**
 * Timeout define o tempo máximo para o pedido completo (conexão + leitura do corpo).
 * O padrão é 30 segundos.
 *
 * Exemplo:
 *  req.Timeout(10 * time.Second)
 *
 * @param d time.Duration
 * @return *PendingRequest
 */
func (pr *PendingRequest) Timeout(d time.Duration) *PendingRequest {
	pr.timeout = d
	pr.client.Timeout = d
	return pr
}

/**
 * WithHeader adiciona um cabeçalho ao pedido. É encadeável.
 *
 * Exemplo:
 *  req.WithHeader("X-Custom-Header", "valor")
 *
 * @param key string
 * @param value string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithHeader(key, value string) *PendingRequest {
	pr.headers[key] = value
	return pr
}

/**
 * WithHeaders adiciona múltiplos cabeçalhos de uma vez através de um map.
 *
 * Exemplo:
 *  req.WithHeaders(map[string]string{"Accept": "application/json", "X-App": "Mobile"})
 *
 * @param headers map[string]string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithHeaders(headers map[string]string) *PendingRequest {
	for k, v := range headers {
		pr.headers[k] = v
	}
	return pr
}

/**
 * WithToken adiciona um cabeçalho Authorization: Bearer <token>.
 * Equivalente ao Http::withToken() do Laravel.
 *
 * Exemplo:
 *  req.WithToken("eyJhbGciOi...")
 *
 * @param token string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithToken(token string) *PendingRequest {
	pr.headers["Authorization"] = "Bearer " + token
	return pr
}

/**
 * WithBasicAuth adiciona autenticação HTTP Basic ao cabeçalho Authorization.
 * Equivalente ao Http::withBasicAuth() do Laravel.
 *
 * Exemplo:
 *  req.WithBasicAuth("usuario", "senha123")
 *
 * @param user string
 * @param password string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithBasicAuth(user, password string) *PendingRequest {
	pr.headers["Authorization"] = "Basic " + basicAuth(user, password)
	return pr
}

/**
 * Accept define o cabeçalho Accept do pedido.
 *
 * Exemplo:
 *  req.Accept("text/xml")
 *
 * @param contentType string
 * @return *PendingRequest
 */
func (pr *PendingRequest) Accept(contentType string) *PendingRequest {
	pr.headers["Accept"] = contentType
	return pr
}

/**
 * AcceptJSON é um atalho para Accept("application/json").
 *
 * Exemplo:
 *  req.AcceptJSON()
 *
 * @return *PendingRequest
 */
func (pr *PendingRequest) AcceptJSON() *PendingRequest {
	return pr.Accept("application/json")
}

/**
 * ContentType define o cabeçalho Content-Type do pedido.
 *
 * Exemplo:
 *  req.ContentType("application/x-www-form-urlencoded")
 *
 * @param ct string
 * @return *PendingRequest
 */
func (pr *PendingRequest) ContentType(ct string) *PendingRequest {
	pr.headers["Content-Type"] = ct
	return pr
}

/**
 * WithQuery adiciona um único parâmetro à query string da URL.
 *
 * Exemplo:
 *  req.WithQuery("page", "2")
 *
 * @param key string
 * @param value string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithQuery(key, value string) *PendingRequest {
	pr.query.Set(key, value)
	return pr
}

/**
 * WithQueryParams adiciona múltiplos parâmetros de query string via map.
 *
 * Exemplo:
 *  req.WithQueryParams(map[string]string{"page": "1", "limit": "20"})
 *
 * @param params map[string]string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithQueryParams(params map[string]string) *PendingRequest {
	for k, v := range params {
		pr.query.Set(k, v)
	}
	return pr
}

/**
 * WithCookie adiciona um cookie ao pedido.
 *
 * Exemplo:
 *  req.WithCookie("session_id", "abc123xyz")
 *
 * @param name string
 * @param value string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithCookie(name, value string) *PendingRequest {
	pr.cookies = append(pr.cookies, &http.Cookie{Name: name, Value: value})
	return pr
}

/**
 * WithCookies adiciona múltiplos cookies ao pedido via map.
 *
 * Exemplo:
 *  req.WithCookies(map[string]string{"theme": "dark", "lang": "pt"})
 *
 * @param cookies map[string]string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithCookies(cookies map[string]string) *PendingRequest {
	for name, value := range cookies {
		pr.cookies = append(pr.cookies, &http.Cookie{Name: name, Value: value})
	}
	return pr
}

/**
 * Retry configura a quantidade de tentativas e o intervalo de espera entre elas em caso de falha de rede ou erros 5xx.
 * O pedido será tentado até retries+1 vezes no total.
 *
 * Exemplo:
 *  resp, err := http.NewRequest().
 *      Retry(3, 500*time.Millisecond).
 *      Get("https://api.instavel.com/dados")
 *
 * @param retries int
 * @param delay time.Duration
 * @return *PendingRequest
 */
func (pr *PendingRequest) Retry(retries int, delay time.Duration) *PendingRequest {
	pr.retries = retries
	pr.retryMs = delay
	return pr
}

/**
 * WithoutRedirects desativa o seguimento automático de redirecionamentos (3xx).
 *
 * Exemplo:
 *  req.WithoutRedirects()
 *
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithoutRedirects() *PendingRequest {
	pr.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return pr
}

// ----------------------------------------------------------------------
// Métodos de envio (verbos HTTP)
// ----------------------------------------------------------------------

/**
 * Get envia um pedido HTTP GET para a URL indicada.
 *
 * Exemplo:
 *  resp, err := req.Get("/users")
 *
 * @param url string
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Get(url string) (*ClientResponse, error) {
	return pr.send(http.MethodGet, url, nil)
}

/**
 * Post envia um pedido HTTP POST com corpo JSON.
 * O parâmetro data é serializado para JSON automaticamente. Passe nil se o pedido não tiver corpo.
 *
 * Exemplo:
 *  resp, err := req.Post("/users", map[string]string{"name": "Ana"})
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Post(url string, data interface{}) (*ClientResponse, error) {
	return pr.sendJSON(http.MethodPost, url, data)
}

/**
 * Put envia um pedido HTTP PUT com corpo JSON.
 *
 * Exemplo:
 *  resp, err := req.Put("/users/1", userStruct)
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Put(url string, data interface{}) (*ClientResponse, error) {
	return pr.sendJSON(http.MethodPut, url, data)
}

/**
 * Patch envia um pedido HTTP PATCH com corpo JSON.
 *
 * Exemplo:
 *  resp, err := req.Patch("/users/1", map[string]interface{}{"status": "active"})
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Patch(url string, data interface{}) (*ClientResponse, error) {
	return pr.sendJSON(http.MethodPatch, url, data)
}

/**
 * Delete envia um pedido HTTP DELETE. O parâmetro data é opcional (pode ser omitido ou nil).
 *
 * Exemplo:
 *  resp, err := req.Delete("/users/1")
 *
 * @param url string
 * @param data ...interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Delete(url string, data ...interface{}) (*ClientResponse, error) {
	var body interface{}
	if len(data) > 0 {
		body = data[0]
	}
	if body != nil {
		return pr.sendJSON(http.MethodDelete, url, body)
	}
	return pr.send(http.MethodDelete, url, nil)
}

/**
 * Head envia um pedido HTTP HEAD para a URL indicada.
 *
 * Exemplo:
 *  resp, err := req.Head("/files/download.zip")
 *
 * @param url string
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Head(url string) (*ClientResponse, error) {
	return pr.send(http.MethodHead, url, nil)
}

/**
 * Options envia um pedido HTTP OPTIONS para a URL indicada.
 *
 * Exemplo:
 *  resp, err := req.Options("/api/v1")
 *
 * @param url string
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Options(url string) (*ClientResponse, error) {
	return pr.send(http.MethodOptions, url, nil)
}

/**
 * PostForm envia um pedido HTTP POST codificado como application/x-www-form-urlencoded.
 *
 * Exemplo:
 *  resp, _ := http.NewRequest().PostForm("https://api.com/login", map[string]string{
 *      "email":    "ana@ex.com",
 *      "password": "123456",
 *  })
 *
 * @param reqURL string
 * @param data map[string]string
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) PostForm(reqURL string, data map[string]string) (*ClientResponse, error) {
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}
	body := strings.NewReader(form.Encode())
	pr.headers["Content-Type"] = "application/x-www-form-urlencoded"
	return pr.send(http.MethodPost, reqURL, body)
}

/**
 * PostMultipart envia um pedido HTTP POST multipart/form-data, ideal para upload de arquivos combinados com campos de texto.
 *
 * Exemplo:
 *  resp, _ := http.NewRequest().PostMultipart("https://api.com/upload",
 *      map[string]string{"descricao": "foto de perfil"},
 *      map[string][]byte{"avatar": avatarBytes},
 *  )
 *
 * @param reqURL string
 * @param fields map[string]string
 * @param files map[string][]byte
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) PostMultipart(reqURL string, fields map[string]string, files map[string][]byte) (*ClientResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			return nil, fmt.Errorf("http: erro ao escrever campo %q: %w", key, err)
		}
	}

	for name, data := range files {
		part, err := writer.CreateFormFile(name, name)
		if err != nil {
			return nil, fmt.Errorf("http: erro ao criar parte multipart %q: %w", name, err)
		}
		if _, err := part.Write(data); err != nil {
			return nil, fmt.Errorf("http: erro ao escrever ficheiro %q: %w", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("http: erro ao fechar writer multipart: %w", err)
	}

	pr.headers["Content-Type"] = writer.FormDataContentType()
	return pr.send(http.MethodPost, reqURL, &buf)
}

// ----------------------------------------------------------------------
// Envio interno
// ----------------------------------------------------------------------

/**
 * sendJSON auxilia a serialização do corpo em JSON e ajusta o Content-Type antes de chamar o método send.
 *
 * @param method string
 * @param reqURL string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) sendJSON(method, reqURL string, data interface{}) (*ClientResponse, error) {
	var body io.Reader
	if data != nil {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("http: erro ao serializar JSON: %w", err)
		}
		body = bytes.NewReader(jsonBytes)
		if _, ok := pr.headers["Content-Type"]; !ok {
			pr.headers["Content-Type"] = "application/json; charset=utf-8"
		}
	}
	return pr.send(method, reqURL, body)
}

/**
 * resolveURL monta a URL final unindo a baseURL (se definida) e anexando os parâmetros de query string.
 *
 * @param rawURL string
 * @return string
 */
func (pr *PendingRequest) resolveURL(rawURL string) string {
	if pr.baseURL != "" && !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = pr.baseURL + "/" + strings.TrimLeft(rawURL, "/")
	}
	if len(pr.query) > 0 {
		sep := "?"
		if strings.Contains(rawURL, "?") {
			sep = "&"
		}
		rawURL += sep + pr.query.Encode()
	}
	return rawURL
}

/**
 * send executa o pedido HTTP nativo lidando com tentativas de retry, envio de headers, cookies e leitura da resposta.
 *
 * @param method string
 * @param rawURL string
 * @param body io.Reader
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) send(method, rawURL string, body io.Reader) (*ClientResponse, error) {
	fullURL := pr.resolveURL(rawURL)

	var lastErr error
	maxAttempts := 1 + pr.retries

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 && pr.retryMs > 0 {
			time.Sleep(pr.retryMs)
		}

		req, err := http.NewRequest(method, fullURL, body)
		if err != nil {
			return nil, fmt.Errorf("http: erro ao criar pedido: %w", err)
		}

		for k, v := range pr.headers {
			req.Header.Set(k, v)
		}
		for _, c := range pr.cookies {
			req.AddCookie(c)
		}

		resp, err := pr.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		cr := &ClientResponse{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			body:       respBody,
			cookies:    resp.Cookies(),
		}

		// Só faz retry em server errors (5xx)
		if resp.StatusCode >= 500 && attempt < maxAttempts-1 {
			lastErr = fmt.Errorf("http: status %d", resp.StatusCode)
			continue
		}

		return cr, nil
	}

	return nil, fmt.Errorf("http: todas as %d tentativas falharam: %w", maxAttempts, lastErr)
}

// ----------------------------------------------------------------------
// ClientResponse: resposta recebida de uma API externa
// ----------------------------------------------------------------------

/**
 * ClientResponse encapsula a resposta de um pedido HTTP feito pelo
 * cliente, com helpers para ler o corpo como JSON, texto, verificar
 * status, etc. — equivalente ao Response do Http:: do Laravel.
 */
type ClientResponse struct {
	statusCode int
	headers    http.Header
	body       []byte
	cookies    []*http.Cookie
}

/**
 * Status devolve o código de estado HTTP da resposta (ex: 200, 404, 500).
 *
 * @return int
 */
func (r *ClientResponse) Status() int {
	return r.statusCode
}

/**
 * Ok indica se o status está na faixa de sucesso (200-299).
 *
 * @return bool
 */
func (r *ClientResponse) Ok() bool {
	return r.statusCode >= 200 && r.statusCode < 300
}

/**
 * Successful é um alias para Ok(). Indica se a requisição foi bem-sucedida (2xx).
 *
 * @return bool
 */
func (r *ClientResponse) Successful() bool {
	return r.Ok()
}

/**
 * Failed indica se a resposta possui um código de erro (status >= 400).
 *
 * @return bool
 */
func (r *ClientResponse) Failed() bool {
	return r.statusCode >= 400
}

/**
 * ServerError indica se ocorreu um erro interno no servidor remoto (status 5xx).
 *
 * @return bool
 */
func (r *ClientResponse) ServerError() bool {
	return r.statusCode >= 500
}

/**
 * ClientError indica se o erro foi provocado por um parâmetro/requisição do cliente (status 4xx).
 *
 * @return bool
 */
func (r *ClientResponse) ClientError() bool {
	return r.statusCode >= 400 && r.statusCode < 500
}

/**
 * Redirect indica se a resposta é um redirecionamento (status 3xx).
 *
 * @return bool
 */
func (r *ClientResponse) Redirect() bool {
	return r.statusCode >= 300 && r.statusCode < 400
}

/**
 * Unauthorized indica se o pedido não foi autorizado (status 401).
 *
 * @return bool
 */
func (r *ClientResponse) Unauthorized() bool {
	return r.statusCode == 401
}

/**
 * Forbidden indica se o acesso ao recurso foi proibido (status 403).
 *
 * @return bool
 */
func (r *ClientResponse) Forbidden() bool {
	return r.statusCode == 403
}

/**
 * NotFound indica se o recurso requisitado não foi encontrado (status 404).
 *
 * @return bool
 */
func (r *ClientResponse) NotFound() bool {
	return r.statusCode == 404
}

/**
 * Body devolve o corpo bruto da resposta convertido para string.
 *
 * @return string
 */
func (r *ClientResponse) Body() string {
	return string(r.body)
}

/**
 * Bytes devolve a fatia de bytes ([]byte) bruta do corpo da resposta.
 *
 * @return []byte
 */
func (r *ClientResponse) Bytes() []byte {
	return r.body
}

/**
 * Json descodifica o corpo JSON da resposta para a estrutura de destino fornecida (ponteiro).
 *
 * Exemplo:
 *  var user User
 *  resp.Json(&user)
 *
 * @param dest interface{}
 * @return error
 */
func (r *ClientResponse) Json(dest interface{}) error {
	if len(r.body) == 0 {
		return fmt.Errorf("http: corpo vazio, não é possível descodificar JSON")
	}
	return json.Unmarshal(r.body, dest)
}

/**
 * Map descodifica o corpo JSON num map[string]interface{}, útil para respostas sem struct definida.
 *
 * Exemplo:
 *  dados, _ := resp.Map()
 *  fmt.Println(dados["nome"])
 *
 * @return (map[string]interface{}, error)
 */
func (r *ClientResponse) Map() (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := r.Json(&m); err != nil {
		return nil, err
	}
	return m, nil
}

/**
 * Collect descodifica o corpo JSON numa lista/slice de maps, útil quando a API devolve um array de objetos JSON.
 *
 * Exemplo:
 *  items, _ := resp.Collect()
 *  for _, item := range items {
 *      fmt.Println(item["nome"])
 *  }
 *
 * @return ([]map[string]interface{}, error)
 */
func (r *ClientResponse) Collect() ([]map[string]interface{}, error) {
	var items []map[string]interface{}
	if err := r.Json(&items); err != nil {
		return nil, err
	}
	return items, nil
}

/**
 * Header devolve o primeiro valor associado ao cabeçalho informado da resposta.
 *
 * @param name string
 * @return string
 */
func (r *ClientResponse) Header(name string) string {
	return r.headers.Get(name)
}

/**
 * Headers devolve todos os cabeçalhos recebidos na resposta HTTP.
 *
 * @return http.Header
 */
func (r *ClientResponse) Headers() http.Header {
	return r.headers
}

/**
 * Cookies devolve todos os cookies definidos pelo servidor através da resposta.
 *
 * @return []*http.Cookie
 */
func (r *ClientResponse) Cookies() []*http.Cookie {
	return r.cookies
}

/**
 * Cookie devolve o valor de um cookie específico pelo nome, ou uma string vazia caso não seja encontrado.
 *
 * @param name string
 * @return string
 */
func (r *ClientResponse) Cookie(name string) string {
	for _, c := range r.cookies {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// ----------------------------------------------------------------------
// Atalhos globais (funções de pacote)
// ----------------------------------------------------------------------

/**
 * Get é um atalho global para NewRequest().Get(url).
 *
 * Exemplo:
 *  resp, err := http.Get("https://api.exemplo.com/users")
 *
 * @param url string
 * @return (*ClientResponse, error)
 */
func Get(url string) (*ClientResponse, error) {
	return NewRequest().Get(url)
}

/**
 * Post é um atalho global para NewRequest().Post(url, data).
 *
 * Exemplo:
 *  resp, err := http.Post("https://api.exemplo.com/users", payload)
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func Post(url string, data interface{}) (*ClientResponse, error) {
	return NewRequest().Post(url, data)
}

/**
 * Put é um atalho global para NewRequest().Put(url, data).
 *
 * Exemplo:
 *  resp, err := http.Put("https://api.exemplo.com/users/1", payload)
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func Put(url string, data interface{}) (*ClientResponse, error) {
	return NewRequest().Put(url, data)
}

/**
 * Patch é um atalho global para NewRequest().Patch(url, data).
 *
 * Exemplo:
 *  resp, err := http.Patch("https://api.exemplo.com/users/1", payload)
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func Patch(url string, data interface{}) (*ClientResponse, error) {
	return NewRequest().Patch(url, data)
}

/**
 * Delete é um atalho global para NewRequest().Delete(url, data...).
 *
 * Exemplo:
 *  resp, err := http.Delete("https://api.exemplo.com/users/1")
 *
 * @param url string
 * @param data ...interface{}
 * @return (*ClientResponse, error)
 */
func Delete(url string, data ...interface{}) (*ClientResponse, error) {
	return NewRequest().Delete(url, data...)
}

// ----------------------------------------------------------------------
// Helpers internos
// ----------------------------------------------------------------------

/**
 * basicAuth codifica credenciais user:password em Base64 para ser utilizado no cabeçalho HTTP Basic Auth.
 *
 * @param user string
 * @param password string
 * @return string
 */
func basicAuth(user, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
}