package kite

import (
	"net/http"
	"strconv"
	"strings"
)

/**
 * CORSConfig controla o comportamento do middleware CORS.
 * Os zero values não são utilizáveis diretamente — utilize DefaultCORSConfig() para inicialização.
 */
type CORSConfig struct {
	/**
	 * AllowOrigins define as origens permitidas.
	 * Quando definido como "*", libera qualquer origem. Se AllowCredentials for verdadeiro,
	 * a origem da requisição é refletida dinamicamente.
	 */
	AllowOrigins []string

	/**
	 * AllowMethods define os métodos HTTP permitidos durante requisições de preflight.
	 */
	AllowMethods []string

	/**
	 * AllowHeaders define os cabeçalhos HTTP liberados para envio por parte do cliente.
	 */
	AllowHeaders []string

	/**
	 * ExposeHeaders define os cabeçalhos de resposta que ficam acessíveis via JavaScript no navegador.
	 */
	ExposeHeaders []string

	/**
	 * AllowCredentials indica se a requisição pode incluir cookies ou cabeçalhos de autorização em chamadas cross-origin.
	 */
	AllowCredentials bool

	/**
	 * MaxAge especifica o tempo de vida (em segundos) para o cache da resposta de preflight (OPTIONS).
	 */
	MaxAge int
}

/**
 * DefaultCORSConfig retorna a configuração padrão de CORS orientada a ambientes de desenvolvimento.
 *
 * @return CORSConfig
 */
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:  []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders: []string{},
		MaxAge:        86400, // 24h
	}
}

/**
 * CORS cria e retorna o middleware de interceptação e tratamento de políticas CORS da aplicação Kite.
 *
 * @param config ...CORSConfig Configuração opcional de CORS. Se omitida, utiliza DefaultCORSConfig().
 * @return MiddlewareFunc
 */
func CORS(config ...CORSConfig) MiddlewareFunc {
	cfg := DefaultCORSConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	allowMethods := strings.Join(cfg.AllowMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowHeaders, ", ")
	exposeHeaders := strings.Join(cfg.ExposeHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	wildcard := len(cfg.AllowOrigins) == 1 && cfg.AllowOrigins[0] == "*"

	return func(next HandlerFunc) HandlerFunc {
		return func(req Request, res Response) error {
			origin := req.Header("Origin")

			if origin != "" {
				allowedOrigin := ""
				switch {
				case cfg.AllowCredentials:
					if originAllowed(origin, cfg.AllowOrigins) || wildcard {
						allowedOrigin = origin
					}
				case wildcard:
					allowedOrigin = "*"
				case originAllowed(origin, cfg.AllowOrigins):
					allowedOrigin = origin
				}

				if allowedOrigin != "" {
					res.SetHeader("Access-Control-Allow-Origin", allowedOrigin)
					res.Vary("Origin")
					if cfg.AllowCredentials {
						res.SetHeader("Access-Control-Allow-Credentials", "true")
					}
					if exposeHeaders != "" {
						res.SetHeader("Access-Control-Expose-Headers", exposeHeaders)
					}
				}
			}

			if req.Method == http.MethodOptions {
				res.SetHeader("Access-Control-Allow-Methods", allowMethods)
				res.SetHeader("Access-Control-Allow-Headers", allowHeaders)
				res.SetHeader("Access-Control-Max-Age", maxAge)
				return res.Status(http.StatusNoContent).NoContent()
			}

			return next(req, res)
		}
	}
}

/**
 * originAllowed verifica se a origem fornecida corresponde a uma das origens permitidas na lista.
 *
 * @param origin string
 * @param allowed []string
 * @return bool
 */
func originAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == origin || a == "*" {
			return true
		}
	}
	return false
}