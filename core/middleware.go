package kite

/**
 * MiddlewareFunc representa a assinatura para middlewares no Kite.
 * Envolve um HandlerFunc, permitindo executar lógicas antes e depois do manipulador principal.
 */
type MiddlewareFunc func(HandlerFunc) HandlerFunc

/**
 * chain aplica uma lista de middlewares a um HandlerFunc em ordem encadeada.
 * O primeiro middleware registrado será o mais externo (executado primeiro).
 *
 * @param h HandlerFunc
 * @param middlewares []MiddlewareFunc
 * @return HandlerFunc
 */
func chain(h HandlerFunc, middlewares []MiddlewareFunc) HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}