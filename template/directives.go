package template

import "regexp"

/**
 * Expressões regulares compiladas para transpilado de diretivas estilo Blade em sintaxe html/template.
 */
var (
	commentDirective       = regexp.MustCompile(`(?s)\{\{--.*?--\}\}`)
	ifDirective            = regexp.MustCompile(`@if\s*\((.+?)\)`)
	elseifDirective        = regexp.MustCompile(`@elseif\s*\((.+?)\)`)
	elseDirective          = regexp.MustCompile(`@else\b`)
	endifDirective         = regexp.MustCompile(`@endif\b`)
	foreachAssignDirective = regexp.MustCompile(`@foreach\s*\(\s*(\$\w+)\s+in\s+(.+?)\)`)
	endforeachDirective    = regexp.MustCompile(`@endforeach\b`)
	includeDirective       = regexp.MustCompile(`@include\s*\(\s*"(.+?)"\s*\)`)
)

/**
 * compileDirectives transpila a sintaxe amigável estilo Blade para a sintaxe nativa do html/template antes do parsing.
 *
 * A ordem das substituições é crítica: `foreachAssignDirective` é processada antes de `foreachDirective`
 * para evitar que a palavra reservada "in" seja capturada erroneamente como parte da expressão.
 *
 * Exemplo de transformação:
 *  @if(condicao)                       -> {{if condicao}}
 *  @foreach($post in .Posts)           -> {{range $post := .Posts}}
 *  @include("partials/header")         -> {{template "partials/header" .}}
 *
 * @param src string conteúdo bruto do arquivo de template.
 * @return string conteúdo convertido para sintaxe html/template.
 */
func compileDirectives(src string) string {
	src = commentDirective.ReplaceAllString(src, "")
	src = ifDirective.ReplaceAllString(src, `{{if $1}}`)
	src = elseifDirective.ReplaceAllString(src, `{{else if $1}}`)
	src = elseDirective.ReplaceAllString(src, `{{else}}`)
	src = endifDirective.ReplaceAllString(src, `{{end}}`)
	src = foreachAssignDirective.ReplaceAllString(src, `{{range $1 := $2}}`)
	src = endforeachDirective.ReplaceAllString(src, `{{end}}`)
	src = includeDirective.ReplaceAllString(src, `{{template "$1" .}}`)
	return src
}