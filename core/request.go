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

// Request encapsula o *http.Request nativo do Go e acrescenta-lhe
// utilitários para leitura de parâmetros de rota, query string,
// cabeçalhos, cookies, corpo do pedido (JSON, formulários e ficheiros
// enviados por multipart).
//
// Como o *http.Request está embutido (embedded), qualquer campo ou
// método nativo continua acessível diretamente — por exemplo
// req.Method, req.URL ou req.Context() funcionam sem qualquer wrapper.
//
// Campos:
//   - Params: parâmetros de rota extraídos pelo Router a partir do
//     padrão da rota (ex.: numa rota "/users/:id", Params["id"] terá
//     o valor do segmento correspondente do pedido recebido).
type Request struct {
	*http.Request
	Params map[string]string

	// bodyBytes guarda o corpo do pedido já lido, permitindo chamar
	// Body() ou BindJson() mais do que uma vez sem "esvaziar" o
	// io.ReadCloser original (que só pode ser lido uma única vez).
	bodyBytes []byte
	bodyRead  bool
}

// ----------------------------------------------------------------------
// Parâmetros de rota
// ----------------------------------------------------------------------

// Param devolve o valor de um parâmetro de rota pelo nome. Se o
// parâmetro não existir, devolve uma string vazia.
//
// Exemplo:
//
//	// rota registada como "/users/:id"
//	req.Param("id") // -> "42"
func (r Request) Param(name string) string {
	return r.Params[name]
}

// HasParam indica se um parâmetro de rota com o nome dado está
// presente (útil para distinguir "não veio" de "veio vazio", caso a
// rota o permita).
func (r Request) HasParam(name string) bool {
	_, ok := r.Params[name]
	return ok
}

// ParamInt lê um parâmetro de rota e converte-o para int. Caso o
// parâmetro não exista ou não seja um número válido, devolve def.
//
// Exemplo:
//
//	id := req.ParamInt("id", 0)
func (r Request) ParamInt(name string, def int) int {
	val, err := strconv.Atoi(r.Params[name])
	if err != nil {
		return def
	}
	return val
}

// ParamInt64 é equivalente a ParamInt, mas devolve int64 — útil para
// chaves primárias grandes (bigint) ou identificadores externos.
func (r Request) ParamInt64(name string, def int64) int64 {
	val, err := strconv.ParseInt(r.Params[name], 10, 64)
	if err != nil {
		return def
	}
	return val
}

// ----------------------------------------------------------------------
// Query string
// ----------------------------------------------------------------------

// Query devolve o valor de um único parâmetro da query string
// (?nome=valor). Se existirem vários valores com o mesmo nome, devolve
// apenas o primeiro — use Queries() para obter todos.
//
// Exemplo:
//
//	// GET /search?q=golang
//	req.Query("q") // -> "golang"
func (r Request) Query(name string) string {
	return r.URL.Query().Get(name)
}

// QueryDefault é como Query, mas devolve def quando o parâmetro não
// existir ou vier vazio.
func (r Request) QueryDefault(name, def string) string {
	val := r.Query(name)
	if val == "" {
		return def
	}
	return val
}

// QueryInt lê um parâmetro da query string e converte-o para int. Se
// não existir ou não for um número válido, devolve def.
//
// Exemplo:
//
//	// GET /posts?page=2
//	page := req.QueryInt("page", 1)
func (r Request) QueryInt(name string, def int) int {
	val, err := strconv.Atoi(r.Query(name))
	if err != nil {
		return def
	}
	return val
}

// QueryInt64 é equivalente a QueryInt, mas devolve int64.
func (r Request) QueryInt64(name string, def int64) int64 {
	val, err := strconv.ParseInt(r.Query(name), 10, 64)
	if err != nil {
		return def
	}
	return val
}

// QueryFloat lê um parâmetro da query string e converte-o para
// float64. Se não existir ou não for válido, devolve def.
func (r Request) QueryFloat(name string, def float64) float64 {
	val, err := strconv.ParseFloat(r.Query(name), 64)
	if err != nil {
		return def
	}
	return val
}

// QueryBool lê um parâmetro da query string e interpreta-o como
// booleano. Considera verdadeiro "1", "true", "t", "yes", "y" (sem
// distinção de maiúsculas/minúsculas). Se o parâmetro não existir,
// devolve def.
//
// Exemplo:
//
//	// GET /users?active=true
//	active := req.QueryBool("active", false)
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

// Queries devolve todos os parâmetros da query string, incluindo os
// que aparecem repetidos (?tag=a&tag=b).
//
// Retorno: map[string][]string, ex.: {"tag": ["a", "b"]}
func (r Request) Queries() map[string][]string {
	return r.URL.Query()
}

