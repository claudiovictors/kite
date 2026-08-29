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

// Response encapsula o http.ResponseWriter nativo e expõe métodos
// encadeáveis (chainable) que terminam sempre num verbo de escrita
// (WithJson, Send, Redirect, File, ...) devolvendo um error — a
// assinatura esperada pelo HandlerFunc.
//
// Response é passado por valor entre os métodos que apenas configuram
// estado (Status, SetHeader, Type, ...), mas escreve sempre no mesmo
// http.ResponseWriter subjacente (uma interface), pelo que o
// encadeamento se comporta como esperado:
//
//	return res.Status(201).SetHeader("X-Request-Id", id).Json(payload)
type Response struct {
	http.ResponseWriter

	// req guarda o *http.Request original do pedido que originou esta
	// resposta. É necessário para operações como File/Download, que
	// delegam em http.ServeFile.
	req *http.Request

	// views aponta para o engine de templates configurado no App (via
	// App.LoadViews), permitindo que Render(name, data) funcione sem
	// precisar receber o engine em todo handler.
	views *template.Engine

	statusCode int
	written    bool
}

// newResponse cria uma Response a partir do http.ResponseWriter e do
// *http.Request nativos, e do engine de views configurado no App (pode
// ser nil se app.LoadViews nunca foi chamado — nesse caso Render devolve
// erro explicando o que fazer). Usado internamente por App.ServeHTTP.
func newResponse(w http.ResponseWriter, r *http.Request, views *template.Engine) Response {
	return Response{ResponseWriter: w, req: r, statusCode: http.StatusOK, views: views}
}

// writeHeaderOnce garante que WriteHeader é chamado no máximo uma vez
// por resposta. Isto evita o aviso "superfluous response.WriteHeader
// call" e respostas corrompidas quando dois métodos de escrita (ex.:
// Json seguido de Redirect por engano) tentam escrever o cabeçalho
// separadamente.
func (res *Response) writeHeaderOnce() {
	if !res.written {
		res.WriteHeader(res.statusCode)
		res.written = true
	}
}

// Written indica se esta resposta já escreveu o cabeçalho de estado
// (ou seja, se já foi "enviada" ao cliente). Útil em middlewares que
// precisem de decidir se ainda podem alterar cabeçalhos ou o código de
// estado.
func (res Response) Written() bool {
	return res.written
}

// ----------------------------------------------------------------------
// Configuração da resposta (estado, cabeçalhos, cookies)
// ----------------------------------------------------------------------

// WithStatus define o código de estado a usar na próxima escrita.
// Mantido por compatibilidade com código existente — para código novo,
// prefira Status, com o mesmo comportamento.
func (res Response) WithStatus(code int) Response {
	res.statusCode = code
	return res
}

// Status define o código de estado HTTP a usar na próxima escrita, ao
// estilo do res.status() do Express. É encadeável.
//
// Exemplo:
//
//	return res.Status(201).Json(payload)
func (res Response) Status(code int) Response {
	res.statusCode = code
	return res
}

// SetHeader define um cabeçalho na resposta. É encadeável.
//
// Nota: como o protocolo HTTP não permite alterar cabeçalhos depois de
// enviado o código de estado, este método deve ser chamado antes de
// qualquer método de escrita (Json, Send, WithHtml, ...).
//
// Exemplo:
//
//	return res.SetHeader("X-Request-Id", reqID).Json(payload)
func (res Response) SetHeader(key, value string) Response {
	res.Header().Set(key, value)
	return res
}

// Type define o Content-Type da resposta manualmente. É encadeável —
// útil quando nenhum dos atalhos (Json, WithHtml, WithText) serve,
// por exemplo para responder XML ou um formato binário customizado.
func (res Response) Type(contentType string) Response {
	res.Header().Set("Content-Type", contentType)
	return res
}

// Vary acrescenta um valor ao cabeçalho Vary, indicando às caches
// (proxies, CDNs, browser) que a resposta varia consoante o cabeçalho
// de pedido indicado (ex.: "Accept-Encoding", "Authorization").
func (res Response) Vary(header string) Response {
	res.Header().Add("Vary", header)
	return res
}

