package kite

import (
	"fmt"
	"net/url"
	"strings"
)

/**
 * HandlerFunc define a assinatura padrão para os manipuladores de requisição do Kite.
 *
 * Recebe o objeto Request e o Response encadeável, retornando um erro caso ocorra.
 */
type HandlerFunc func(req Request, res Response) error

/**
 * node representa um nó (segmento de path) na árvore Trie de roteamento HTTP do Kite.
 *
 * Além do roteamento em si, cada nó-folha (com handler definido) guarda os metadados
 * necessários para suportar middlewares por rota e rotas nomeadas: rawHandler (o
 * handler original, sem middlewares aplicados), middlewares (a lista específica
 * dessa rota), method/fullPath (usados no reverse routing) e name (identificador
 * usado em Route.Name / Router.URLFor).
 */
type node struct {
	segment  string
	children []*node
	handler  HandlerFunc
	isParam  bool
	isWild   bool
	paramKey string

	rawHandler  HandlerFunc
	middlewares []MiddlewareFunc
	method      string
	fullPath    string
	name        string
	doc         *RouteDoc
}

/**
 * Router mapeia métodos HTTP (GET, POST, PUT, DELETE, etc.) para suas respectivas
 * árvores de rotas, além de manter um índice de rotas nomeadas para reverse routing.
 */
type Router struct {
	trees map[string]*node
	names map[string]*node
}

/**
 * newRouter inicializa e retorna uma nova instância do roteador do Kite.
 *
 * @return *Router
 */
func newRouter() *Router {
	return &Router{trees: make(map[string]*node), names: make(map[string]*node)}
}

/**
 * Add registra uma rota na árvore Trie associada ao método HTTP informado e devolve
 * o nó-folha correspondente, para que App/RouteGroup possam anexar middlewares por
 * rota e um nome através do *Route retornado por eles.
 *
 * Suporta segmentos estáticos ("users"), parâmetros nomeados (":id") e wildcards ("*").
 *
 * @param method string
 * @param path string
 * @param handler HandlerFunc
 * @return *node
 */
func (r *Router) Add(method, path string, handler HandlerFunc) *node {
	root, ok := r.trees[method]
	if !ok {
		root = &node{segment: "/"}
		r.trees[method] = root
	}

	segments := splitPath(path)
	current := root

	for _, seg := range segments {
		var child *node
		for _, c := range current.children {
			if c.segment == seg {
				child = c
				break
			}
		}
		if child == nil {
			child = &node{segment: seg}
			if strings.HasPrefix(seg, ":") {
				child.isParam = true
				child.paramKey = seg[1:]
			} else if seg == "*" {
				child.isWild = true
			}
			current.children = append(current.children, child)
		}
		current = child
	}

	current.handler = handler
	current.rawHandler = handler
	current.method = method
	current.fullPath = path
	return current
}

/**
 * Match percorre a árvore Trie correspondente ao método HTTP e resolve a rota para a URL fornecida.
 *
 * Retorna o HandlerFunc registrado, um mapa com os parâmetros extraídos da URL (ex: ":id") e um booleano de confirmação.
 * Ordem de precedência nos nós: 1) Literal estático -> 2) Parâmetro (:key) -> 3) Wildcard (*).
 *
 * @param method string
 * @param path string
 * @return (HandlerFunc, map[string]string, bool)
 */
func (r *Router) Match(method, path string) (HandlerFunc, map[string]string, bool) {
	root, ok := r.trees[method]
	if !ok {
		return nil, nil, false
	}

	segments := splitPath(path)
	params := make(map[string]string)

	current := root
	for _, seg := range segments {
		var next *node
		// 1) Match literal exato tem prioridade
		for _, c := range current.children {
			if !c.isParam && !c.isWild && c.segment == seg {
				next = c
				break
			}
		}
		// 2) Tenta parâmetro (:id)
		if next == nil {
			for _, c := range current.children {
				if c.isParam {
					next = c
					params[c.paramKey] = seg
					break
				}
			}
		}
		// 3) Tenta wildcard (*)
		if next == nil {
			for _, c := range current.children {
				if c.isWild {
					next = c
					break
				}
			}
		}
		if next == nil {
			return nil, nil, false
		}
		current = next
	}

	if current.handler == nil {
		return nil, nil, false
	}
	return current.handler, params, true
}

/**
 * splitPath divide a string de caminho enviada em um slice de segmentos limpos.
 *
 * @param path string
 * @return []string
 */
func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

/**
 * setName registra (ou substitui) o nó associado a um nome de rota. Chamado
 * internamente por Route.Name — não use diretamente.
 */
func (r *Router) setName(name string, n *node) {
	r.names[name] = n
}

/**
 * URLFor gera a URL de uma rota nomeada, substituindo os segmentos de parâmetro
 * (ex: ":id") pelos valores informados em params. Parâmetros que não correspondem
 * a nenhum segmento da rota são anexados como query string — equivalente ao que o
 * route() do Laravel faz quando você passa dados extras.
 *
 * Exemplo:
 *
 *	app.Get("/users/:id", showUser).Name("users.show")
 *
 *	url, err := app.URLFor("users.show", map[string]string{"id": "42"})
 *	// url == "/users/42"
 *
 *	url, err = app.URLFor("users.show", map[string]string{"id": "42", "tab": "posts"})
 *	// url == "/users/42?tab=posts"
 *
 * @param name string
 * @param params map[string]string
 * @return (string, error)
 */
func (r *Router) URLFor(name string, params map[string]string) (string, error) {
	n, ok := r.names[name]
	if !ok {
		return "", fmt.Errorf("kite: rota nomeada %q não encontrada", name)
	}

	used := make(map[string]bool, len(params))
	segments := splitPath(n.fullPath)

	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			key := seg[1:]
			val, ok := params[key]
			if !ok {
				return "", fmt.Errorf("kite: parâmetro %q é obrigatório para gerar a URL de %q", key, name)
			}
			segments[i] = val
			used[key] = true
		}
	}

	path := "/" + strings.Join(segments, "/")

	extra := url.Values{}
	for k, v := range params {
		if !used[k] {
			extra.Set(k, v)
		}
	}
	if len(extra) > 0 {
		return path + "?" + extra.Encode(), nil
	}
	return path, nil
}

/**
 * Routes retorna todos os nós de rota registrados no Router com seus handlers e metadados.
 */
func (r *Router) Routes() []*node {
	var list []*node
	for _, root := range r.trees {
		collectNodes(root, &list)
	}
	return list
}

func collectNodes(n *node, list *[]*node) {
	if n == nil {
		return
	}
	if n.handler != nil && n.fullPath != "" {
		*list = append(*list, n)
	}
	for _, child := range n.children {
		collectNodes(child, list)
	}
}