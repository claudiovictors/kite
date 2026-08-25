# Changelog

Todas as alterações relevantes deste projeto serão documentadas neste
ficheiro.

O formato segue as recomendações do
[Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/), e este
projeto segue o [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## v1.0.0

Primeira versão de desenvolvimento do Kite. Ainda não foi publicada
uma tag/release; a API pode sofrer alterações até a primeira versão
estável (1.0.0).

### Adicionado

- **core**: `App` com `New()`, `Use`, `Group`, `Listen`, e registo de
  rotas via `Get`/`Post`/`Put`/`Delete`/`Patch`.
- **core**: router baseado em árvore de segmentos, com suporte a
  parâmetros nomeados (`:id`) e wildcard (`*`), com prioridade de
  casamento literal > parâmetro > wildcard.
- **core**: `Request` com `Param`, `Query`, `Queries`, `Header`,
  `Cookie`, `IP`, `ContentType`, `Is`, `Body`, `BindJson` e `Ctx`.
- **core**: `Response` com `Status`, `SetHeader`, `Cookie`, `WithJson`
  / `Json`, `WithText`, `WithHtml`, `Send`, `SendStatus`, `NoContent`,
  `Redirect` e `File`.
- **core**: `MiddlewareFunc` e cadeia de middlewares, com suporte a
  middlewares globais (`App.Use`) e de grupo (`RouteGroup.Use`).
- **core**: `NotFoundHandler` e `ErrorHandler` configuráveis, com
  respostas padrão em JSON via `ErrorResponse`.
- **database**: `Connect`, `Model` base e registo de metadados de
  models via reflection, com tags `db:"..."`.
- **database**: `QueryBuilder[T]` genérico e fluente, com `Where`,
  `OrWhere`, `WhereIn`, `Join`, `LeftJoin`, `Select`, `OrderBy`,
  `OrderByDesc`, `Limit`, `Offset`, `Get`, `First`, `Count`, `Exists`
  e `Paginate`.
- **database**: `Find`, `FindOrFail`, `Raw`, `HasMany` e `BelongsTo`.
- **auth**: autenticação JWT (HS256) com `Sign`, `Verify`, `NewClaims`
  e o middleware `RequireJWT`.
- **auth**: autenticação por sessão com `SessionManager`,
  `MemoryStore`, `Login`, `Logout` e o middleware `RequireSession`.
- **template**: `Engine` sobre `html/template`, com `Load`, `Render`,
  `RenderToString`, `AddFunc` e recarregamento automático em modo
  `Debug`.
- **template**: diretivas estilo Blade (`@if`, `@elseif`, `@else`,
  `@endif`, `@foreach`, `@endforeach`, `@include` e comentários
  `{{-- --}}`), transpiladas para a sintaxe nativa de `html/template`.
- **examples**: aplicação de exemplo demonstrando o uso conjunto de
  router, middlewares, ORM, autenticação e templates.