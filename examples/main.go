package main

import (
	"log"

	kite "github.com/claudiovictors/kite/core"
)

type CreateUserDTO struct {
	Name  string `json:"name" doc:"Nome completo do usuário" example:"Carlos Silva" validate:"required"`
	Email string `json:"email" doc:"Endereço de e-mail válido" format:"email" example:"carlos@email.com" validate:"required"`
	Role  string `json:"role" doc:"Cargo do usuário" example:"admin" default:"member"`
}

type UserResponse struct {
	ID    string `json:"id" example:"usr_100"`
	Name  string `json:"name" example:"Carlos Silva"`
	Email string `json:"email" example:"carlos@email.com"`
	Role  string `json:"role" example:"admin"`
}

func main() {
	// Cria a aplicação Kite com OpenAPI 3.1 e Scalar embutidos
	app := kite.New(kite.Config{
		Title:        "Kite Store API",
		Version:      "1.0.0",
		Description:  "Demonstração de API em Go com documentação interativa via Scalar estilo FastAPI.",
		ScalarTheme:  kite.ThemeDefault,
		ScalarLayout: "classic",
	})

	// Rota básica
	app.Get("/", func(req kite.Request, res kite.Response) error {
		return res.Json(kite.Map{
			"message": "Bem-vindo ao Kite!",
			"docs":    "/docs",
			"openapi": "/openapi.json",
		})
	}).
		Summary("Healthcheck e boas-vindas").
		Tags("Geral")

	// Listar usuários
	app.Get("/users", func(req kite.Request, res kite.Response) error {
		users := []UserResponse{
			{ID: "1", Name: "Carlos Silva", Email: "carlos@email.com", Role: "admin"},
			{ID: "2", Name: "Ana Souza", Email: "ana@email.com", Role: "member"},
		}
		return res.Json(users)
	}).
		Summary("Listar todos os usuários").
		Description("Retorna a lista completa de usuários cadastrados no sistema").
		Tags("Usuários").
		Response(200, []UserResponse{}, "Lista de usuários")

	// Buscar usuário por ID
	app.Get("/users/:id", func(req kite.Request, res kite.Response) error {
		id := req.Param("id")
		return res.Json(UserResponse{
			ID:    id,
			Name:  "Carlos Silva",
			Email: "carlos@email.com",
			Role:  "admin",
		})
	}).
		Summary("Obter usuário por ID").
		Description("Busca um usuário específico pelo seu identificador único").
		Tags("Usuários").
		Response(200, UserResponse{}, "Usuário encontrado").
		Response(404, kite.ErrorResponse{}, "Usuário não encontrado")

	// Criar usuário com validação e DTO documentado
	app.Post("/users", func(req kite.Request, res kite.Response) error {
		var dto CreateUserDTO
		if err := req.BindJson(&dto); err != nil {
			return res.Status(400).Json(kite.ErrorResponse{
				Error:  "Corpo da requisição inválido: " + err.Error(),
				Status: 400,
			})
		}

		created := UserResponse{
			ID:    "usr_999",
			Name:  dto.Name,
			Email: dto.Email,
			Role:  dto.Role,
		}
		return res.Status(201).Json(created)
	}).
		Summary("Cadastrar novo usuário").
		Description("Cria um novo usuário a partir dos dados enviados no corpo da requisição").
		Tags("Usuários").
		Body(CreateUserDTO{}, "Dados de criação do usuário").
		Response(201, UserResponse{}, "Usuário criado com sucesso").
		Response(400, kite.ErrorResponse{}, "Erro de validação nos dados")

	log.Println("🚀 Servidor iniciado!")
	log.Println("📖 Documentação interativa (Scalar): http://localhost:3000/docs")
	log.Println("📄 Especificação OpenAPI: http://localhost:3000/openapi.json")

	log.Fatal(app.Listen(":3000"))
}