// HasQuery indica se um parâmetro da query string está presente no
// pedido, independentemente do valor.
func (r Request) HasQuery(name string) bool {
	return r.URL.Query().Has(name)
}

// ----------------------------------------------------------------------
// Cabeçalhos e metadados do pedido
// ----------------------------------------------------------------------

// Header devolve o valor de um cabeçalho do pedido. A comparação do
// nome não é sensível a maiúsculas/minúsculas, tal como o
// comportamento nativo do net/http.
//
// Exemplo:
//
//	req.Header("Authorization") // -> "Bearer xyz"
func (r Request) Header(name string) string {
	return r.Request.Header.Get(name)
}

// HeaderDefault é como Header, mas devolve def quando o cabeçalho não
// estiver presente.
func (r Request) HeaderDefault(name, def string) string {
	val := r.Header(name)
	if val == "" {
		return def
	}
	return val
}

// Cookie devolve o valor de um cookie pelo nome. O segundo valor
// devolvido indica se o cookie foi encontrado no pedido.
//
// Exemplo:
//
//	if sessao, ok := req.Cookie("session_id"); ok {
//	    // ...
//	}
func (r Request) Cookie(name string) (string, bool) {
	c, err := r.Request.Cookie(name)
	if err != nil {
		return "", false
	}
	return c.Value, true
}

