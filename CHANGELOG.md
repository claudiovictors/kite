# Changelog

Todas as alterações relevantes deste projeto serão documentadas neste
ficheiro.

O formato segue as recomendações do
[Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/), e este
projeto segue o [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## v1.0.1

### Corrigido

- Removida a dependência do driver `mattn/go-sqlite3` do módulo
  principal (`go.mod`/`go.sum` da raiz agora não têm nenhuma
  dependência externa, só stdlib). O exemplo `examples/api` passou a
  viver no seu próprio módulo Go (`examples/api/go.mod`, com
  `replace github.com/claudiovictors/kite => ../..`), evitando que
  quem instalar o Kite como biblioteca puxe drivers de base de dados
  que não vai usar.
- Removido `examples/kite.db` (ficheiro SQLite gerado durante testes
  manuais, indevidamente versionado).
- Adicionado `.gitignore` cobrindo binários, `*.db`/`*.sqlite`, `.env`,
  logs e ficheiros de IDE.

### Testado

- Ciclo CRUD completo validado manualmente via `curl` contra
  `examples/api` (SQLite): `POST /users`, `GET /users/:id`,
  `PUT /users/:id`, `GET /users?page=&per_page=` e
  `DELETE /users/:id`, incluindo o 404 automático após apagar.

## v1.0.0

Primeira versão estável do Kite.

### Adicionado

- **core**: `App` com `New()`, `Use`, `Group`, `Listen`, e registo de
  rotas via `Get`/`Post`/`Put`/`Delete`/`Patch`.
- **core**: router baseado em árvore de segmentos, com suporte a
  parâmetros nomeados (`:id`) e wildcard (`*`), com prioridade de
  casamento literal > parâmetro > wildcard.
- **core**: `NotFoundHandler` e `ErrorHandler` configuráveis, com
  respostas padrão em JSON via `ErrorResponse`.
- **core**: `Request` com parâmetros de rota (`Param`, `ParamInt`,
  `ParamInt64`, `HasParam`), query string (`Query`, `QueryDefault`,
  `QueryInt`, `QueryInt64`, `QueryFloat`, `QueryBool`, `Queries`,
  `HasQuery`), cabeçalhos e metadados (`Header`, `HeaderDefault`,
  `Cookie`, `IP`, `ContentType`, `Is`, `Accepts`, `AcceptsJSON`,
  `AcceptsHTML`, `XHR`, `Host`, `Path`, `Scheme`, `FullURL`,
  `UserAgent`, `Referer`), corpo do pedido (`Body`, `BindJson`,
  `FormValue`, `FormValues`, `FormFile`, `SaveUploadedFile`) e
  contexto (`Ctx`, `WithContext`).
- **core**: `Response` com métodos de configuração (`Status`,
  `SetHeader`, `Type`, `Vary`, `CacheControl`, `Cookie`,
  `ClearCookie`, `Written`), de corpo (`WithJson`/`Json`, `WithText`,
  `WithHtml`, `Send`, `SendStatus`, `NoContent`, `Redirect`, `Render`)
  e de ficheiros (`Attachment`, `File`, `Download` — já com suporte a
  `Range`/cabeçalhos condicionais via o `*http.Request` original).
- **core**: `Response.Send` responde sempre em JSON por predefinição
  (incluindo `string`), comportamento mais previsível para APIs.
- **core**: `MiddlewareFunc` e cadeia de middlewares, globais
  (`App.Use`) e de grupo (`RouteGroup.Use`).
- **database**: `Connect`, `Model` base e registo de metadados via
  reflection, com tags `db:"..."`.
- **database**: `QueryBuilder[T]` genérico e fluente — `Where`,
  `OrWhere`, `WhereIn`, `Join`, `LeftJoin`, `Select`, `OrderBy`,
  `OrderByDesc`, `Limit`, `Offset`, `Get`, `First`, `Count`, `Exists`,
  `Paginate`.
- **database**: CRUD estilo Eloquent — `Create`, `Update`, `Save`,
  `Delete`, `DeleteModel`, e as variantes em massa
  `QueryBuilder.Update(values)` / `QueryBuilder.Delete()`.
- **database**: `Find`, `FindOrFail` (com `ErrNotFound`), `Raw`,
  `HasMany`, `BelongsTo`.
- **auth**: JWT (HS256) com `Sign`, `Verify`, `NewClaims`,
  `RequireJWT`.
- **auth**: sessões com `SessionManager`, `MemoryStore`, `Login`,
  `Logout`, `RequireSession`.
- **template**: `Engine` sobre `html/template` com árvore única de
  templates, `Load`, `Render`, `RenderToString`, `AddFunc`,
  recarregamento automático em modo `Debug`.
- **template**: diretivas estilo Blade (`@if`, `@elseif`, `@else`,
  `@endif`, `@foreach`, `@endforeach`, `@include`, comentários
  `{{-- --}}`).
- **examples**: `examples/api` — mini API com CRUD completo sobre
  SQLite.