// Package template implementa a engine de views do kite.
//
// V1 (implementada): wrapper sobre html/template, com:
//   - cache de templates parseados (parse único, reuso em cada request)
//   - todos os arquivos são associados a uma única árvore de templates,
//     permitindo {{template "outro/arquivo" .}} / @include entre views
//     diferentes, e layouts via {{define "content"}} / {{template "content" .}}
//   - diretivas estilo Blade (@if, @foreach, {{-- comentário --}},
//     @include) transpiladas em directives.go antes do parse
//   - reload automático em modo dev (Debug=true)
//
// V2 (planejada): lexer/parser dedicado, transpilando para html/template
// ou para um AST próprio (as diretivas atuais são regex, suficientes
// pro dia a dia mas frágeis em casos aninhados/complexos).
package template

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sync"
)

// Engine gerencia o parsing e a renderização de views.
type Engine struct {
	dir   string // diretório raiz das views, ex: "./views"
	ext   string // extensão dos arquivos, ex: ".html"
	Debug bool   // se true, reparseia todas as views a cada Render (hot reload)

	mu        sync.RWMutex
	templates map[string]*template.Template // nome da view -> árvore raiz (todas apontam pro mesmo *template.Template)
	funcs     template.FuncMap
}

// New cria uma engine apontando para o diretório de views informado.
// ext é a extensão dos arquivos (ex: ".html", ".tmpl").
func New(dir, ext string) *Engine {
	return &Engine{
		dir:       dir,
		ext:       ext,
		templates: make(map[string]*template.Template),
		funcs:     template.FuncMap{},
	}
}

// AddFunc registra uma função customizada disponível em todas as views
// (ex: engine.AddFunc("upper", strings.ToUpper)).
func (e *Engine) AddFunc(name string, fn interface{}) {
	e.funcs[name] = fn
}

// Load faz o parsing de todos os templates do diretório configurado.
// Diferente da versão anterior (que parseava cada arquivo isoladamente
// com ParseFiles), agora todos os arquivos são associados a uma única
// árvore de templates nomeados — isso é o que permite {{template "x" .}}
// e @include referenciarem views de arquivos diferentes.
//
// Deve ser chamado uma vez no boot da aplicação (a menos que Debug=true,
// que reparseia sob demanda a cada Render).
func (e *Engine) Load() error {
	pattern := filepath.Join(e.dir, "**", "*"+e.ext)
	matches, err := doubleStarGlob(e.dir, e.ext)
	if err != nil {
		return fmt.Errorf("template: falha ao listar views em %s: %w", e.dir, err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("template: nenhuma view encontrada em %s (pattern %s)", e.dir, pattern)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	root := template.New("root").Funcs(e.funcs)
	names := make(map[string]struct{}, len(matches))

	for _, file := range matches {
		name := e.templateName(file)
		names[name] = struct{}{}

		raw, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("template: erro ao ler %s: %w", file, err)
		}

		// Transpila as diretivas estilo Blade (@if, @foreach, {{-- --}},
		// @include) pra sintaxe nativa do html/template antes de parsear.
		compiled := compileDirectives(string(raw))

		root, err = root.New(name).Parse(compiled)
		if err != nil {
			return fmt.Errorf("template: erro ao parsear %s: %w", file, err)
		}
	}

	templates := make(map[string]*template.Template, len(names))
	for name := range names {
		templates[name] = root
	}
	e.templates = templates

	return nil
}

// Render renderiza a view identificada por name (path relativo ao dir,
// sem extensão, ex: "users/show") com os dados informados, escrevendo
// o resultado em w.
func (e *Engine) Render(w *bytes.Buffer, name string, data interface{}) error {
	if e.Debug {
		if err := e.Load(); err != nil {
			return err
		}
	}

	e.mu.RLock()
	root, ok := e.templates[name]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("template: view %q não encontrada (esqueceu de chamar engine.Load()?)", name)
	}

	return root.ExecuteTemplate(w, name, data)
}

// RenderToString é um atalho de Render que retorna a view já renderizada
// como string, útil para passar direto a res.WithHtml(...).
func (e *Engine) RenderToString(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := e.Render(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *Engine) templateName(file string) string {
	rel, _ := filepath.Rel(e.dir, file)
	rel = rel[:len(rel)-len(e.ext)]
	return filepath.ToSlash(rel)
}

// doubleStarGlob varre recursivamente o diretório procurando arquivos
// com a extensão informada (Go não suporta ** nativo em filepath.Glob).
func doubleStarGlob(root, ext string) ([]string, error) {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ext {
			matches = append(matches, path)
		}
		return nil
	})
	return matches, err
}