// Package template implements the framework's view engine.
//
// V1 (implemented): wrapper around html/template, with:
//   - parsed templates cache (single parse, reused in every request)
//   - all files are associated with a single template tree,
//     allowing {{template "other/file" .}} / @include between different
//     views, and layouts via {{define "content"}} / {{template "content" .}}
//   - Blade-style directives (@if, @foreach, {{-- comment --}},
//     @include) transpiled in directives.go before parsing
//   - automatic reload in dev mode (Debug=true)
//
// V2 (planned): dedicated lexer/parser, transpiling to html/template
// or to a custom AST (current directives are regex, sufficient
// for daily use but fragile in nested/complex cases).
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
 * Engine manages the lifecycle, compilation, memory caching, and rendering of views.
 */
type Engine struct {
	dir   string // Root directory of the views, e.g.: "./views"
	ext   string // Template files extension, e.g.: ".html"
	Debug bool   // When true, forces re-reading and compiling all views on every Render (Hot Reload)

	mu        sync.RWMutex
	templates map[string]*template.Template // Mapping view name -> shared template tree
	funcs     template.FuncMap
}

/**
 * New instantiates a new Engine pointing to the given views directory.
 *
 * Example:
 *  engine := template.New("./views", ".html")
 *
 * @param dir string path to the directory containing the templates.
 * @param ext string template files extension.
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
 * AddFunc registers a custom helper function to be globally available in the views.
 *
 * Example:
 *  engine.AddFunc("upper", strings.ToUpper)
 *
 * @param name string name used to invoke the function inside the template.
 * @param fn interface{} function to be registered.
 */
func (e *Engine) AddFunc(name string, fn interface{}) {
	e.funcs[name] = fn
}

/**
 * Load reads and parses all files from the views directory into a shared template tree.
 *
 * Transpiles Blade-style directives into native syntax before parsing in html/template.
 * Must be executed at application startup (boot), except when Debug=true.
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

		// Transpiles Blade-style directives (@if, @foreach, {{-- --}},
		// @include) to native html/template syntax before parsing.
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
 * Render executes the template indicated by the name and writes the processed HTML to the buffer w.
 *
 * The name must be the relative path to the views directory without the extension (e.g.: "users/show").
 *
 * Example:
 *  var buf bytes.Buffer
 *  err := engine.Render(&buf, "users/index", data)
 *
 * @param w *bytes.Buffer output buffer.
 * @param name string view name.
 * @param data interface{} data passed for rendering in the view.
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
 * RenderToString is a utility helper that renders the given view and returns the result as a string.
 *
 * Example:
 *  html, err := engine.RenderToString("emails/welcome", user)
 *
 * @param name string view name.
 * @param data interface{} data passed to the view.
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
 * templateName extracts the normalized relative view name from the absolute/relative file path on disk.
 *
 * @param file string full file path.
 * @return string view name formatted with forward slashes (e.g.: "layouts/main").
 */
func (e *Engine) templateName(file string) string {
	rel, _ := filepath.Rel(e.dir, file)
	rel = rel[:len(rel)-len(e.ext)]
	return filepath.ToSlash(rel)
}

/**
 * doubleStarGlob performs a recursive file search in the specified directory, filtering by extension.
 *
 * @param root string root search directory.
 * @param ext string target extension (e.g.: ".html").
 * @return ([]string, error) list with the paths of the found files.
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