package kite

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
)

// Request encapsula o *http.Request original e adiciona parâmetros de rota
// (ex: /users/:id), utilitários de leitura de body/query/headers/cookies.
//
// Campos:
//   - Params: parâmetros de rota extraídos pelo Router (ex: {"id": "42"})
type Request struct {
	*http.Request
	Params map[string]string

	// bodyBytes guarda o body já lido, permitindo chamar BindJson/Body
	// mais de uma vez sem "esvaziar" o io.ReadCloser original.
	bodyBytes []byte
	bodyRead  bool
}

// Param retorna o valor de um parâmetro de rota.
//
// Exemplo:
//
//	// rota: /users/:id
//	req.Param("id") // -> "42"
func (r Request) Param(name string) string {
	return r.Params[name]
}

// Query retorna o valor de um único query param (?name=valor).
//
// Exemplo:
//
//	// GET /search?q=golang
//	req.Query("q") // -> "golang"
func (r Request) Query(name string) string {
	return r.URL.Query().Get(name)
}

// Queries retorna todos os query params da URL, incluindo os que
// aparecem mais de uma vez (?tag=a&tag=b).
//
// Retorno: map[string][]string, ex: {"tag": ["a", "b"]}
func (r Request) Queries() map[string][]string {
	return r.URL.Query()
}

// Header retorna o valor de um header da requisição (case-insensitive,
// igual o comportamento nativo do net/http).
//
// Exemplo:
//
//	req.Header("Authorization") // -> "Bearer xyz"
func (r Request) Header(name string) string {
	return r.Request.Header.Get(name)
}

// Cookie retorna o valor de um cookie pelo nome. O segundo valor de
// retorno indica se o cookie foi encontrado.
//
// Exemplo:
//
//	if session, ok := req.Cookie("session_id"); ok {
//	    // ...
//	}
func (r Request) Cookie(name string) (string, bool) {
	c, err := r.Request.Cookie(name)
	if err != nil {
		return "", false
	}
	return c.Value, true
}

// IP retorna o endereço IP do cliente, priorizando o header
// X-Forwarded-For (comum atrás de proxy/load balancer) e caindo para
// RemoteAddr quando o header não existir.
func (r Request) IP() string {
	if fwd := r.Header("X-Forwarded-For"); fwd != "" {
		// X-Forwarded-For pode vir como "cliente, proxy1, proxy2"
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr
}

// ContentType retorna o Content-Type da requisição, sem os parâmetros
// extras (ex: "application/json; charset=utf-8" -> "application/json").
func (r Request) ContentType() string {
	ct := r.Header("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = ct[:idx]
	}
	return strings.TrimSpace(ct)
}

// Is verifica se o Content-Type da requisição bate com o informado.
//
// Exemplo:
//
//	if req.Is("application/json") { ... }
func (r Request) Is(contentType string) bool {
	return r.ContentType() == contentType
}

// Body lê e retorna o corpo bruto da requisição como []byte. Pode ser
// chamado múltiplas vezes (o resultado fica em cache no próprio Request),
// diferente de usar r.Request.Body diretamente, que só pode ser lido uma vez.
func (r *Request) Body() ([]byte, error) {
	if r.bodyRead {
		return r.bodyBytes, nil
	}
	defer r.Request.Body.Close()

	data, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return nil, err
	}
	r.bodyBytes = data
	r.bodyRead = true
	return data, nil
}

