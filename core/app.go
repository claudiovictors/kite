package kite

import (
	"fmt"
	"log"
	"net/http"

	"github.com/claudiovictors/kite/template"
)

/**
 * ErrorResponse encapsulates the standard JSON structure used for error payloads.
 *
 * @property {string} Error - Detailed description of the error occurrence.
 * @property {int} Status - Associated HTTP status code.
 */
type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

/**
 * ServerConfig defines metadata for OpenAPI server endpoints.
 */
type ServerConfig struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

/**
 * Config encapsulates configuration parameters for the Kite application,
 * including OpenAPI specifications and Scalar UI presentation.
 */
type Config struct {
	Title        string         `json:"title,omitempty"`
	Version      string         `json:"version,omitempty"`
	Description  string         `json:"description,omitempty"`
	DocsURL      string         `json:"docsUrl,omitempty"`
	OpenAPIURL   string         `json:"openApiUrl,omitempty"`
	ScalarTheme  ScalarTheme    `json:"scalarTheme,omitempty"`
	ScalarLayout string         `json:"scalarLayout,omitempty"`
	DisableDocs  bool           `json:"disableDocs,omitempty"`
	Servers      []ServerConfig `json:"servers,omitempty"`
}

/**
 * App represents the core engine of the kite package, managing the router,
 * global middlewares, the optional view engine, and default lifecycle
 * error/not-found handlers.
 */
type App struct {
	config      Config
	router      *Router
	middlewares []MiddlewareFunc
	views       *template.Engine

	NotFoundHandler HandlerFunc
	ErrorHandler    func(req Request, res Response, err error)
}

/**
 * New instantiates and returns a new core App application instance
 * with a pre-configured router and default handlers.
 *
 * @param {...Config} configs - Optional application and OpenAPI/Scalar configurations.
 * @return {*App} Pointer to the newly created App instance.
 */
