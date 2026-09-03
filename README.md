<div align="center">
	<img src="logo.svg"  width="70px">
</div>

<h1 align="center">Kite Framework</h1>

<div align="center">

[![Version](https://img.shields.io/badge/version-1.0.5-blue)](CHANGELOG.md)
[![Go Version](https://img.shields.io/badge/go-1.26.5-00ADD8?logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Status](https://img.shields.io/badge/status-em%20produção-green)](CHANGELOG.md)
[![Go Reference](https://pkg.go.dev/badge/github.com/claudiovictors/kite.svg)](https://pkg.go.dev/github.com/claudiovictors/kite)
[![Go Report Card](https://goreportcard.com/badge/github.com/claudiovictors/kite)](https://goreportcard.com/report/github.com/claudiovictors/kite)

</div>

Kite is a web micro-framework for Go, inspired by Express.js and Laravel. It brings, in a single module and without external dependencies, an HTTP core with routing and middlewares, an Eloquent-style ORM over `database/sql`, JWT/session authentication, and a Blade-like template engine.

## Features

- Zero external dependencies: built entirely on the Go standard library (`net/http`, `database/sql`, `html/template`, `crypto/hmac`, `crypto/rand`, `reflect`).
- Integrated interactive documentation with **Scalar** (FastAPI-like): automatic **OpenAPI 3.1** generation and a modern UI at `/docs` with no extra setup.
- Fluent, chainable API, close to the syntax of Express and Eloquent.
- Tree-based router, with support for named parameters (`:id`) and wildcard (`*`).
- ORM with a generic Query Builder (`Query[T]`), pagination, joins and manual relationships (`HasMany`, `BelongsTo`).
- JWT authentication (HS256) and cookie-based session with an in-memory store included.
- Template engine on top of `html/template`, with directives `@if`, `@foreach`, `@include` and comments `{{-- --}}`.

## Installation

```bash
go get github.com/claudiovictors/kite
```

Requires Go 1.26.5 or newer.

## Quick example

```go
package main

import kite "github.com/claudiovictors/kite/core"

func main() {
	app := kite.New()

	app.Get("/", func(req kite.Request, res kite.Response) error {
		return res.Json(map[string]string{"message": "hello from Kite"})
	})

	app.Get("/users/:id", func(req kite.Request, res kite.Response) error {
		return res.Json(map[string]string{"id": req.Param("id")})
	})

	app.Listen(":3000")
}
```

## Interactive Documentation with Scalar (FastAPI-like)

Kite automatically generates the **OpenAPI 3.1** specification and serves the interactive **Scalar** UI at `/docs`:

```go
type CreateUserDTO struct {
	Name  string `json:"name" doc:"Full name" example:"Carlos Silva" validate:"required"`
	Email string `json:"email" doc:"Email" format:"email" example:"carlos@email.com" validate:"required"`
}

type UserResponse struct {
	ID    string `json:"id" example:"usr_123"`
	Name  string `json:"name" example:"Carlos Silva"`
	Email string `json:"email" example:"carlos@email.com"`
}

func main() {
	app := kite.New(kite.Config{
		Title:   "My API",
		Version: "1.0.0",
		DocsURL: "/docs", // Scalar UI (default: /docs)
	})

	app.Post("/users", func(req kite.Request, res kite.Response) error {
		var dto CreateUserDTO
		if err := req.BindJson(&dto); err != nil {
			return res.Status(400).Json(kite.Map{"error": err.Error()})
		}
		return res.Status(201).Json(UserResponse{ID: "1", Name: dto.Name, Email: dto.Email})
	}).
		Summary("Register user").
		Tags("Users").
		Body(CreateUserDTO{}).
		Response(201, UserResponse{})

	app.Listen(":3000")
}
```

When running the application:
- Visit `http://localhost:3000/docs` to see the interactive **Scalar** UI.
- Visit `http://localhost:3000/openapi.json` to get the OpenAPI 3.1 schema.

### Customizing the Look (Theme and Layout)

By default Kite uses the Scalar `classic` layout, which resembles a traditional Swagger-like interface. If you prefer a more modern API-focused UI, change the `ScalarLayout` to `"modern"`. You can also change the theme color via `ScalarTheme`.

```go
	app := kite.New(kite.Config{
		Title:        "My API",
		Version:      "1.0.0",
		ScalarLayout: "modern",         // "classic" (default) or "modern"
		ScalarTheme:  kite.ThemePurple, // various theme options available
	})
```

## Project structure

```
kite/
├── auth/           # JWT and session authentication
│   ├── jwt.go
│   ├── middleware.go
│   └── session.go
├── core/           # App, router, middlewares, Request/Response
│   ├── app.go
│   ├── context.go
│   ├── middleware.go
│   └── router.go
├── database/       # Eloquent-style ORM over database/sql
│   ├── orm.go
│   ├── query_builder.go
│   └── relations.go
├── template/       # Blade-style template engine
│   ├── directives.go
│   └── engine.go
├── examples/       # example application
│   └── main.go
├── go.mod
├── LICENSE
├── README.md
└── CHANGELOG.md
```

## Running tests

```bash
go test ./... -v
```

## Running the example

```bash
cd examples
go run main.go
```

## Project status

Kite is in active development. The `core`, `database`, `auth` and `template` packages are already functional, but the API may change until the first stable release. Known roadmap items:

- Active Record in the ORM (`model.Save()`, `model.Delete()`)
- Migrations
- Eager loading of relationships (`.With(...)`)
- Support for RS256/ES256 in JWT
- Dedicated parser for template directives to replace the current regex-based implementation

See the [CHANGELOG](CHANGELOG.md) for the history of changes.

## License