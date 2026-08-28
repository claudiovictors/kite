package kite

import "strings"

/**
 * HandlerFunc define a assinatura padrão para os manipuladores de requisição do Kite.
 *
 * Recebe o objeto Request e o Response encadeável, retornando um erro caso ocorra.
 */
type HandlerFunc func(req Request, res Response) error

/**
 * node representa um nó (segmento de path) na árvore Trie de roteamento HTTP do Kite.
 */
type node struct {
	segment  string
	children []*node
	handler  HandlerFunc
	isParam  bool
	isWild   bool
	paramKey string
}

/**
 * Router mapeia métodos HTTP (GET, POST, PUT, DELETE, etc.) para suas respectivas árvores de rotas.
 */
type Router struct {
	trees map[string]*node
}

/**
 * newRouter inicializa e retorna uma nova instância do roteador do Kite.
 *
 * @return *Router
 */
func newRouter() *Router {
	return &Router{trees: make(map[string]*node)}
}

/**
 * Add registra uma rota na árvore Trie associada ao método HTTP informado.
 *
 * Suporta segmentos estáticos ("users"), parâmetros nomeados (":id") e wildcards ("*").
 *
 * @param method string
 * @param path string
 * @param handler HandlerFunc
 */
func (r *Router) Add(method, path string, handler HandlerFunc) {
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