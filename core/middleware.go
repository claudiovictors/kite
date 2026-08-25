package kite

// MiddlewareFunc envolve um HandlerFunc, permitindo executar lógica
// antes/depois do handler (auth, logging, cors, etc).
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// chain aplica os middlewares em ordem, de forma que o primeiro
// registrado seja o mais externo (executa primeiro).
func chain(h HandlerFunc, middlewares []MiddlewareFunc) HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
