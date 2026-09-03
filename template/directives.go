package template

import "regexp"

/**
 * Compiled regular expressions for transpiling Blade-style directives into html/template syntax.
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
 * compileDirectives transpiles the friendly Blade-style syntax to the native html/template syntax before parsing.
 *
 * Transformation example:
 *  @if(condition)                      -> {{if condition}}
 *  @foreach($post in .Posts)           -> {{range $post := .Posts}}
 *  @include("partials/header")         -> {{template "partials/header" .}}
 *
 * @param src string raw content of the template file.
 * @return string content converted to html/template syntax.
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