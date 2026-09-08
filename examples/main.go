package main

import (
	"log"

	kite "github.com/claudiovictors/kite/core"
)

type CreateUserDTO struct {
	Name  string `json:"name" doc:"Full name of the user" example:"Carlos Silva" validate:"required"`
	Email string `json:"email" doc:"Valid email address" format:"email" example:"carlos@email.com" validate:"required"`
	Role  string `json:"role" doc:"User role" example:"admin" default:"member"`
}

type UserResponse struct {
	ID    string `json:"id" example:"usr_100"`
	Name  string `json:"name" example:"Carlos Silva"`
	Email string `json:"email" example:"carlos@email.com"`
	Role  string `json:"role" example:"admin"`
}

var mockUsers = []UserResponse{
	{ID: "1", Name: "Carlos Silva", Email: "carlos@email.com", Role: "admin"},
	{ID: "2", Name: "Ana Souza", Email: "ana@email.com", Role: "member"},
}

func main() {
	// Creates the Kite application with built-in OpenAPI 3.1 and Scalar
	app := kite.New(kite.Config{
		Title:       "Kite Store API",
		Version:     "1.0.0",
		Description: "Go API demonstration with interactive documentation via FastAPI-style Scalar.",
	})

	// Configure the template engine
	if err := app.LoadViews("views", ".html"); err != nil {
		log.Fatalf("Error loading views: %v", err)
	}

	// Basic route
	app.Get("/", func(req kite.Request, res kite.Response) error {
		return res.Json(kite.Map{
			"message": "Welcome to Kite!",
			"docs":    "/docs",
			"openapi": "/openapi.json",
		})
	}).
		Summary("Healthcheck and welcome").
		Tags("General")

	// List users
	app.Get("/users", func(req kite.Request, res kite.Response) error {
		return res.Json(mockUsers)
	}).
		Summary("List all users").
		Description("Returns the full list of users registered in the system").
		Tags("Users").
		Response(200, []UserResponse{}, "List of users")

	// Get user by ID
	app.Get("/users/:id", func(req kite.Request, res kite.Response) error {
		id := req.Param("id")
		for _, user := range mockUsers {
			if user.ID == id {
				return res.Json(user)
			}
		}
		return res.Status(404).Json(kite.ErrorResponse{
			Error:  "User not found",
			Status: 404,
		})
	}).
		Summary("Get user by ID").
		Description("Fetches a specific user by their unique identifier").
		Tags("Users").
		Response(200, UserResponse{}, "User found").
		Response(404, kite.ErrorResponse{}, "User not found")

	// Create user with validation and documented DTO
	app.Post("/users", func(req kite.Request, res kite.Response) error {
		var dto CreateUserDTO
		if err := req.BindJson(&dto); err != nil {
			return res.Status(400).Json(kite.ErrorResponse{
				Error:  "Invalid request body: " + err.Error(),
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
		Summary("Register new user").
		Description("Creates a new user from the data sent in the request body").
		Tags("Users").
		Body(CreateUserDTO{}, "User creation data").
		Response(201, UserResponse{}, "User successfully created").
		Response(400, kite.ErrorResponse{}, "Validation error in data")

	// Render HTML Template
	app.Get("/html", func(req kite.Request, res kite.Response) error {
		return res.Render("hello", kite.Map{
			"title": "Kite Template Example",
			"name":  "Visitor",
			"users": mockUsers,
		})
	}).
		Summary("Render HTML Template").
		Description("Demonstrates how to render HTML using Kite's Blade-like template engine").
		Tags("General")

	log.Println("🚀 Server started!")
	log.Println("📖 Interactive documentation (Scalar): http://localhost:3000/docs")
	log.Println("📄 OpenAPI specification: http://localhost:3000/openapi.json")
	log.Println("🖼️ Template Example: http://localhost:3000/html")

	log.Fatal(app.Listen(":3000"))
}