// CacheControl define o cabeçalho Cache-Control da resposta. É
// encadeável.
//
// Exemplo:
//
//	return res.CacheControl("public, max-age=3600").Json(dados)
func (res Response) CacheControl(value string) Response {
	res.Header().Set("Cache-Control", value)
	return res
}

// Cookie adiciona um Set-Cookie à resposta. É encadeável.
//
// Exemplo:
//
//	return res.Cookie(&http.Cookie{
//	    Name:     "session_id",
//	    Value:    sessionID,
//	    HttpOnly: true,
//	    Path:     "/",
//	}).NoContent()
func (res Response) Cookie(cookie *http.Cookie) Response {
	http.SetCookie(res.ResponseWriter, cookie)
	return res
}

// ClearCookie remove um cookie do cliente, enviando um Set-Cookie com
// o mesmo nome, valor vazio e data de expiração no passado — a forma
// convencional de "apagar" um cookie em HTTP, já que o servidor não
// consegue remover diretamente o que está guardado no browser.
//
// Exemplo:
//
//	return res.ClearCookie("session_id").Redirect().To("/login")
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

// ----------------------------------------------------------------------
// Corpo da resposta
// ----------------------------------------------------------------------

// WithJson serializa v para JSON e escreve-o na resposta, definindo o
// Content-Type apropriado.
func (res Response) WithJson(v interface{}) error {
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.writeHeaderOnce()
	return json.NewEncoder(res).Encode(v)
}

// Json é um alias de WithJson, ao estilo do res.json() do Express.
func (res Response) Json(v interface{}) error {
	return res.WithJson(v)
}

// WithText escreve uma resposta em texto simples.
func (res Response) WithText(text string) error {
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.writeHeaderOnce()
	_, err := res.Write([]byte(text))
	return err
}

// WithHtml escreve uma resposta HTML já pronta (string), sem passar
// por nenhum motor de templates. Para renderizar uma view a partir de
// um ficheiro, veja Render.
func (res Response) WithHtml(html string) error {
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.writeHeaderOnce()
	_, err := res.Write([]byte(html))
	return err
}

// Send envia a resposta em JSON por predefinição — incluindo strings,
// ao contrário do res.send() do Express (que trata strings como texto
// simples). Isto mantém o comportamento previsível: tudo o que passa
// por Send tem sempre Content-Type application/json, exceto nil, que
// só escreve o código de estado sem corpo.
//
// Exemplo:
//
//	return res.Send("Hello, World") // -> "Hello, World" (corpo JSON)
//	return res.Send(user)           // -> {"id":1,"nome":"Ana",...}
//	return res.Send(nil)            // -> sem corpo, só o status
func (res Response) Send(v interface{}) error {
	if v == nil {
		res.writeHeaderOnce()
		return nil
	}
	return res.WithJson(v)
}

// SendStatus define o código de estado e envia o respetivo texto
// convencional como corpo (ex.: res.SendStatus(404) escreve "Not
// Found"), ao estilo do res.sendStatus() do Express.
func (res Response) SendStatus(code int) error {
	res.statusCode = code
	return res.WithText(http.StatusText(code))
}

// NoContent responde com 204 (No Content) e sem corpo. Útil para
// operações DELETE/PUT que não devolvem nenhum payload.
func (res Response) NoContent() error {
	res.statusCode = http.StatusNoContent
	res.writeHeaderOnce()
	return nil
}

// ----------------------------------------------------------------------
// Redirecionamento
// ----------------------------------------------------------------------

// Redirect inicia um redirecionamento encadeável, no estilo do
// return redirect()->... do Laravel:
//
//	return res.Redirect().To("/login")
//	return res.Redirect().To("/login", http.StatusMovedPermanently)
//	return res.Redirect().Permanently("/novo-endereco")
//	return res.Redirect().Back(req)
func (res Response) Redirect() *Redirector {
	return &Redirector{res: res}
}

