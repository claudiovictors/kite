package kite

import (
	"log"
	"net/http"
)

// ErrorResponse é o formato padrão de erro em JSON devolvido pelo
// framework (404, 500, etc), a menos que NotFoundHandler/ErrorHandler
// sejam sobrescritos pelo usuário.
type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// App é o ponto de entrada do framework, equivalente ao "app" do Express.
type App struct {
	router      *Router
	middlewares []MiddlewareFunc

	// NotFoundHandler é chamado quando nenhuma rota bate com o
	// método+path da requisição. Por padrão responde 404 em JSON.
	NotFoundHandler HandlerFunc

	// ErrorHandler é chamado quando um handler retorna um error não
	// tratado. Por padrão loga no stdout e responde 500 em JSON.
	ErrorHandler func(req Request, res Response, err error)
}

// New cria uma nova instância do App, já com handlers padrão de 404 e
// erro respondendo em JSON.
func New() *App {
	app := &App{router: newRouter()}
	app.NotFoundHandler = defaultNotFoundHandler
	app.ErrorHandler = defaultErrorHandler
	return app
}

func defaultNotFoundHandler(req Request, res Response) error {
	return res.Status(http.StatusNotFound).WithJson(ErrorResponse{
		Error:  "rota não encontrada: " + req.Method + " " + req.URL.Path,
		Status: http.StatusNotFound,
	})
}

func defaultErrorHandler(req Request, res Response, err error) {
	log.Printf("kite: erro no handler %s %s: %v", req.Method, req.URL.Path, err)
	if writeErr := res.Status(http.StatusInternalServerError).WithJson(ErrorResponse{
		Error:  "erro interno do servidor",
		Status: http.StatusInternalServerError,
	}); writeErr != nil {
		log.Printf("kite: falha ao escrever resposta de erro: %v", writeErr)
	}
}

// Use registra um middleware global, aplicado a todas as rotas
// registradas a partir deste ponto.
func (a *App) Use(mw MiddlewareFunc) {
	a.middlewares = append(a.middlewares, mw)
}

func (a *App) Get(path string, h HandlerFunc)    { a.register(http.MethodGet, path, h) }
func (a *App) Post(path string, h HandlerFunc)   { a.register(http.MethodPost, path, h) }
func (a *App) Put(path string, h HandlerFunc)    { a.register(http.MethodPut, path, h) }
func (a *App) Delete(path string, h HandlerFunc) { a.register(http.MethodDelete, path, h) }
func (a *App) Patch(path string, h HandlerFunc)  { a.register(http.MethodPatch, path, h) }

func (a *App) register(method, path string, h HandlerFunc) {
	a.router.Add(method, path, chain(h, a.middlewares))
}

// Group cria um sub-roteador com prefixo de path, herdando os
// middlewares já registrados no App até este ponto.
func (a *App) Group(prefix string) *RouteGroup {
	inherited := make([]MiddlewareFunc, len(a.middlewares))
	copy(inherited, a.middlewares)
	return &RouteGroup{app: a, prefix: prefix, middlewares: inherited}
}

// ServeHTTP implementa http.Handler, permitindo usar o App diretamente
// com http.ListenAndServe ou em testes com httptest.
//
// Mudanças em relação à versão anterior:
//   - rota não encontrada -> NotFoundHandler (JSON 404), não mais
//     http.NotFound (que respondia texto puro "404 page not found")
//   - handler retornando error -> ErrorHandler (JSON 500); antes só
//     era logado e o cliente ficava sem nenhuma resposta
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, params, ok := a.router.Match(r.Method, r.URL.Path)
	req := Request{Request: r, Params: params}
	res := newResponse(w)

	if !ok {
		_ = a.NotFoundHandler(req, res)
		return
	}

	if err := handler(req, res); err != nil {
		a.ErrorHandler(req, res, err)
	}
}

// Listen inicia o servidor HTTP no endereço informado (ex: ":3000").
func (a *App) Listen(addr string) error {
	log.Printf("kite: ouvindo em %s", addr)
	return http.ListenAndServe(addr, a)
}

// RouteGroup representa um conjunto de rotas com prefixo comum,
// equivalente ao Router() do Express (ex: app.Group("/api")).
type RouteGroup struct {
	app         *App
	prefix      string
	middlewares []MiddlewareFunc
}

// Use registra um middleware que se aplica só às rotas deste grupo.
func (g *RouteGroup) Use(mw MiddlewareFunc) {
	g.middlewares = append(g.middlewares, mw)
}

func (g *RouteGroup) Get(path string, h HandlerFunc)    { g.register(http.MethodGet, path, h) }
func (g *RouteGroup) Post(path string, h HandlerFunc)   { g.register(http.MethodPost, path, h) }
func (g *RouteGroup) Put(path string, h HandlerFunc)    { g.register(http.MethodPut, path, h) }
func (g *RouteGroup) Delete(path string, h HandlerFunc) { g.register(http.MethodDelete, path, h) }
func (g *RouteGroup) Patch(path string, h HandlerFunc)  { g.register(http.MethodPatch, path, h) }

func (g *RouteGroup) register(method, path string, h HandlerFunc) {
	full := joinPrefix(g.prefix, path)
	g.app.router.Add(method, full, chain(h, g.middlewares))
}

func joinPrefix(prefix, path string) string {
	if prefix == "" {
		return path
	}
	if path == "/" {
		return prefix
	}
	return prefix + path
}