func New(configs ...Config) *App {
	var cfg Config
	if len(configs) > 0 {
		cfg = configs[0]
	}
	if cfg.Title == "" {
		cfg.Title = "Kite API"
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.DocsURL == "" {
		cfg.DocsURL = "/docs"
	}
	if cfg.OpenAPIURL == "" {
		cfg.OpenAPIURL = "/openapi.json"
	}
	if cfg.ScalarTheme == "" {
		cfg.ScalarTheme = ThemeLaserwave
	}
	if cfg.ScalarLayout == "" {
		cfg.ScalarLayout = "classic"
	}

	app := &App{
		config: cfg,
		router: newRouter(),
	}
	app.NotFoundHandler = defaultNotFoundHandler
	app.ErrorHandler = defaultErrorHandler

	// Documentation routes are registered here, inside New(), synchronously
	// and before any request is served — eliminating the race condition that
	// would occur if registration happened inside ServeHTTP, which is called
	// concurrently by multiple goroutines.
	if !cfg.DisableDocs {
		app.registerDocs()
	}

	return app
}

/**
 * defaultNotFoundHandler manages requests that fail to match any registered route.
 *
 * @param {Request} req - The wrapped incoming HTTP request.
 * @param {Response} res - The wrapped HTTP response writer.
 * @return {error} Returns any write error encountered during response delivery.
 */
func defaultNotFoundHandler(req Request, res Response) error {
	return res.Status(http.StatusNotFound).WithJson(ErrorResponse{
		Error:  "rota não encontrada: " + req.Method + " " + req.URL.Path,
		Status: http.StatusNotFound,
	})
}

/**
 * defaultErrorHandler acts as the global fallback recovery for unhandled handler errors.
 *
 * @param {Request} req - The incoming request that triggered the error.
 * @param {Response} res - The response context.
 * @param {error} err - The captured error interface.
 */
func defaultErrorHandler(req Request, res Response, err error) {
	log.Printf("kite: erro no handler %s %s: %v", req.Method, req.URL.Path, err)
	if writeErr := res.Status(http.StatusInternalServerError).WithJson(ErrorResponse{
		Error:  "erro interno do servidor",
		Status: http.StatusInternalServerError,
	}); writeErr != nil {
		log.Printf("kite: falha ao escrever resposta de erro: %v", writeErr)
	}
}

/**
 * Use registers a global middleware function to be executed across all incoming requests.
 *
 * @param {MiddlewareFunc} mw - The middleware function to append.
 */
func (a *App) Use(mw MiddlewareFunc) {
	a.middlewares = append(a.middlewares, mw)
}

/**
 * Get registers a route handling HTTP GET requests and returns a *Route, que permite
 * encadear .Middleware(...) (específico da rota) e .Name(...) (para reverse routing).
 *
 *	app.Get("/users/:id", showUser).
 *	    Middleware(auth.RequireAuth).
 *	    Name("users.show")
 *
 * @param {string} path - The target route pattern.
 * @param {HandlerFunc} h - The designated handler function.
 * @return {*Route}
 */
func (a *App) Get(path string, h HandlerFunc) *Route { return a.register(http.MethodGet, path, h) }

/**
 * Post registers a route handling HTTP POST requests. See Get for chaining details.
 *
 * @param {string} path
 * @param {HandlerFunc} h
 * @return {*Route}
 */
func (a *App) Post(path string, h HandlerFunc) *Route { return a.register(http.MethodPost, path, h) }

/**
 * Put registers a route handling HTTP PUT requests. See Get for chaining details.
 *
 * @param {string} path
 * @param {HandlerFunc} h
 * @return {*Route}
 */
func (a *App) Put(path string, h HandlerFunc) *Route { return a.register(http.MethodPut, path, h) }

/**
 * Delete registers a route handling HTTP DELETE requests. See Get for chaining details.
 *
 * @param {string} path
 * @param {HandlerFunc} h
 * @return {*Route}
 */
func (a *App) Delete(path string, h HandlerFunc) *Route {
	return a.register(http.MethodDelete, path, h)
}

/**
 * Patch registers a route handling HTTP PATCH requests. See Get for chaining details.
 *
 * @param {string} path
 * @param {HandlerFunc} h
 * @return {*Route}
 */
func (a *App) Patch(path string, h HandlerFunc) *Route {
	return a.register(http.MethodPatch, path, h)
}

/**
 * register internally binds an HTTP method and path combination with the app's
 * global middleware chain, guardando o handler original (rawHandler) e a lista
 * de middlewares base no *node subjacente — necessário para que Route.Middleware
 * possa recompor a cadeia depois, já incluindo middlewares específicos da rota.
 *
 * @param {string} method - The HTTP verb.
 * @param {string} path - The route endpoint path.
 * @param {HandlerFunc} h - The original (unwrapped) handler.
 * @return {*Route}
 */
func (a *App) register(method, path string, h HandlerFunc) *Route {
	base := make([]MiddlewareFunc, len(a.middlewares))
	copy(base, a.middlewares)

	n := a.router.Add(method, path, chain(h, base))
	// BUG CORRIGIDO: Router.Add guarda em rawHandler o mesmo handler que
	// recebe (já encadeado com base). Sem esta linha, Route.Middleware
	// recompunha a cadeia a partir de um handler que já continha "base",
	// fazendo os middlewares globais/do grupo rodarem duas vezes. rawHandler
	// precisa ser sempre o handler ORIGINAL, sem nenhum middleware aplicado.
	n.rawHandler = h

	registerAutoOptions(a.router, method, path, base)
	return &Route{node: n, router: a.router, base: base}
}

/**
 * registerAutoOptions garante que exista um handler OPTIONS para path,
 * passando pela mesma cadeia de middlewares (base) das outras rotas
 * registradas nesse caminho.
 *
 * BUG CORRIGIDO: sem isso, uma requisição de preflight CORS (método
 * OPTIONS) nunca batia em nenhum node da árvore de rotas — só existiam
 * nodes para GET/POST/etc — e o Router.Match devolvia ok=false direto
 * para App.NotFoundHandler, sem passar por NENHUM middleware. Ou seja,
 * kite.CORS() registrado via app.Use nunca era executado num preflight
 * real de browser. Este auto-registro faz o OPTIONS cair na mesma cadeia
 * de middlewares, permitindo que CORS() responda o preflight normalmente
 * (ele mesmo decide o que fazer quando req.Method == OPTIONS). Se nenhum
 * middleware tratar o OPTIONS, o handler default aqui só responde 204.
 *
 * @param router *Router
 * @param method string - Método HTTP da rota que disparou o registro (GET, POST, ...).
 * @param path string - Path completo já resolvido (com prefixo de grupo, se houver).
 * @param base []MiddlewareFunc - Middlewares (globais e/ou de grupo) a aplicar no OPTIONS.
 */
func registerAutoOptions(router *Router, method, path string, base []MiddlewareFunc) {
	if method == http.MethodOptions {
		return
	}
	optionsHandler := func(req Request, res Response) error {
		return res.NoContent()
	}
	router.Add(http.MethodOptions, path, chain(optionsHandler, base))
}

/**
 * URLFor gera a URL de uma rota nomeada previamente com .Name(...). Veja
 * Router.URLFor para detalhes sobre parâmetros de rota e query string extra.
 *
 * @param {string} name - Nome atribuído via Route.Name.
 * @param {map[string]string} params - Valores dos parâmetros de rota (e extras).
 * @return {(string, error)}
 */
func (a *App) URLFor(name string, params map[string]string) (string, error) {
	return a.router.URLFor(name, params)
}

/**
 * LoadViews configura o motor de templates da aplicação: cria um
 * template.Engine apontando para dir (extensão ext, ex.: ".html"), carrega
 * todas as views encontradas e guarda o engine no App, para que
 * res.Render(name, data) funcione em qualquer handler sem precisar receber
 * o engine manualmente.
 *
 *	app := kite.New()
 *	if err := app.LoadViews("./views", ".html"); err != nil {
 *	    log.Fatal(err)
 *	}
 *	...
 *	app.Get("/", func(req kite.Request, res kite.Response) error {
 *	    return res.Render("index", kite.Map{"Title": "Hello, World!"})
 *	})
 *
 * @param {string} dir - Diretório raiz das views.
 * @param {string} ext - Extensão dos arquivos de view (ex.: ".html").
 * @return {error}
 */
func (a *App) LoadViews(dir, ext string) error {
	engine := template.New(dir, ext)
	if err := engine.Load(); err != nil {
		return fmt.Errorf("kite: falha ao carregar views de %s: %w", dir, err)
	}
	a.views = engine
	return nil
}

/**
 * Views devolve o motor de templates configurado via LoadViews (ou nil, se
 * ainda não tiver sido chamado) — útil para passar o engine adiante
 * manualmente (ex.: para RenderWith) ou para registrar funções extras com
 * engine.AddFunc antes de recarregar as views em modo Debug.
 *
 * @return {*template.Engine}
 */
func (a *App) Views() *template.Engine {
	return a.views
}

/**
 * Group initializes a logical route grouping scoped under a common prefix path.
 *
 * @param {string} prefix - The root segment to prepend to group routes.
 * @return {*RouteGroup} Configured route group instance.
 */
func (a *App) Group(prefix string) *RouteGroup {
	inherited := make([]MiddlewareFunc, len(a.middlewares))
	copy(inherited, a.middlewares)
	return &RouteGroup{app: a, prefix: prefix, middlewares: inherited}
}

/**
 * registerDocs registers the built-in OpenAPI spec and Scalar UI routes.
 * Called synchronously by New() — never inside ServeHTTP — ensuring routes
 * are added exactly once before any request is served, with no need for
 * a mutex or sync.Once.
 */
func (a *App) registerDocs() {
	openAPIURL := a.config.OpenAPIURL
	docsURL := a.config.DocsURL

	// Route that serves the OpenAPI 3.1 specification as JSON
	a.Get(openAPIURL, func(req Request, res Response) error {
		gen := NewOpenAPIGenerator(a.config)
		spec := gen.Generate(a.router.Routes())
		return res.Json(spec)
	}).Hidden()

	// Route that serves the Scalar API Reference interactive UI
	a.Get(docsURL, func(req Request, res Response) error {
		html, err := RenderScalarHTML(ScalarConfig{
			Title:   a.config.Title + " - API Reference",
			SpecURL: openAPIURL,
			Theme:   a.config.ScalarTheme,
			Layout:  a.config.ScalarLayout,
		})
		if err != nil {
			return res.Status(http.StatusInternalServerError).Send("Failed to render documentation")
		}
		return res.WithHtml(html)
	}).Hidden()
}

/**
 * ServeHTTP satisfies the standard net/http Handler interface for dispatching requests.
 * Repassa o engine de views configurado (a.views, pode ser nil) para a Response
 * gerada, permitindo que res.Render funcione dentro do handler.
 *
 * @param {http.ResponseWriter} w - Native response writer.
 * @param {*http.Request} r - Native HTTP request pointer.
 */
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, params, ok := a.router.Match(r.Method, r.URL.Path)
	req := Request{Request: r, Params: params}
	res := newResponse(w, r, a.views)

	if !ok {
		_ = a.NotFoundHandler(req, res)
		return
	}

	if err := handler(req, res); err != nil {
		a.ErrorHandler(req, res, err)
	}
}

