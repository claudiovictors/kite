package kite

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

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
 * defaultNotFoundHandler handles requests that fail to match any registered route.
 *
 * @param {Request} req - The wrapped incoming HTTP request.
 * @param {Response} res - The wrapped HTTP response writer.
 * @return {error} Returns any write error encountered during response delivery.
 */
func defaultNotFoundHandler(req Request, res Response) error {
	return res.Status(http.StatusNotFound).WithJson(ErrorResponse{
		Error:  "route not found: " + req.Method + " " + req.URL.Path,
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
	log.Printf("kite: handler error %s %s: %v", req.Method, req.URL.Path, err)
	if writeErr := res.Status(http.StatusInternalServerError).WithJson(ErrorResponse{
		Error:  "internal server error",
		Status: http.StatusInternalServerError,
	}); writeErr != nil {
		log.Printf("kite: failed to write error response: %v", writeErr)
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
 * Get registers a route handling HTTP GET requests and returns a *Route that allows
 * chaining .Middleware(...) (route-specific) and .Name(...) (for reverse routing).
 *
 * Example:
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
 * global middleware chain, storing the original handler (rawHandler) and the base
 * middleware list on the underlying *node — necessary so Route.Middleware can
 * re-compose the chain later including route-specific middlewares.
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
	// FIXED BUG: Router.Add used to store rawHandler as the handler it received
	// (already chained with base). Without this assignment, Route.Middleware
	// would recompute the chain from a handler that already contained "base",
	// causing global/group middlewares to run twice. rawHandler must always be
	// the ORIGINAL handler, without any middleware applied.
	n.rawHandler = h

	registerAutoOptions(a.router, method, path, base)
	return &Route{node: n, router: a.router, base: base}
}

/**
 * registerAutoOptions ensures there is an OPTIONS handler for the given path,
 * passing through the same middleware chain (base) as other routes registered
 * for that path.
 *
 * FIXED BUG: without this, CORS preflight requests (OPTIONS) never matched any
 * node in the routing tree — only GET/POST/etc nodes existed — and Router.Match
 * returned ok=false directly to App.NotFoundHandler without going through ANY
 * middleware. That meant app.Use(kite.CORS()) never ran on real browser preflights.
 * This auto-registration makes OPTIONS go through the same middleware chain so
 * CORS() can respond to the preflight normally (it decides what to do when
 * req.Method == OPTIONS). If no middleware handles OPTIONS, the default handler
 * here responds with 204.
 *
 * @param router *Router
 * @param method string - HTTP method of the route that triggered registration (GET, POST, ...).
 * @param path string - Fully resolved path (with group prefix if present).
 * @param base []MiddlewareFunc - Middlewares (global and/or group) to apply to OPTIONS.
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
 * URLFor generates the URL for a route previously named with .Name(...).
 * See Router.URLFor for details about route parameters and extra query strings.
 *
 * @param {string} name - Name assigned via Route.Name.
 * @param {map[string]string} params - Values for route parameters (and extras).
 * @return {(string, error)}
 */
func (a *App) URLFor(name string, params map[string]string) (string, error) {
	return a.router.URLFor(name, params)
}

/**
 * LoadViews configures the application's view engine: creates a template.Engine
 * pointed at dir (ext extension, e.g. ".html"), loads all found views and stores
 * the engine on the App so res.Render(name, data) works in any handler without
 * passing the engine manually.
 *
 * Example:
 *	app := kite.New()
 *	if err := app.LoadViews("./views", ".html"); err != nil {
 *	    log.Fatal(err)
 *	}
 *	...
 *	app.Get("/", func(req kite.Request, res kite.Response) error {
 *	    return res.Render("index", kite.Map{"Title": "Hello, World!"})
 *	})
 *
 * @param {string} dir - Root directory for views.
 * @param {string} ext - File extension for view files (e.g. ".html").
 * @return {error}
 */
func (a *App) LoadViews(dir, ext string) error {
	engine := template.New(dir, ext)
	if err := engine.Load(); err != nil {
		return fmt.Errorf("kite: failed to load views from %s: %w", dir, err)
	}
	a.views = engine
	return nil
}

/**
 * Views returns the template engine configured via LoadViews (or nil if not set)
 * — useful to pass the engine onward manually (eg: to RenderWith) or to register
 * extra functions with engine.AddFunc before reloading views in Debug mode.
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
 * It passes the configured view engine (a.views, may be nil) to the generated Response
 * so that res.Render works inside handlers.
 *
 * It also recovers from any panic raised while matching or executing a handler
 * (nil pointer dereference, index out of range, etc). The panic is logged with
 * its full stack trace, and the client receives a generic 500 JSON response
 * instead of the connection being dropped and the whole process crashing.
 *
 * @param {http.ResponseWriter} w - Native response writer.
 * @param {*http.Request} r - Native HTTP request pointer.
 */
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Deferred at the very top of the request lifecycle so it catches panics
	// raised anywhere downstream: inside the matched handler, inside any
	// middleware wrapping it, or even inside NotFoundHandler/ErrorHandler
	// themselves if they were customized incorrectly.
	defer func() {
		if err := recover(); err != nil {
			log.Printf("kite: recovered panic in %s %s: %v\n%s", r.Method, r.URL.Path, err, debug.Stack())
			http.Error(w, `{"error":"internal server error","status":500}`, http.StatusInternalServerError)
		}
	}()

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
	log.Printf("Kite server running on port  %s", addr)
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
 * allowing chaining .Middleware(...) and .Name(...) just like on App.
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
 * chain (global App middlewares are not included here — they were already inherited
 * when a.Group() copied a.middlewares into the RouteGroup).
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
	// Same fix as App.register: rawHandler must be the original handler (h),
	// not the one already chained with "base" — otherwise Route.Middleware
	// would duplicate the group's middlewares when recomposing the chain.
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
 * Route is returned by App.Get/Post/Put/Delete/Patch and by the equivalent
 * RouteGroup methods. It allows attaching route-specific middlewares and/or
 * naming the route for reverse routing, in a chainable way:
 *
 * Example:
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
 * Middleware appends one or more middlewares specific to this route. They run
 * after global/group middlewares and before the final handler — i.e. they are
 * the innermost middlewares. This can be called multiple times; each call
 * appends to the existing list and recomposes the final chain from the
 * original handler (rawHandler), avoiding applying middlewares twice.
 *
 * @param {...MiddlewareFunc} mw - Middlewares exclusive to this route.
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
 * Name assigns a unique identifier to this route, allowing its URL to be
 * generated later via app.URLFor(name, params) — equivalent to ->name() + route()
 * in other frameworks. Re-naming a route replaces the previous entry in the index.
 *
 * @param {string} name - Route identifier, e.g.: "users.show".
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
 * Summary sets a short summary for the operation in the OpenAPI documentation.
 */
func (rt *Route) Summary(summary string) *Route {
	rt.ensureDoc().Summary = summary
	return rt
}

/**
 * Description sets a detailed description for the operation in the OpenAPI documentation.
 */
func (rt *Route) Description(desc string) *Route {
	rt.ensureDoc().Description = desc
	return rt
}

/**
 * Tags categorizes the route into tags/groups in the OpenAPI docs and Scalar UI.
 */
func (rt *Route) Tags(tags ...string) *Route {
	rt.ensureDoc().Tags = append(rt.ensureDoc().Tags, tags...)
	return rt
}

/**
 * Deprecated marks the route as deprecated in the documentation.
 */
func (rt *Route) Deprecated() *Route {
	rt.ensureDoc().Deprecated = true
	return rt
}

/**
 * Hidden hides the route from both OpenAPI documentation and the Scalar interface.
 */
func (rt *Route) Hidden() *Route {
	rt.ensureDoc().Hidden = true
	return rt
}

/**
 * Body defines the structure and schema of the expected request body (application/json).
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
 * Query documents a query string parameter for the route.
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
 * Header documents an expected HTTP header for the route's request.
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
 * Response documents a response with an HTTP status code and returned data structure.
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
 * Doc replaces all route documentation metadata with a custom RouteDoc.
 */
func (rt *Route) Doc(doc RouteDoc) *Route {
	current := rt.ensureDoc()
	*current = doc
	return rt
}