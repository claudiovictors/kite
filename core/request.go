package kite

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

/**
 * Request encapsula o *http.Request nativo do Go e disponibiliza utilitários
 * para leitura de parâmetros de rota, query string, cabeçalhos, cookies e corpo da requisição.
 */
type Request struct {
	*http.Request
	Params map[string]string

	bodyBytes []byte
	bodyRead  bool
}

/* ---------------------------------------------------------------------- */
/* Parâmetros de rota                                                     */
/* ---------------------------------------------------------------------- */

/**
 * Param retorna o valor de um parâmetro de rota pelo nome.
 * Retorna uma string vazia caso o parâmetro não exista.
 *
 * Exemplo:
 *  req.Param("id") // -> "42"
 *
 * @param name string
 * @return string
 */
func (r Request) Param(name string) string {
	return r.Params[name]
}

/**
 * HasParam verifica se um determinado parâmetro de rota está presente na requisição.
 *
 * @param name string
 * @return bool
 */
func (r Request) HasParam(name string) bool {
	_, ok := r.Params[name]
	return ok
}

/**
 * ParamInt lê um parâmetro de rota e o converte para o tipo inteiro.
 * Retorna o valor padrão (def) se o parâmetro for inexistente ou inválido.
 *
 * Exemplo:
 *  id := req.ParamInt("id", 0)
 *
 * @param name string
 * @param def int
 * @return int
 */
func (r Request) ParamInt(name string, def int) int {
	val, err := strconv.Atoi(r.Params[name])
	if err != nil {
		return def
	}
	return val
}

/**
 * ParamInt64 lê um parâmetro de rota e o converte para o tipo int64.
 * Retorna o valor padrão (def) se o parâmetro for inexistente ou inválido.
 *
 * @param name string
 * @param def int64
 * @return int64
 */
func (r Request) ParamInt64(name string, def int64) int64 {
	val, err := strconv.ParseInt(r.Params[name], 10, 64)
	if err != nil {
		return def
	}
	return val
}

/* ---------------------------------------------------------------------- */
/* Query string                                                           */
/* ---------------------------------------------------------------------- */

/**
 * Query recupera o valor de um parâmetro de query string (?nome=valor).
 * Retorna apenas o primeiro valor em caso de múltiplas ocorrências.
 *
 * Exemplo:
 *  req.Query("q") // -> "golang"
 *
 * @param name string
 * @return string
 */
func (r Request) Query(name string) string {
	return r.URL.Query().Get(name)
}

/**
 * QueryDefault recupera um parâmetro da query string, retornando o valor padrão (def) caso esteja ausente ou vazio.
 *
 * @param name string
 * @param def string
 * @return string
 */
func (r Request) QueryDefault(name, def string) string {
	val := r.Query(name)
	if val == "" {
		return def
	}
	return val
}

/**
 * QueryInt recupera um parâmetro da query string e o converte para inteiro.
 *
 * Exemplo:
 *  page := req.QueryInt("page", 1)
 *
 * @param name string
 * @param def int
 * @return int
 */
func (r Request) QueryInt(name string, def int) int {
	val, err := strconv.Atoi(r.Query(name))
	if err != nil {
		return def
	}
	return val
}

/**
 * QueryInt64 recupera um parâmetro da query string e o converte para int64.
 *
 * @param name string
 * @param def int64
 * @return int64
 */
func (r Request) QueryInt64(name string, def int64) int64 {
	val, err := strconv.ParseInt(r.Query(name), 10, 64)
	if err != nil {
		return def
	}
	return val
}

/**
 * QueryFloat recupera um parâmetro da query string e o converte para float64.
 *
 * @param name string
 * @param def float64
 * @return float64
 */
func (r Request) QueryFloat(name string, def float64) float64 {
	val, err := strconv.ParseFloat(r.Query(name), 64)
	if err != nil {
		return def
	}
	return val
}

/**
 * QueryBool recupera um parâmetro da query string e o interpreta como booleano.
 * Aceita as representações "1", "true", "t", "yes", "y" como verdadeiro.
 *
 * Exemplo:
 *  active := req.QueryBool("active", false)
 *
 * @param name string
 * @param def bool
 * @return bool
 */
