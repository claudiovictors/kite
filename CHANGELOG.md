## [Unreleased]

### Added
- **`examples`**: Added a new HTML template example (`examples/views/hello.html`) using Kite's Blade-like template engine, styled with Pico CSS classless.
- **`examples`**: Refactored `examples/main.go` to remove duplicate code and extract mock data to a package-level variable.

### Changed
- **`docs`**: Translated the entire `CHANGELOG.md` and inline comments in `examples/main.go` to English for broader accessibility.

### Fixed
- **`template`**: Removed the unused variable `pattern` in `engine.go` and fixed the "view not found" error message to display the correct extension.
- **`template`**: Removed the ambiguous fallback regex `foreachDirective` in `directives.go`, ensuring that only the strict syntax `@foreach($var in collection)` is supported (prevents silent generation of invalid Go templates).

## [1.0.4] - 2026-08-31

### Added
- **Interactive documentation with Scalar** (`docs`, FastAPI style): Kite now automatically generates the **OpenAPI 3.1** specification from registered routes and serves the interactive **Scalar** interface at `/docs` (configurable via `kite.Config.DocsURL`). The raw schema is available at `/openapi.json`.
- **`kite.Config`**: initial application configuration struct (`Title`, `Version`, `DocsURL`, `ScalarLayout`, `ScalarTheme`), passed in `kite.New(kite.Config{...})`.
- **Chainable route metadata**: `.Summary(text)`, `.Tags(names...)`, `.Body(dto)` and `.Response(status, dto)`, allowing you to describe each endpoint (summary, tagging, input/output schema) directly in the `app.Post/Get/...` chaining.
- **Struct tags for OpenAPI**: support for `doc` (field description), `example` (example value) and `format` (e.g. `email`) in DTO/response structs, used in automatic schema generation.
- **Scalar Themes**: `ScalarTheme` with several pre-defined options (e.g. `kite.ThemePurple`), applied in the `/docs` UI.
- **Scalar Layout**: `ScalarLayout` with `"classic"` (default, closer to traditional Swagger) or `"modern"` (more focused on API clients).

### Credits
- Feature implemented in collaboration — PR reviewed and integrated together with [Carlos Felipe Araújo](https://github.com/carlosxfelipe), who brought the initial implementation of OpenAPI schema generation and integration with Scalar.

## [1.0.2] - 2026-08-29

### Added
- **Input validation** (`validation`): `validation.Make(data, rules)` in Laravel Validator style, with rules `required`, `email`, `min`, `max`, `numeric`, `integer`, `boolean`, `string`, `alpha`, `alpha_num`, `in`, `same`, `confirmed`, `url`, `uuid`, `regex`, `array`, `nullable`, customizable messages and support for custom rules via `validation.RegisterRule`.
- **Laravel style migrations** (`database`): `Schema`/`Blueprint`/`Migrator`, with `Create`, `Table`, `Drop`, `DropIfExists`, typed columns (`ID`, `String`, `Text`, `Integer`, `BigInteger`, `Float`, `Decimal`, `Boolean`, `Date`, `Timestamp`, `Timestamps`, `SoftDeletes`, `ForeignID().References().On()`), modifiers (`Nullable`, `Unique`, `Default`), and `Migrator.Run/Rollback/Status` with batch control.
- **Seeders** (`database`): `Seeder`, `NamedSeeder` and `SeederRunner` to populate initial/test data.
- **`database.RunCLI`**: dispatcher to bind real `migrate`, `migrate:status`, `migrate:rollback` and `seed` to the application's `main()`, used by the global CLI via `go run . <command>`.
- **CORS** (`core/cors.go`): `kite.CORS(config...)` middleware with configurable origin/methods/headers, credentials support and automatic preflight.
- **Automatic OPTIONS registration**: every registered route (`Get`, `Post`, ...) now also registers a silent `OPTIONS` handler in the same middleware chain, allowing `CORS()` to properly respond to preflights.
- **Middleware per route and named routes**: `app.Get(...).Middleware(...)`, `.Name(...)` and `app.URLFor(name, params)` for reverse routing (extra parameters become query string).
- **`Request.Input`/`Has`/`All`/`Only`/`Except`**: unified access to input data (JSON, form and query string), in Laravel's `$request->input()` style. `GetBody()` as an alias for `Body()`.
- **`Response.Render(name, data)`**: rendering views via `app.LoadViews(dir, ext)`, without needing to pass the engine manually. `RenderWith(engine, name, data)` for cases with multiple engines.
- **Chainable `Response.Redirect()`**: `res.Redirect().To(url)`, `.Permanently(url)`, `.Back(req)`.
- **`core/helpers.go`**: `kite.Map`, `kite.Env`, `kite.Must`, `kite.Ptr`, `kite.Coalesce`, `kite.Contains`, `kite.Truncate`, `kite.Slugify`, `kite.RandomString`, `kite.ToJSON`.
- **CLI**: `make:seeder`, `make:request`, `make:test`, `serve`, `migrate`, `migrate:status`, `migrate:rollback`, `seed`.

### Fixed
- Global/group middlewares running in duplicate on routes that used `.Middleware(...)` (the `rawHandler` stored in the route node already came with the base middlewares applied).
- CORS preflight (`OPTIONS`) returning 404 because there was no route registered for this method.
- `cmd/kite/main.go`: `runMakeModel` passed an extra argument to `fmt.Sprintf` (a single `%s` in the template, two `name`s passed).

### Changed
- **Breaking**: `Response.Redirect(url string, code ...int) error` became `Response.Redirect() *Redirector`, use `.To(url, code...)`.
- **Breaking**: `Response.Render(engine, name, data)` became `Response.Render(name, data)`, using the configured engine via `app.LoadViews`. The old behavior (passing the engine manually) is available in `RenderWith`.