## [1.0.2] - 2026-08-29

### Adicionado
- **Validação de input** (`validation`): `validation.Make(data, rules)` no estilo Laravel Validator, com regras `required`, `email`, `min`, `max`, `numeric`, `integer`, `boolean`, `string`, `alpha`, `alpha_num`, `in`, `same`, `confirmed`, `url`, `uuid`, `regex`, `array`, `nullable`, mensagens customizáveis e suporte a regras próprias via `validation.RegisterRule`.
- **Migrations estilo Laravel** (`database`): `Schema`/`Blueprint`/`Migrator`, com `Create`, `Table`, `Drop`, `DropIfExists`, colunas tipadas (`ID`, `String`, `Text`, `Integer`, `BigInteger`, `Float`, `Decimal`, `Boolean`, `Date`, `Timestamp`, `Timestamps`, `SoftDeletes`, `ForeignID().References().On()`), modificadores (`Nullable`, `Unique`, `Default`), e `Migrator.Run/Rollback/Status` com controle de batches.
- **Seeders** (`database`): `Seeder`, `NamedSeeder` e `SeederRunner` para popular dados iniciais/de teste.
- **`database.RunCLI`**: dispatcher para ligar `migrate`, `migrate:status`, `migrate:rollback` e `seed` de verdade ao `main()` da aplicação, usado pela CLI global via `go run . <comando>`.
- **CORS** (`core/cors.go`): middleware `kite.CORS(config...)` com origem/métodos/headers configuráveis, suporte a credenciais e preflight automático.
- **Registro automático de OPTIONS**: toda rota registrada (`Get`, `Post`, ...) agora também registra um handler `OPTIONS` silencioso na mesma cadeia de middlewares, permitindo que `CORS()` responda preflights de verdade.
- **Middleware por rota e rotas nomeadas**: `app.Get(...).Middleware(...)`, `.Name(...)` e `app.URLFor(name, params)` para reverse routing (com parâmetros extras viram query string).
- **`Request.Input`/`Has`/`All`/`Only`/`Except`**: acesso unificado a dados de entrada (JSON, formulário e query string), no estilo `$request->input()` do Laravel. `GetBody()` como alias de `Body()`.
- **`Response.Render(name, data)`**: renderização de views via `app.LoadViews(dir, ext)`, sem precisar passar o engine manualmente. `RenderWith(engine, name, data)` para casos com múltiplos engines.
- **`Response.Redirect()` encadeável**: `res.Redirect().To(url)`, `.Permanently(url)`, `.Back(req)`.
- **`core/helpers.go`**: `kite.Map`, `kite.Env`, `kite.Must`, `kite.Ptr`, `kite.Coalesce`, `kite.Contains`, `kite.Truncate`, `kite.Slugify`, `kite.RandomString`, `kite.ToJSON`.
- **CLI**: `make:seeder`, `make:request`, `make:test`, `serve`, `migrate`, `migrate:status`, `migrate:rollback`, `seed`.

### Corrigido
- Middlewares globais/de grupo rodando em duplicidade em rotas que usavam `.Middleware(...)` (o `rawHandler` guardado no node de rota já vinha com os middlewares base aplicados).
- Preflight CORS (`OPTIONS`) devolvendo 404 por não existir nenhuma rota registrada para esse método.
- `cmd/kite/main.go`: `runMakeModel` passava um argumento a mais pro `fmt.Sprintf` (`%s` único no template, dois `name` passados).

### Alterado
- **Breaking**: `Response.Redirect(url string, code ...int) error` virou `Response.Redirect() *Redirector`, use `.To(url, code...)`.
- **Breaking**: `Response.Render(engine, name, data)` virou `Response.Render(name, data)`, usando o engine configurado via `app.LoadViews`. O comportamento antigo (passar o engine na mão) está disponível em `RenderWith`.