func (r Request) QueryBool(name string, def bool) bool {
	val := strings.ToLower(strings.TrimSpace(r.Query(name)))
	if val == "" {
		return def
	}
	switch val {
	case "1", "true", "t", "yes", "y":
		return true
	case "0", "false", "f", "no", "n":
		return false
	default:
		return def
	}
}

/**
 * Queries retorna todos os parâmetros da query string, incluindo chaves duplicadas.
 *
 * @return map[string][]string
 */
func (r Request) Queries() map[string][]string {
	return r.URL.Query()
}

/**
 * HasQuery verifica se um determinado parâmetro existe na query string da requisição.
 *
 * @param name string
 * @return bool
 */
func (r Request) HasQuery(name string) bool {
	return r.URL.Query().Has(name)
}

/* ---------------------------------------------------------------------- */
/* Cabeçalhos e metadados do pedido                                       */
/* ---------------------------------------------------------------------- */

/**
 * Header retorna o valor de um cabeçalho HTTP da requisição (case-insensitive).
 *
 * Exemplo:
 *  req.Header("Authorization") // -> "Bearer xyz"
 *
 * @param name string
 * @return string
 */
func (r Request) Header(name string) string {
	return r.Request.Header.Get(name)
}

/**
 * HeaderDefault retorna o valor de um cabeçalho HTTP ou o valor padrão (def) se o cabeçalho estiver ausente.
 *
 * @param name string
 * @param def string
 * @return string
 */
func (r Request) HeaderDefault(name, def string) string {
	val := r.Header(name)
	if val == "" {
		return def
	}
	return val
}

/**
 * Cookie recupera o valor de um cookie da requisição pelo seu nome.
 * Retorna o valor em string e um indicador booleano de presença.
 *
 * Exemplo:
 *  if sessao, ok := req.Cookie("session_id"); ok { ... }
 *
 * @param name string
 * @return string
 * @return bool
 */
func (r Request) Cookie(name string) (string, bool) {
	c, err := r.Request.Cookie(name)
	if err != nil {
		return "", false
	}
	return c.Value, true
}

/**
 * IP retorna o endereço IP de origem do cliente, priorizando o cabeçalho X-Forwarded-For em cenários de proxy.
 *
 * @return string
 */