/**
 * Listen binds the application to a network address and starts accepting TCP connections.
 *
 * @param {string} addr - Network address format (e.g. ":8080").
 * @return {error} Returns server error states upon failure.
 */
func (a *App) Listen(addr string) error {
	log.Printf("kite: ouvindo em %s", addr)
	return http.ListenAndServe(addr, a)
}

/**
 * RouteGroup manages scoped route collections and localized middleware inheritance.
 */
type RouteGroup struct {
	app         *App
	prefix      string
	middlewares []MiddlewareFunc
}

/**
 * Use appends a middleware scoped exclusively to this route group.
 *
 * @param {MiddlewareFunc} mw - The group middleware function.
 */
func (g *RouteGroup) Use(mw MiddlewareFunc) {
	g.middlewares = append(g.middlewares, mw)
}

/**
 * Get registers a scoped HTTP GET route within the group and returns a *Route,
 * que permite encadear .Middleware(...) e .Name(...) do mesmo jeito que em App.
 *
 * @param {string} path
 * @param {HandlerFunc} h
 * @return {*Route}
 */
func (g *RouteGroup) Get(path string, h HandlerFunc) *Route {
	return g.register(http.MethodGet, path, h)
}

/**
 * Post registers a scoped HTTP POST route within the group. See Get for chaining.
 *
 * @param {string} path
 * @param {HandlerFunc} h
 * @return {*Route}
 */
