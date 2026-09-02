// Package template implementa a engine de views do framework.
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

/**
 * Engine gerencia o ciclo de vida, compilação, cache em memória e renderização das views.
 */
type Engine struct {
	dir   string // Diretório raiz das views, ex: "./views"
	ext   string // Extensão dos arquivos de template, ex: ".html"
	Debug bool   // Quando true, força a re-leitura e compilação de todas as views a cada Render (Hot Reload)

	mu        sync.RWMutex
	templates map[string]*template.Template // Mapeamento nome da view -> árvore de templates compartilhada
	funcs     template.FuncMap
}

/**
 * New instancia um novo Engine apontando para o diretório de views informado.
 *
 * Exemplo:
 *  engine := template.New("./views", ".html")
 *
 * @param dir string caminho do diretório contendo os templates.
 * @param ext string extensão dos arquivos de template.
 * @return *Engine
 */
func New(dir, ext string) *Engine {
	return &Engine{
		dir:       dir,
		ext:       ext,
		templates: make(map[string]*template.Template),
		funcs:     template.FuncMap{},
	}
}

/**
 * AddFunc registra uma função helper customizada para ficar disponível globalmente nas views.
 *
 * Exemplo:
 *  engine.AddFunc("upper", strings.ToUpper)
 *
 * @param name string nome utilizado para invocar a função dentro do template.
 * @param fn interface{} função a ser registrada.
 */
func (e *Engine) AddFunc(name string, fn interface{}) {
	e.funcs[name] = fn
}

/**
 * Load faz a leitura e parsing de todos os arquivos do diretório de views para uma árvore de templates compartilhada.
 *
 * Transpila as diretivas estilo Blade em sintaxe nativa antes de realizar o parse no html/template.
 * Deve ser executado na inicialização da aplicação (boot), exceto quando Debug=true.
 *
 * @return error
 */
func (e *Engine) Load() error {
	matches, err := doubleStarGlob(e.dir, e.ext)
	if err != nil {
		return fmt.Errorf("template: falha ao listar views em %s: %w", e.dir, err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("template: nenhuma view encontrada em %s (extensão %q)", e.dir, e.ext)
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

/**
 * Render executa o template indicado pelo nome e escreve o HTML processado no buffer w.
 *
 * O nome deve ser o caminho relativo ao diretório de views sem a extensão (ex: "users/show").
 *
 * Exemplo:
 *  var buf bytes.Buffer
 *  err := engine.Render(&buf, "users/index", data)
 *
 * @param w *bytes.Buffer buffer de saída.
 * @param name string nome da view.
 * @param data interface{} dados passados para renderização na view.
 * @return error
 */
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

/**
 * RenderToString é um helper utilitário que renderiza a view informada e retorna o resultado como string.
 *
 * Exemplo:
 *  html, err := engine.RenderToString("emails/welcome", user)
 *
 * @param name string nome da view.
 * @param data interface{} dados passados para a view.
 * @return (string, error)
 */
func (e *Engine) RenderToString(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := e.Render(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

/**
 * templateName extrai o nome relativo normalizado da view a partir do caminho absoluto/relativo do arquivo no disco.
 *
 * @param file string caminho completo do arquivo.
 * @return string nome da view formatado com barras normais (ex: "layouts/main").
 */
func (e *Engine) templateName(file string) string {
	rel, _ := filepath.Rel(e.dir, file)
	rel = rel[:len(rel)-len(e.ext)]
	return filepath.ToSlash(rel)
}

/**
 * doubleStarGlob realiza a busca recursiva de arquivos no diretório especificado filtrando pela extensão.
 *
 * @param root string diretório raiz de busca.
 * @param ext string extensão alvo (ex: ".html").
 * @return ([]string, error) lista com o caminho dos arquivos encontrados.
 */
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