// Redirector é devolvido por Response.Redirect() e concentra as
// variações de redirecionamento (To, Permanently, Back) atrás de uma
// única API encadeável.
type Redirector struct {
	res Response
}

// To redireciona para a URL informada. code é opcional e usa 302
// (Found) por predefinição, à semelhança do Express/Laravel.
func (rd *Redirector) To(url string, code ...int) error {
	status := http.StatusFound
	if len(code) > 0 {
		status = code[0]
	}
	rd.res.Header().Set("Location", url)
	rd.res.statusCode = status
	rd.res.writeHeaderOnce()
	return nil
}

// Permanently redireciona com 301 (Moved Permanently) — sinaliza a
// navegadores e crawlers que o endereço antigo não deve mais ser usado.
func (rd *Redirector) Permanently(url string) error {
	return rd.To(url, http.StatusMovedPermanently)
}

// Back redireciona de volta para o cabeçalho Referer do pedido
// original. fallback (opcional, padrão "/") é usado quando o Referer
// não vier preenchido — equivalente ao redirect()->back() do Laravel.
//
//	return res.Redirect().Back(req)
//	return res.Redirect().Back(req, "/dashboard")
func (rd *Redirector) Back(req Request, fallback ...string) error {
	ref := req.Referer()
	if ref == "" {
		ref = "/"
		if len(fallback) > 0 {
			ref = fallback[0]
		}
	}
	return rd.To(ref)
}

// ----------------------------------------------------------------------
// Views
// ----------------------------------------------------------------------

// Render renderiza a view identificada por name usando o motor de
// templates configurado no App (via app.LoadViews(dir, ext)), no
// estilo do view()/Blade do Laravel:
//
//	return res.Render("index", kite.Map{"Title": "Hello, World!"})
//
// Devolve um erro explicativo se app.LoadViews nunca foi chamado.
func (res Response) Render(name string, data interface{}) error {
	if res.views == nil {
		return fmt.Errorf("kite: nenhuma view engine configurada — chame app.LoadViews(dir, ext) antes de usar res.Render")
	}
	html, err := res.views.RenderToString(name, data)
	if err != nil {
		return err
	}
	return res.WithHtml(html)
}

// RenderWith renderiza a view usando um engine específico, passado
// diretamente — útil quando você mantém mais de um Engine (ex.: views
// públicas vs. templates de e-mail) e precisa fugir do padrão
// app.LoadViews/res.Render.
func (res Response) RenderWith(engine *template.Engine, name string, data interface{}) error {
	html, err := engine.RenderToString(name, data)
	if err != nil {
		return err
	}
	return res.WithHtml(html)
}

// ----------------------------------------------------------------------
// Ficheiros
// ----------------------------------------------------------------------

// Attachment define o cabeçalho Content-Disposition como anexo
// (attachment), sugerindo ao browser o nome de ficheiro a usar ao
// gravar o download. É encadeável — combine com Send/WithText/Json
// quando o conteúdo não vem diretamente do disco.
func (res Response) Attachment(filename string) Response {
	res.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	return res
}

// File serve o conteúdo de um ficheiro do disco como resposta
// (imagens, PDFs, ficheiros estáticos, ...), deixando ao browser a
// decisão de como o exibir. Usa o *http.Request original guardado na
// Response, pelo que suporta corretamente cabeçalhos condicionais
// (If-Modified-Since, If-None-Match) e pedidos parciais (Range).
//
// Exemplo:
//
//	return res.File("./public/logo.png")
func (res Response) File(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	http.ServeFile(res.ResponseWriter, res.req, path)
	return nil
}

// Download serve um ficheiro do disco forçando o download no browser
// (Content-Disposition: attachment), em vez de o exibir inline como
// File faz. Se filename vier vazio, usa o nome base do caminho.
//
// Exemplo:
//
//	return res.Download("./storage/relatorio.pdf", "relatorio-2026.pdf")
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