// IP devolve o endereço IP do cliente. Dá prioridade ao cabeçalho
// X-Forwarded-For (comum atrás de um proxy reverso ou load balancer),
// usando RemoteAddr como alternativa quando o cabeçalho não existir.
//
// Nota: X-Forwarded-For é definido pelo cliente ou por um proxy
// intermédio, pelo que não deve ser tratado como um valor de confiança
// absoluta em contextos de segurança sem um proxy fidedigno a filtrá-lo.
func (r Request) IP() string {
	if fwd := r.Header("X-Forwarded-For"); fwd != "" {
		// X-Forwarded-For pode vir como "cliente, proxy1, proxy2"
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr
}

// ContentType devolve o Content-Type do pedido, sem os parâmetros
// adicionais (ex.: "application/json; charset=utf-8" torna-se
// "application/json").
func (r Request) ContentType() string {
	ct := r.Header("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = ct[:idx]
	}
	return strings.TrimSpace(ct)
}

// Is verifica se o Content-Type do pedido corresponde ao indicado.
//
// Exemplo:
//
//	if req.Is("application/json") { ... }
func (r Request) Is(contentType string) bool {
	return r.ContentType() == contentType
}

// Accepts verifica se o cabeçalho Accept do pedido inclui o
// content-type indicado (ou aceita qualquer tipo via "*/*"). Não faz
// parsing dos valores de qualidade (q=...) do cabeçalho — para esse
// nível de precisão, use uma biblioteca de negociação de conteúdo.
//
// Exemplo:
//
//	if req.Accepts("application/json") { ... }
func (r Request) Accepts(contentType string) bool {
	accept := r.Header("Accept")
	return strings.Contains(accept, contentType) || strings.Contains(accept, "*/*")
}

// AcceptsJSON é um atalho para Accepts("application/json").
func (r Request) AcceptsJSON() bool {
	return r.Accepts("application/json")
}

// AcceptsHTML é um atalho para Accepts("text/html").
func (r Request) AcceptsHTML() bool {
	return r.Accepts("text/html")
}

// XHR indica se o pedido foi feito através de XMLHttpRequest
// (cabeçalho X-Requested-With), uma convenção usada por várias
// bibliotecas front-end para distinguir chamadas AJAX de navegação
// normal do browser.
func (r Request) XHR() bool {
	return r.Header("X-Requested-With") == "XMLHttpRequest"
}

// Host devolve o anfitrião (domínio + porta, se aplicável) indicado no
// pedido.
func (r Request) Host() string {
	return r.Request.Host
}

// Path devolve apenas o caminho da URL do pedido, sem a query string.
func (r Request) Path() string {
	return r.URL.Path
}

// Scheme tenta determinar o esquema (http ou https) do pedido
// original, tendo em conta que a aplicação pode estar atrás de um
// proxy reverso que termina o TLS antes de reencaminhar para o Go
// (nesse caso, o cabeçalho X-Forwarded-Proto é a fonte correta).
func (r Request) Scheme() string {
	if proto := r.Header("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// FullURL reconstrói a URL absoluta do pedido (esquema + anfitrião +
// caminho + query string), útil para gerar links absolutos em
// respostas (ex.: emails de confirmação, redirecionamentos, Location).
func (r Request) FullURL() string {
	return r.Scheme() + "://" + r.Host() + r.URL.RequestURI()
}

// UserAgent devolve o cabeçalho User-Agent do pedido.
func (r Request) UserAgent() string {
	return r.Header("User-Agent")
}

// Referer devolve o cabeçalho Referer do pedido (mantém-se a grafia
// histórica "Referer", tal como definida no protocolo HTTP).
func (r Request) Referer() string {
	return r.Header("Referer")
}

// ----------------------------------------------------------------------
// Corpo do pedido
// ----------------------------------------------------------------------

// Body lê e devolve o corpo bruto do pedido como []byte. Pode ser
// chamado várias vezes — o resultado fica em cache na própria
// instância de Request — ao contrário de ler r.Request.Body
// diretamente, que só pode ser consumido uma única vez.
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

// BindJson descodifica o corpo JSON do pedido para o destino indicado.
// Usa Body() internamente, pelo que pode ser combinado com outras
// leituras do corpo sem qualquer conflito.
//
// Exemplo:
//
//	var payload struct {
//	    Nome string `json:"nome"`
//	}
//	if err := req.BindJson(&payload); err != nil {
//	    return res.Status(400).WithJson(kite.ErrorResponse{Error: "JSON inválido", Status: 400})
//	}
func (r *Request) BindJson(dest interface{}) error {
	data, err := r.Body()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// maxMultipartMemory define o limite de memória (em bytes) usado ao
// fazer o parse de formulários multipart antes de os campos grandes
// (ficheiros) transbordarem para ficheiros temporários em disco.
const maxMultipartMemory = 32 << 20 // 32 MB

// FormValue faz o parse do corpo como application/x-www-form-urlencoded
// ou multipart/form-data e devolve o valor do campo indicado. Chamadas
// seguintes reaproveitam o formulário já processado.
//
// Exemplo:
//
//	nome := req.FormValue("nome")
func (r *Request) FormValue(name string) string {
	_ = r.Request.ParseMultipartForm(maxMultipartMemory) // erro ignorado: o corpo pode não ser multipart
	return r.Request.FormValue(name)
}

// FormValues devolve todos os valores de formulário submetidos
// (application/x-www-form-urlencoded ou multipart/form-data), no
// mesmo formato de url.Values usado pelo net/http.
func (r *Request) FormValues() (url.Values, error) {
	if err := r.Request.ParseMultipartForm(maxMultipartMemory); err != nil && err != http.ErrNotMultipart {
		return nil, err
	}
	return r.Request.Form, nil
}

// FormFile devolve o ficheiro submetido através do campo multipart
// indicado, pronto a ser lido ou copiado para disco. É
// responsabilidade de quem chama fechar o multipart.File devolvido.
//
// Exemplo:
//
//	ficheiro, cabecalho, err := req.FormFile("avatar")
//	if err != nil {
//	    return res.Status(400).WithJson(kite.ErrorResponse{Error: "ficheiro em falta", Status: 400})
//	}
//	defer ficheiro.Close()
func (r *Request) FormFile(name string) (multipart.File, *multipart.FileHeader, error) {
	if err := r.Request.ParseMultipartForm(maxMultipartMemory); err != nil {
		return nil, nil, err
	}
	return r.Request.FormFile(name)
}

// SaveUploadedFile é um atalho que lê o ficheiro submetido no campo
// multipart indicado e grava-o em destino no disco, tratando da
// abertura, cópia e fecho de ambos os ficheiros.
//
// Exemplo:
//
//	if err := req.SaveUploadedFile("avatar", "./uploads/avatar.png"); err != nil {
//	    return res.Status(500).WithJson(kite.ErrorResponse{Error: "falha ao guardar ficheiro", Status: 500})
//	}
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

// ----------------------------------------------------------------------
// Contexto
// ----------------------------------------------------------------------

// Ctx devolve o context.Context associado ao pedido, útil para
// propagar cancelamento, prazos (timeouts) e valores para camadas
// internas — por exemplo, para passar diretamente a consultas à base
// de dados que aceitem um context.Context.
func (r Request) Ctx() context.Context {
	return r.Request.Context()
}

// WithContext devolve uma cópia do Request com o context.Context
// substituído pelo indicado, mantendo Params e o restante estado
// intactos. Útil em middlewares que precisem de propagar valores
// adicionais (ex.: o utilizador autenticado) pela cadeia de handlers.
//
// Exemplo:
//
//	ctx := context.WithValue(req.Ctx(), userCtxKey, user)
//	req = req.WithContext(ctx)
func (r Request) WithContext(ctx context.Context) Request {
	r.Request = r.Request.WithContext(ctx)
	return r
}