// BindJson decodifica o body JSON da requisição no destino informado.
// Usa Body() internamente, então pode ser combinado com outras leituras
// do corpo sem conflito.
//
// Exemplo:
//
//	var payload struct{ Name string `json:"name"` }
//	if err := req.BindJson(&payload); err != nil {
//	    return res.WithStatus(400).WithJson(map[string]string{"error": "json inválido"})
//	}
func (r *Request) BindJson(dest interface{}) error {
	data, err := r.Body()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Ctx retorna o context.Context da requisição, útil para propagar
// cancelamento/timeout e valores para camadas internas (ex: DB queries).
func (r Request) Ctx() context.Context {
	return r.Request.Context()
}

// Response encapsula o http.ResponseWriter e expõe métodos encadeáveis
// que retornam error, compatíveis com a assinatura do HandlerFunc.
//
// Nota: Response é passado por valor entre os métodos With*/Status/etc,
// mas escreve sempre no mesmo http.ResponseWriter subjacente (interface),
// então o encadeamento funciona normalmente.
type Response struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponse(w http.ResponseWriter) Response {
	return Response{ResponseWriter: w, statusCode: http.StatusOK}
}

// writeHeaderOnce garante que WriteHeader seja chamado no máximo uma vez
// por resposta, evitando o warning "superfluous response.WriteHeader call"
// e respostas corrompidas quando dois métodos With*/Send tentam escrever
// o header separadamente.
func (res *Response) writeHeaderOnce() {
	if !res.written {
		res.WriteHeader(res.statusCode)
		res.written = true
	}
}

// WithStatus define o status code que será usado na próxima escrita.
// Mantido por compatibilidade; prefira Status para código novo.
func (res Response) WithStatus(code int) Response {
	res.statusCode = code
	return res
}

// Status define o status code que será usado na próxima escrita,
// no estilo do res.status() do Express. Encadeável.
//
// Exemplo:
//
//	return res.Status(201).Json(payload)
func (res Response) Status(code int) Response {
	res.statusCode = code
	return res
}

// SetHeader define um header customizado na resposta. Encadeável.
// Precisa ser chamado antes de qualquer With*/Send/Json, já que o Go
// não permite alterar headers depois do WriteHeader.
//
// Exemplo:
//
//	return res.SetHeader("X-Request-Id", reqID).Json(payload)
func (res Response) SetHeader(key, value string) Response {
	res.Header().Set(key, value)
	return res
}

// Cookie adiciona um Set-Cookie na resposta. Encadeável.
func (res Response) Cookie(cookie *http.Cookie) Response {
	http.SetCookie(res.ResponseWriter, cookie)
	return res
}

// WithJson serializa v para JSON e escreve na resposta.
func (res Response) WithJson(v interface{}) error {
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.writeHeaderOnce()
	return json.NewEncoder(res).Encode(v)
}

// Json é um alias de WithJson no estilo res.json() do Express.
func (res Response) Json(v interface{}) error {
	return res.WithJson(v)
}

// WithText escreve uma resposta em texto puro.
func (res Response) WithText(text string) error {
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.writeHeaderOnce()
	_, err := res.Write([]byte(text))
	return err
}

// WithHtml escreve uma resposta HTML já renderizada (string).
func (res Response) WithHtml(html string) error {
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.writeHeaderOnce()
	_, err := res.Write([]byte(html))
	return err
}

// Send envia a resposta inferindo o Content-Type a partir do tipo de v,
// espelhando o comportamento do res.send() do Express.
//
//   - string      -> texto puro
//   - []byte      -> binário (application/octet-stream)
//   - nil         -> apenas escreve o status, sem body
//   - qualquer outro (struct, map, slice) -> JSON
func (res Response) Send(v interface{}) error {
	switch val := v.(type) {
	case string:
		return res.WithText(val)
	case []byte:
		res.Header().Set("Content-Type", "application/octet-stream")
		res.writeHeaderOnce()
		_, err := res.Write(val)
		return err
	case nil:
		res.writeHeaderOnce()
		return nil
	default:
		return res.WithJson(val)
	}
}

// SendStatus define o status e envia o texto padrão dele como body
// (ex: res.SendStatus(404) -> "Not Found"), igual res.sendStatus() do Express.
func (res Response) SendStatus(code int) error {
	res.statusCode = code
	return res.WithText(http.StatusText(code))
}

// NoContent responde com 204 e nenhum body. Útil pra DELETE/PUT que não
// retornam payload.
func (res Response) NoContent() error {
	res.statusCode = http.StatusNoContent
	res.writeHeaderOnce()
	return nil
}

// Redirect envia um redirect HTTP para a URL informada. O código é
// opcional e usa 302 (Found) como padrão, igual o Express.
//
// Exemplo:
//
//	return res.Redirect("/login")
//	return res.Redirect("/login", http.StatusMovedPermanently)
func (res Response) Redirect(url string, code ...int) error {
	status := http.StatusFound
	if len(code) > 0 {
		status = code[0]
	}
	res.Header().Set("Location", url)
	res.statusCode = status
	res.writeHeaderOnce()
	return nil
}

// File serve o conteúdo de um arquivo do disco como resposta (imagens,
// PDFs, downloads, etc). Não passa pelo mecanismo de status/writeHeaderOnce
// porque http.ServeFile já cuida disso internamente.
func (res Response) File(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	http.ServeFile(res.ResponseWriter, &http.Request{}, path)
	return nil
}