func (r Request) IP() string {
	if fwd := r.Header("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr
}

/**
 * ContentType retorna o tipo de conteúdo MIME da requisição desconsiderando parâmetros extras.
 *
 * @return string
 */
func (r Request) ContentType() string {
	ct := r.Header("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = ct[:idx]
	}
	return strings.TrimSpace(ct)
}

/**
 * Is verifica se o Content-Type da requisição coincide com o tipo informado.
 *
 * Exemplo:
 *  if req.Is("application/json") { ... }
 *
 * @param contentType string
 * @return bool
 */
func (r Request) Is(contentType string) bool {
	return r.ContentType() == contentType
}

/**
 * Accepts verifica se o cabeçalho Accept da requisição é compatível com o tipo de conteúdo fornecido.
 *
 * Exemplo:
 *  if req.Accepts("application/json") { ... }
 *
 * @param contentType string
 * @return bool
 */
func (r Request) Accepts(contentType string) bool {
	accept := r.Header("Accept")
	return strings.Contains(accept, contentType) || strings.Contains(accept, "*/*")
}

/**
 * AcceptsJSON é um método utilitário para verificar se a requisição aceita respostas no formato JSON.
 *
 * @return bool
 */
func (r Request) AcceptsJSON() bool {
	return r.Accepts("application/json")
}

/**
 * AcceptsHTML é um método utilitário para verificar se a requisição aceita respostas no formato HTML.
 *
 * @return bool
 */
func (r Request) AcceptsHTML() bool {
	return r.Accepts("text/html")
}

/**
 * XHR verifica se a requisição foi feita via XMLHttpRequest (cabeçalho X-Requested-With).
 *
 * @return bool
 */
func (r Request) XHR() bool {
	return r.Header("X-Requested-With") == "XMLHttpRequest"
}

/**
 * Host retorna o domínio/host da requisição original.
 *
 * @return string
 */
func (r Request) Host() string {
	return r.Request.Host
}

/**
 * Path retorna o caminho (path) da URL da requisição sem os parâmetros de query string.
 *
 * @return string
 */
func (r Request) Path() string {
	return r.URL.Path
}

/**
 * Scheme determina o protocolo (http ou https) utilizado na requisição.
 *
 * @return string
 */
func (r Request) Scheme() string {
	if proto := r.Header("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

/**
 * FullURL reconstrói e retorna a URL absoluta completa da requisição atual.
 *
 * @return string
 */
func (r Request) FullURL() string {
	return r.Scheme() + "://" + r.Host() + r.URL.RequestURI()
}

/**
 * UserAgent retorna a string do cabeçalho User-Agent enviado pelo cliente.
 *
 * @return string
 */
func (r Request) UserAgent() string {
	return r.Header("User-Agent")
}

/**
 * Referer retorna o valor do cabeçalho HTTP Referer da requisição.
 *
 * @return string
 */
func (r Request) Referer() string {
	return r.Header("Referer")
}

/* ---------------------------------------------------------------------- */
/* Corpo do pedido                                                        */
/* ---------------------------------------------------------------------- */

/**
 * Body realiza a leitura e armazenamento em cache dos bytes do corpo da requisição.
 * Permite múltiplas leituras consecutivas do payload sem esgotar a stream.
 *
 * @return []byte
 * @return error
 */
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

/**
 * BindJson faz a decodificação do corpo JSON da requisição para a estrutura de destino fornecida.
 *
 * Exemplo:
 *  var payload StructUser
 *  if err := req.BindJson(&payload); err != nil { ... }
 *
 * @param dest interface{}
 * @return error
 */
func (r *Request) BindJson(dest interface{}) error {
	data, err := r.Body()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

const maxMultipartMemory = 32 << 20 // 32 MB

/**
 * FormValue realiza o parse de dados de formulário e retorna o valor da chave informada.
 *
 * @param name string
 * @return string
 */
func (r *Request) FormValue(name string) string {
	_ = r.Request.ParseMultipartForm(maxMultipartMemory)
	return r.Request.FormValue(name)
}

/**
 * FormValues retorna todos os valores presentes nos dados do formulário submetido.
 *
 * @return url.Values
 * @return error
 */
func (r *Request) FormValues() (url.Values, error) {
	if err := r.Request.ParseMultipartForm(maxMultipartMemory); err != nil && err != http.ErrNotMultipart {
		return nil, err
	}
	return r.Request.Form, nil
}

/**
 * FormFile recupera o arquivo enviado no campo multipart correspondente.
 *
 * @param name string
 * @return multipart.File
 * @return *multipart.FileHeader
 * @return error
 */
func (r *Request) FormFile(name string) (multipart.File, *multipart.FileHeader, error) {
	if err := r.Request.ParseMultipartForm(maxMultipartMemory); err != nil {
		return nil, nil, err
	}
	return r.Request.FormFile(name)
}

/**
 * SaveUploadedFile lê o arquivo enviado via formulário multipart e faz o salvamento direto no caminho informado no disco.
 *
 * Exemplo:
 *  err := req.SaveUploadedFile("avatar", "./uploads/avatar.png")
 *
 * @param fieldName string
 * @param destino string
 * @return error
 */
func (r *Request) SaveUploadedFile(fieldName, destino string) error {
	src, _, err := r.FormFile(fieldName)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destino)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

/* ---------------------------------------------------------------------- */
/* Contexto                                                               */
/* ---------------------------------------------------------------------- */

/**
 * Ctx retorna o context.Context associado à requisição HTTP.
 *
 * @return context.Context
 */
func (r Request) Ctx() context.Context {
	return r.Request.Context()
}

/**
 * WithContext retorna uma nova instância de Request atualizada com o context.Context fornecido.
 *
 * Exemplo:
 *  ctx := context.WithValue(req.Ctx(), userCtxKey, user)
 *  req = req.WithContext(ctx)
 *
 * @param ctx context.Context
 * @return Request
 */
func (r Request) WithContext(ctx context.Context) Request {
	r.Request = r.Request.WithContext(ctx)
	return r
}