func (g *RouteGroup) Post(path string, h HandlerFunc) *Route {
	return g.register(http.MethodPost, path, h)
}

/**
 * Put registers a scoped HTTP PUT route within the group. See Get for chaining.
 *
 * @param {string} path
 * @param {HandlerFunc} h
 * @return {*Route}
 */
func (g *RouteGroup) Put(path string, h HandlerFunc) *Route {
	return g.register(http.MethodPut, path, h)
}

/**
 * Delete registers a scoped HTTP DELETE route within the group. See Get for chaining.
 *
 * @param {string} path
 * @param {HandlerFunc} h
 * @return {*Route}
 */
func (g *RouteGroup) Delete(path string, h HandlerFunc) *Route {
	return g.register(http.MethodDelete, path, h)
}

/**
 * Patch registers a scoped HTTP PATCH route within the group. See Get for chaining.
 *
 * @param {string} path
 * @param {HandlerFunc} h
 * @return {*Route}
 */
func (g *RouteGroup) Patch(path string, h HandlerFunc) *Route {
	return g.register(http.MethodPatch, path, h)
}

/**
 * register internally evaluates group prefixes and registers the composed handler
 * chain (middlewares globais do App não entram aqui — eles já foram herdados no
 * momento de a.Group() copiar a.middlewares para dentro do RouteGroup).
 *
 * @param {string} method - Target HTTP method.
 * @param {string} path - Route suffix path.
 * @param {HandlerFunc} h - Original (unwrapped) handler.
 * @return {*Route}
 */
func (g *RouteGroup) register(method, path string, h HandlerFunc) *Route {
	full := joinPrefix(g.prefix, path)

	base := make([]MiddlewareFunc, len(g.middlewares))
	copy(base, g.middlewares)

	n := g.app.router.Add(method, full, chain(h, base))
	// Mesma correção de App.register: rawHandler tem que ser o handler
	// original (h), não o já encadeado com "base" — senão Route.Middleware
	// duplica os middlewares do grupo ao recompor a cadeia.
	n.rawHandler = h

	registerAutoOptions(g.app.router, method, full, base)
	return &Route{node: n, router: g.app.router, base: base}
}

/**
 * joinPrefix safely concatenates a group prefix and a route path string.
 *
 * @param {string} prefix - The parent prefix string.
 * @param {string} path - The target child path.
 * @return {string} The fully joined unified path string.
 */
func joinPrefix(prefix, path string) string {
	if prefix == "" {
		return path
	}
	if path == "/" {
		return prefix
	}
	return prefix + path
}

/**
 * Route é devolvido por App.Get/Post/Put/Delete/Patch e pelos equivalentes de
 * RouteGroup. Permite anexar middlewares específicos da rota e/ou nomeá-la para
 * reverse routing, de forma encadeável:
 *
 *	app.Get("/users/:id", showUser).
 *	    Middleware(auth.RequireAuth, RateLimit(60)).
 *	    Name("users.show")
 *
 *	app.Group("/api").
 *	    Post("/orders", createOrder).
 *	    Middleware(auth.RequireAuth).
 *	    Name("orders.create")
 */
type Route struct {
	node   *node
	router *Router
	base   []MiddlewareFunc
}

