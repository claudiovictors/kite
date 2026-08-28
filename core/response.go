package kite

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/claudiovictors/kite/template"
)

/**
 * Response encapsula o http.ResponseWriter nativo do Go e disponibiliza métodos encadeáveis (chainable)
 * para a escrita de respostas HTTP na aplicação Kite.
 *
 * Exemplo:
 *  return res.Status(201).SetHeader("X-Request-Id", id).Json(payload)
 */
type Response struct {
	http.ResponseWriter

	req        *http.Request
	statusCode int
	written    bool
}

/**
 * newResponse cria e inicializa uma nova instância de Response a partir do http.ResponseWriter e do *http.Request nativos.
 *
 * @param w http.ResponseWriter
 * @param r *http.Request
 * @return Response
 */
func newResponse(w http.ResponseWriter, r *http.Request) Response {
	return Response{ResponseWriter: w, req: r, statusCode: http.StatusOK}
}

/**
 * writeHeaderOnce garante a execução única de WriteHeader por resposta para evitar chamadas supérfluas e corrupção de cabeçalhos.
 */
func (res *Response) writeHeaderOnce() {
	if !res.written {
		res.WriteHeader(res.statusCode)
		res.written = true
	}
}

/**
 * Written verifica se os cabeçalhos e o código de estado da resposta já foram enviados ao cliente.
 *
 * @return bool
 */
func (res Response) Written() bool {
	return res.written
}

/* ---------------------------------------------------------------------- */
/* Configuração da resposta (estado, cabeçalhos, cookies)                 */
/* ---------------------------------------------------------------------- */

/**
 * WithStatus define o código de estado HTTP a ser utilizado na próxima escrita de resposta.
 *
 * @param code int
 * @return Response
 */
func (res Response) WithStatus(code int) Response {
	res.statusCode = code
	return res
}

/**
 * Status define o código de estado HTTP a ser utilizado na próxima escrita de resposta.
 *
 * Exemplo:
 *  return res.Status(201).Json(payload)
 *
 * @param code int
 * @return Response
 */
func (res Response) Status(code int) Response {
	res.statusCode = code
	return res
}

/**
 * SetHeader define o valor de um cabeçalho HTTP de resposta.
 *
 * Exemplo:
 *  return res.SetHeader("X-Request-Id", reqID).Json(payload)
 *
 * @param key string
 * @param value string
 * @return Response
 */
func (res Response) SetHeader(key, value string) Response {
	res.Header().Set(key, value)
	return res
}

/**
 * Type define manualmente o valor do cabeçalho Content-Type da resposta.
 *
 * @param contentType string
 * @return Response
 */
func (res Response) Type(contentType string) Response {
	res.Header().Set("Content-Type", contentType)
	return res
}

/**
 * Vary adiciona um parâmetro ao cabeçalho Vary para controle de cache por proxies e navegadores.
 *
 * @param header string
 * @return Response
 */
func (res Response) Vary(header string) Response {
	res.Header().Add("Vary", header)
	return res
}

/**
 * CacheControl define o cabeçalho HTTP Cache-Control.
 *
 * Exemplo:
 *  return res.CacheControl("public, max-age=3600").Json(dados)
 *
 * @param value string
 * @return Response
 */
func (res Response) CacheControl(value string) Response {
	res.Header().Set("Cache-Control", value)
	return res
}

/**
 * Cookie injeta um cabeçalho Set-Cookie na resposta HTTP.
 *
 * Exemplo:
 *  return res.Cookie(&http.Cookie{
 *      Name:     "session_id",
 *      Value:    sessionID,
 *      HttpOnly: true,
 *      Path:     "/",
 *  }).NoContent()
 *
 * @param cookie *http.Cookie
 * @return Response
 */
func (res Response) Cookie(cookie *http.Cookie) Response {
	http.SetCookie(res.ResponseWriter, cookie)
	return res
}

/**
 * ClearCookie expira e remove um cookie do cliente enviando uma instrução Set-Cookie com data no passado.
 *
 * Exemplo:
 *  return res.ClearCookie("session_id").Redirect("/login")
 *
 * @param name string
 * @return Response
 */
