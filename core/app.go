package kite

import (
    "log"
    "net/http"
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
 * App represents the core engine of the kite package, managing the router,
 * global middlewares, and default lifecycle error/not-found handlers.
 */
type App struct {
    router      *Router
    middlewares []MiddlewareFunc

    NotFoundHandler HandlerFunc
    ErrorHandler    func(req Request, res Response, err error)
}

/**
 * New instantiates and returns a new core App application instance
 * with a pre-configured router and default handlers.
 * 
 * @return {*App} Pointer to the newly created App instance.
 */
func New() *App {
    app := &App{router: newRouter()}
    app.NotFoundHandler = defaultNotFoundHandler
    app.ErrorHandler = defaultErrorHandler
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
 * Get registers a route handling HTTP GET requests.
 * 
 * @param {string} path - The target route pattern.
 * @param {HandlerFunc} h - The designated handler function.
 */
func (a *App) Get(path string, h HandlerFunc)    { a.register(http.MethodGet, path, h) }

/**
 * Post registers a route handling HTTP POST requests.
 * 
 * @param {string} path - The target route pattern.
 * @param {HandlerFunc} h - The designated handler function.
 */
func (a *App) Post(path string, h HandlerFunc)   { a.register(http.MethodPost, path, h) }

/**
 * Put registers a route handling HTTP PUT requests.
 * 
 * @param {string} path - The target route pattern.
 * @param {HandlerFunc} h - The designated handler function.
 */
func (a *App) Put(path string, h HandlerFunc)    { a.register(http.MethodPut, path, h) }

/**
 * Delete registers a route handling HTTP DELETE requests.
 * 
 * @param {string} path - The target route pattern.
 * @param {HandlerFunc} h - The designated handler function.
 */
func (a *App) Delete(path string, h HandlerFunc) { a.register(http.MethodDelete, path, h) }

/**
 * Patch registers a route handling HTTP PATCH requests.
 * 
 * @param {string} path - The target route pattern.
 * @param {HandlerFunc} h - The designated handler function.
 */
func (a *App) Patch(path string, h HandlerFunc)  { a.register(http.MethodPatch, path, h) }

/**
 * register internally binds an HTTP method and path combination with its middleware chain.
 * 
 * @param {string} method - The HTTP verb.
 * @param {string} path - The route endpoint path.
 * @param {HandlerFunc} h - The final executable handler chain.
 */
func (a *App) register(method, path string, h HandlerFunc) {
    a.router.Add(method, path, chain(h, a.middlewares))
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
 * ServeHTTP satisfies the standard net/http Handler interface for dispatching requests.
 * 
 * @param {http.ResponseWriter} w - Native response writer.
 * @param {*http.Request} r - Native HTTP request pointer.
 */
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    handler, params, ok := a.router.Match(r.Method, r.URL.Path)
    req := Request{Request: r, Params: params}
    res := newResponse(w, r)

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
 * Get registers a scoped HTTP GET route within the group.
 * 
 * @param {string} path - Relative route pattern.
 * @param {HandlerFunc} h - Group route handler function.
 */
func (g *RouteGroup) Get(path string, h HandlerFunc)    { g.register(http.MethodGet, path, h) }

/**
 * Post registers a scoped HTTP POST route within the group.
 * 
 * @param {string} path - Relative route pattern.
 * @param {HandlerFunc} h - Group route handler function.
 */
func (g *RouteGroup) Post(path string, h HandlerFunc)   { g.register(http.MethodPost, path, h) }

/**
 * Put registers a scoped HTTP PUT route within the group.
 * 
 * @param {string} path - Relative route pattern.
 * @param {HandlerFunc} h - Group route handler function.
 */
func (g *RouteGroup) Put(path string, h HandlerFunc)    { g.register(http.MethodPut, path, h) }

/**
 * Delete registers a scoped HTTP DELETE route within the group.
 * 
 * @param {string} path - Relative route pattern.
 * @param {HandlerFunc} h - Group route handler function.
 */
func (g *RouteGroup) Delete(path string, h HandlerFunc) { g.register(http.MethodDelete, path, h) }

/**
 * Patch registers a scoped HTTP PATCH route within the group.
 * 
 * @param {string} path - Relative route pattern.
 * @param {HandlerFunc} h - Group route handler function.
 */
func (g *RouteGroup) Patch(path string, h HandlerFunc)  { g.register(http.MethodPatch, path, h) }

/**
 * register internally evaluates group prefixes and registers the composed handler chain.
 * 
 * @param {string} method - Target HTTP method.
 * @param {string} path - Route suffix path.
 * @param {HandlerFunc} h - Target execution handler.
 */
func (g *RouteGroup) register(method, path string, h HandlerFunc) {
    full := joinPrefix(g.prefix, path)
    g.app.router.Add(method, full, chain(h, g.middlewares))
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