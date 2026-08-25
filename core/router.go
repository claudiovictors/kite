package kite

import "strings"

// HandlerFunc é a assinatura de qualquer handler de rota.
type HandlerFunc func(req Request, res Response) error

// node representa um segmento de path na árvore de rotas.
// Ex: para "/users/:id/posts", cada "/" separa um node.
type node struct {
	segment  string // literal ("users"), ":id" (param) ou "*" (wildcard)
	children []*node
	handler  HandlerFunc
	isParam  bool
	isWild   bool
	paramKey string
}

// Router mantém uma árvore por método HTTP (GET, POST, ...).
type Router struct {
	trees map[string]*node
}

func newRouter() *Router {
	return &Router{trees: make(map[string]*node)}
}

// Add registra um handler para um método + path.
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

// Match procura o handler correspondente ao método + path, retornando
// também os parâmetros de rota extraídos.
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
		// 1) match literal exato tem prioridade
		for _, c := range current.children {
			if !c.isParam && !c.isWild && c.segment == seg {
				next = c
				break
			}
		}
		// 2) senão, tenta parâmetro (:id)
		if next == nil {
			for _, c := range current.children {
				if c.isParam {
					next = c
					params[c.paramKey] = seg
					break
				}
			}
		}
		// 3) senão, tenta wildcard (*)
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

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}