/**
 * Middleware anexa um ou mais middlewares específicos desta rota. Eles executam
 * depois dos middlewares globais/do grupo e antes do handler final — ou seja, são
 * os mais "internos" da cadeia. Pode ser chamado múltiplas vezes; cada chamada
 * acrescenta à lista já existente e recompõe a cadeia final a partir do handler
 * original (rawHandler), evitando aplicar os middlewares duas vezes.
 *
 * @param {...MiddlewareFunc} mw - Middlewares exclusivos desta rota.
 * @return {*Route}
 */
func (rt *Route) Middleware(mw ...MiddlewareFunc) *Route {
	rt.node.middlewares = append(rt.node.middlewares, mw...)

	full := make([]MiddlewareFunc, 0, len(rt.base)+len(rt.node.middlewares))
	full = append(full, rt.base...)
	full = append(full, rt.node.middlewares...)

	rt.node.handler = chain(rt.node.rawHandler, full)
	return rt
}

/**
 * Name atribui um identificador único a esta rota, permitindo gerar sua URL
 * depois via app.URLFor(name, params) — equivalente ao ->name() + route() do
 * Laravel. Nomear novamente uma rota substitui a entrada anterior no índice.
 *
 * @param {string} name - Identificador da rota, ex.: "users.show".
 * @return {*Route}
 */
func (rt *Route) Name(name string) *Route {
	rt.node.name = name
	rt.router.setName(name, rt.node)
	return rt
}

func (rt *Route) ensureDoc() *RouteDoc {
	if rt.node.doc == nil {
		rt.node.doc = &RouteDoc{
			Responses: make(map[int]*ResponseDoc),
		}
	}
	if rt.node.doc.Responses == nil {
		rt.node.doc.Responses = make(map[int]*ResponseDoc)
	}
	return rt.node.doc
}

/**
 * Summary define um resumo curto da operação na documentação OpenAPI.
 */
func (rt *Route) Summary(summary string) *Route {
	rt.ensureDoc().Summary = summary
	return rt
}

/**
 * Description define uma descrição detalhada da operação na documentação OpenAPI.
 */
func (rt *Route) Description(desc string) *Route {
	rt.ensureDoc().Description = desc
	return rt
}

/**
 * Tags categoriza a rota em tags/grupos na documentação OpenAPI e no Scalar.
 */
func (rt *Route) Tags(tags ...string) *Route {
	rt.ensureDoc().Tags = append(rt.ensureDoc().Tags, tags...)
	return rt
}

/**
 * Deprecated marca a rota como obsoleta/descontinuada na documentação.
 */
func (rt *Route) Deprecated() *Route {
	rt.ensureDoc().Deprecated = true
	return rt
}

/**
 * Hidden oculta a rota da documentação OpenAPI e da interface do Scalar.
 */
func (rt *Route) Hidden() *Route {
	rt.ensureDoc().Hidden = true
	return rt
}

/**
 * Body define a estrutura e schema do corpo esperado na requisição HTTP (application/json).
 */
func (rt *Route) Body(model any, description ...string) *Route {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	rt.ensureDoc().RequestBody = &BodyDoc{
		Model:       model,
		Description: desc,
		Required:    true,
	}
	return rt
}

/**
 * Query documenta um parâmetro de query string para a rota.
 */
func (rt *Route) Query(name string, model any, description ...string) *Route {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	rt.ensureDoc().Parameters = append(rt.ensureDoc().Parameters, ParamDoc{
		In:          "query",
		Name:        name,
		Model:       model,
		Description: desc,
	})
	return rt
}

/**
 * Header documenta um cabeçalho HTTP esperado na requisição da rota.
 */
func (rt *Route) Header(name string, model any, description ...string) *Route {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	rt.ensureDoc().Parameters = append(rt.ensureDoc().Parameters, ParamDoc{
		In:          "header",
		Name:        name,
		Model:       model,
		Description: desc,
	})
	return rt
}

/**
 * Response documenta uma resposta com código HTTP e estrutura de dados retornada.
 */
func (rt *Route) Response(statusCode int, model any, description ...string) *Route {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	rt.ensureDoc().Responses[statusCode] = &ResponseDoc{
		Model:       model,
		Description: desc,
	}
	return rt
}

/**
 * Doc substitui todos os metadados de documentação da rota por um RouteDoc customizado.
 */
func (rt *Route) Doc(doc RouteDoc) *Route {
	current := rt.ensureDoc()
	*current = doc
	return rt
}