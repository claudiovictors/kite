<h1 align="center">Kite</h1>

<div align="center">

[![Version](https://img.shields.io/badge/version-1.0.0-blue)](CHANGELOG.md)
[![Go Version](https://img.shields.io/badge/go-1.26.5-00ADD8?logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Status](https://img.shields.io/badge/status-em%20produção-green)](CHANGELOG.md)
[![Go Reference](https://pkg.go.dev/badge/github.com/claudiovictors/kite.svg)](https://pkg.go.dev/github.com/claudiovictors/kite)
[![Go Report Card](https://goreportcard.com/badge/github.com/claudiovictors/kite)](https://goreportcard.com/report/github.com/claudiovictors/kite)

</div>

Kite é um micro-framework web para Go, inspirado no Express.js e no
Laravel. Reúne, num único módulo e sem dependências externas, um
núcleo HTTP com roteamento e middlewares, um ORM estilo Eloquent sobre
`database/sql`, autenticação via JWT e sessão, e uma engine de views
com diretivas ao estilo Blade.

## Características

- Zero dependências externas: construído inteiramente sobre a
  standard library do Go (`net/http`, `database/sql`, `html/template`,
  `crypto/hmac`, `crypto/rand`, `reflect`).
- Documentação interativa integrada com **Scalar** (estilo FastAPI): geração automática de **OpenAPI 3.1** e UI moderna em `/docs` sem configurações extras.
- API fluente e encadeável, próxima da sintaxe do Express e do
  Eloquent.
- Router baseado em árvore de segmentos, com suporte a parâmetros
  nomeados (`:id`) e wildcard (`*`).
- ORM com Query Builder genérico (`Query[T]`), paginação, joins e
  relacionamentos manuais (`HasMany`, `BelongsTo`).
- Autenticação JWT (HS256) e sessão baseada em cookie, com store em
  memória incluída.
- Engine de templates sobre `html/template`, com diretivas
  `@if`, `@foreach`, `@include` e comentários `{{-- --}}`.

## Instalação

```bash
go get github.com/claudiovictors/kite
```

Requer Go 1.26.5 ou superior.

## Exemplo rápido

```go
package main

import kite "github.com/claudiovictors/kite/core"

func main() {
	app := kite.New()

	app.Get("/", func(req kite.Request, res kite.Response) error {
		return res.Json(map[string]string{"message": "olá a partir do Kite"})
	})

	app.Get("/users/:id", func(req kite.Request, res kite.Response) error {
		return res.Json(map[string]string{"id": req.Param("id")})
	})

	app.Listen(":3000")
}
```

## Documentação Interativa com Scalar (Estilo FastAPI)

O Kite gera automaticamente a especificação **OpenAPI 3.1** e serve a interface interativa do **Scalar** em `/docs`:

```go
type CreateUserDTO struct {
	Name  string `json:"name" doc:"Nome completo" example:"Carlos Silva" validate:"required"`
	Email string `json:"email" doc:"E-mail" format:"email" example:"carlos@email.com" validate:"required"`
}

type UserResponse struct {
	ID    string `json:"id" example:"usr_123"`
	Name  string `json:"name" example:"Carlos Silva"`
	Email string `json:"email" example:"carlos@email.com"`
}

func main() {
	app := kite.New(kite.Config{
		Title:   "Minha API",
		Version: "1.0.0",
		DocsURL: "/docs", // Scalar UI (padrão: /docs)
	})

	app.Post("/users", func(req kite.Request, res kite.Response) error {
		var dto CreateUserDTO
		if err := req.BindJson(&dto); err != nil {
			return res.Status(400).Json(kite.Map{"error": err.Error()})
		}
		return res.Status(201).Json(UserResponse{ID: "1", Name: dto.Name, Email: dto.Email})
	}).
		Summary("Cadastrar usuário").
		Tags("Usuários").
		Body(CreateUserDTO{}).
		Response(201, UserResponse{})

	app.Listen(":3000")
}
```

Ao rodar a aplicação:
- Acesse `http://localhost:3000/docs` para ver a interface interativa do **Scalar**.
- Acesse `http://localhost:3000/openapi.json` para obter o schema OpenAPI 3.1.

### Personalizando o Visual (Tema e Layout)

Por padrão, o Kite adota o layout `classic` do Scalar, pois ele se assemelha mais à interface tradicional do Swagger. Se preferir um visual mais moderno e focado em clientes de API, basta alterar a configuração `ScalarLayout` para `"modern"`. Você também pode alterar o tema de cores padrão através de `ScalarTheme`.

```go
	app := kite.New(kite.Config{
		Title:        "Minha API",
		Version:      "1.0.0",
		ScalarLayout: "modern",         // "classic" (padrão) ou "modern"
		ScalarTheme:  kite.ThemePurple, // diversas opções de temas disponíveis
	})
```

## Estrutura do projeto

```
kite/
├── auth/           # autenticação JWT e sessão
│   ├── jwt.go
│   ├── middleware.go
│   └── session.go
├── core/           # App, router, middlewares, Request/Response
│   ├── app.go
│   ├── context.go
│   ├── middleware.go
│   └── router.go
├── database/       # ORM estilo Eloquent sobre database/sql
│   ├── orm.go
│   ├── query_builder.go
│   └── relations.go
├── template/       # engine de views com diretivas estilo Blade
│   ├── directives.go
│   └── engine.go
├── examples/       # aplicação de exemplo
│   └── main.go
├── go.mod
├── LICENSE
├── README.md
└── CHANGELOG.md
```

## Executando os testes

```bash
go test ./... -v
```

## Executando o exemplo

```bash
cd examples
go run main.go
```

## Estado do projeto

O Kite está em desenvolvimento ativo. Os pacotes `core`, `database`,
`auth` e `template` já são funcionais, mas a API pode sofrer alterações
até a primeira versão estável. Itens conhecidos no roadmap:

- Active Record no ORM (`model.Save()`, `model.Delete()`)
- Migrations
- Eager loading de relacionamentos (`.With(...)`)
- Suporte a RS256/ES256 no JWT
- Parser dedicado para as diretivas de template, em substituição à
  implementação atual baseada em expressões regulares

Consulte o [CHANGELOG](CHANGELOG.md) para o histórico de alterações.

## Licença

Este projeto está licenciado sob a licença MIT. Consulte o ficheiro
[LICENSE](LICENSE) para o texto completo.