func (res Response) ClearCookie(name string) Response {
	http.SetCookie(res.ResponseWriter, &http.Cookie{
		Name:    name,
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
	return res
}

/* ---------------------------------------------------------------------- */
/* Corpo da resposta                                                      */
/* ---------------------------------------------------------------------- */

/**
 * WithJson serializa a interface fornecida para formato JSON, define o Content-Type apropriado e envia a resposta.
 *
 * @param v interface{}
 * @return error
 */
func (res Response) WithJson(v interface{}) error {
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.writeHeaderOnce()
	return json.NewEncoder(res).Encode(v)
}

/**
 * Json é um método utilitário correspondente a WithJson para serialização JSON na aplicação Kite.
 *
 * @param v interface{}
 * @return error
 */
func (res Response) Json(v interface{}) error {
	return res.WithJson(v)
}

/**
 * WithText escreve uma resposta com o tipo de conteúdo em texto simples (text/plain).
 *
 * @param text string
 * @return error
 */
func (res Response) WithText(text string) error {
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.writeHeaderOnce()
	_, err := res.Write([]byte(text))
	return err
}

/**
 * WithHtml escreve uma resposta em formato HTML a partir de uma string fornecida.
 *
 * @param html string
 * @return error
 */
func (res Response) WithHtml(html string) error {
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.writeHeaderOnce()
	_, err := res.Write([]byte(html))
	return err
}

/**
 * Send processa e envia a resposta com serialização JSON por padrão. Se o parâmetro for nil, escreve apenas o código de estado.
 *
 * Exemplo:
 *  return res.Send("Hello, World") // -> "Hello, World" (payload JSON)
 *  return res.Send(user)           // -> {"id":1,"nome":"Ana"}
 *  return res.Send(nil)            // -> sem corpo, apenas status
 *
 * @param v interface{}
 * @return error
 */
func (res Response) Send(v interface{}) error {
	if v == nil {
		res.writeHeaderOnce()
		return nil
	}
	return res.WithJson(v)
}

/**
 * SendStatus define o código de estado HTTP e envia a mensagem de texto correspondente ao padrão HTTP.
 *
 * @param code int
 * @return error
 */
func (res Response) SendStatus(code int) error {
	res.statusCode = code
	return res.WithText(http.StatusText(code))
}

/**
 * NoContent envia uma resposta HTTP 204 (No Content) sem corpo de resposta.
 *
 * @return error
 */
func (res Response) NoContent() error {
	res.statusCode = http.StatusNoContent
	res.writeHeaderOnce()
	return nil
}

/**
 * Redirect efetua o redirecionamento HTTP para a URL indicada. Se omitido, o código de estado padrão será 302 (Found).
 *
 * Exemplo:
 *  return res.Redirect("/login")
 *  return res.Redirect("/login", http.StatusMovedPermanently)
 *
 * @param url string
 * @param code ...int
 * @return error
 */
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

/**
 * Render utiliza o motor de templates do Kite para processar e renderizar uma view HTML com dados.
 *
 * Exemplo:
 *  return res.Render(views, "users/show", user)
 *
 * @param engine *template.Engine
 * @param name string
 * @param data interface{}
 * @return error
 */
func (res Response) Render(engine *template.Engine, name string, data interface{}) error {
	html, err := engine.RenderToString(name, data)
	if err != nil {
		return err
	}
	return res.WithHtml(html)
}

/* ---------------------------------------------------------------------- */
/* Ficheiros                                                              */
/* ---------------------------------------------------------------------- */

/**
 * Attachment adiciona o cabeçalho Content-Disposition indicando download de anexo com o nome do arquivo sugerido.
 *
 * @param filename string
 * @return Response
 */
func (res Response) Attachment(filename string) Response {
	res.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	return res
}

/**
 * File realiza a entrega de um arquivo presente no disco respeitando requisições condicionais e parciais.
 *
 * Exemplo:
 *  return res.File("./public/logo.png")
 *
 * @param path string
 * @return error
 */
func (res Response) File(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	http.ServeFile(res.ResponseWriter, res.req, path)
	return nil
}

/**
 * Download força o download de um arquivo do disco no navegador enviando o cabeçalho Content-Disposition adequado.
 *
 * Exemplo:
 *  return res.Download("./storage/relatorio.pdf", "relatorio-2026.pdf")
 *
 * @param path string
 * @param filename string
 * @return error
 */
func (res Response) Download(path, filename string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	if filename == "" {
		filename = filepath.Base(path)
	}
	res.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(res.ResponseWriter, res.req, path)
	return nil
}