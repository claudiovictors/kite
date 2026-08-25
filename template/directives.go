package template

import "regexp"

// Este arquivo implementa um pré-processador estilo Blade: converte uma
// sintaxe mais legível (@if, @foreach, comentários {{-- --}}, @include)
// para a sintaxe nativa do html/template ({{if}}, {{range}}, {{end}},
// {{template}}) antes do parse. É transpilação via regex — o
// lexer/parser dedicado (AST próprio) continua sendo o objetivo da v2
// do roadmap do Engine.
//
// Diretivas suportadas:
//
//	{{-- comentário --}}                  -> removido do output
//	@if(condicao) ... @endif              -> {{if condicao}} ... {{end}}
//	@if(condicao) ... @else ... @endif    -> {{if condicao}} ... {{else}} ... {{end}}
//	@elseif(condicao)                     -> {{else if condicao}}
//	@foreach(.Items) ... @endforeach      -> {{range .Items}} ... {{end}}
//	@foreach($post in .Posts) ... @endforeach -> {{range $post := .Posts}} ... {{end}}
//	@include("partials/header")           -> {{template "partials/header" .}}
//
// Exemplo de view:
//
//	<ul>
//	{{-- lista de usuários ativos --}}
//	@foreach($user in .Users)
//	    @if($user.Active)
//	        <li>{{ $user.Name }}</li>
//	    @else
//	        <li class="inativo">{{ $user.Name }}</li>
//	    @endif
//	@endforeach
//	</ul>
var (
	commentDirective       = regexp.MustCompile(`(?s)\{\{--.*?--\}\}`)
	ifDirective            = regexp.MustCompile(`@if\s*\((.+?)\)`)
	elseifDirective        = regexp.MustCompile(`@elseif\s*\((.+?)\)`)
	elseDirective          = regexp.MustCompile(`@else\b`)
	endifDirective         = regexp.MustCompile(`@endif\b`)
	foreachAssignDirective = regexp.MustCompile(`@foreach\s*\(\s*(\$\w+)\s+in\s+(.+?)\)`)
	foreachDirective       = regexp.MustCompile(`@foreach\s*\((.+?)\)`)
	endforeachDirective    = regexp.MustCompile(`@endforeach\b`)
	includeDirective       = regexp.MustCompile(`@include\s*\(\s*"(.+?)"\s*\)`)
)

// compileDirectives roda antes do html/template.Parse, transformando a
// sintaxe estilo Blade em sintaxe nativa do html/template. A ordem
// importa: foreachAssignDirective precisa rodar antes de
// foreachDirective, senão o "in" vira parte da expressão da segunda.
func compileDirectives(src string) string {
	src = commentDirective.ReplaceAllString(src, "")
	src = ifDirective.ReplaceAllString(src, `{{if $1}}`)
	src = elseifDirective.ReplaceAllString(src, `{{else if $1}}`)
	src = elseDirective.ReplaceAllString(src, `{{else}}`)
	src = endifDirective.ReplaceAllString(src, `{{end}}`)
	src = foreachAssignDirective.ReplaceAllString(src, `{{range $1 := $2}}`)
	src = foreachDirective.ReplaceAllString(src, `{{range $1}}`)
	src = endforeachDirective.ReplaceAllString(src, `{{end}}`)
	src = includeDirective.ReplaceAllString(src, `{{template "$1" .}}`)
	return src
}