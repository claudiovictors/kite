package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	FgGreen   = "\033[32m"
	FgYellow  = "\033[33m"
	FgCyan    = "\033[36m"
	FgGray    = "\033[90m"

	BadgeInfo    = "\033[44m\033[97m\033[1m INFO \033[0m"
	BadgeSuccess = "\033[42m\033[30m\033[1m DONE \033[0m"
	BadgeError   = "\033[41m\033[97m\033[1m FAIL \033[0m"
)

const (
	Version = "1.0.0"
	Banner  = FgCyan + `
  _  ___ _       
 | |/ / (_) |_ ___ 
 | ' /| | | __/ _ \
 | . \| | | ||  __/
 |_|\_|_|_|\__\___|` + Reset + Dim + `  v` + Version + Reset + "\n"
)

type Command struct {
	Name        string
	Usage       string
	Description string
	Run         func(args []string) error
}

var commands = map[string]Command{}

func RegisterCommand(cmd Command) {
	commands[cmd.Name] = cmd
}

func main() {
	registerAllCommands()

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	commandName := os.Args[1]

	switch commandName {
	case "-v", "--version", "version":
		fmt.Printf("%sKite Framework CLI%s v%s\n", Bold, Reset, Version)
		os.Exit(0)
	case "-h", "--help", "help":
		printHelp()
		os.Exit(0)
	}

	cmd, exists := commands[commandName]
	if !exists {
		fmt.Printf("\n%s O comando %q não foi encontrado.\n\n", BadgeError, commandName)
		printHelp()
		os.Exit(1)
	}

	if err := cmd.Run(os.Args[2:]); err != nil {
		fmt.Printf("\n%s %v\n\n", BadgeError, err)
		os.Exit(1)
	}
}

func registerAllCommands() {
	RegisterCommand(Command{
		Name:        "new",
		Usage:       "kite new <nome-do-projeto>",
		Description: "Inicializa uma nova estrutura de projeto Kite.",
		Run:         runNewProject,
	})

	RegisterCommand(Command{
		Name:        "make:controller",
		Usage:       "kite make:controller <Nome>",
		Description: "Gera um novo arquivo de controller/handler.",
		Run:         runMakeController,
	})

	RegisterCommand(Command{
		Name:        "make:middleware",
		Usage:       "kite make:middleware <Nome>",
		Description: "Gera uma nova estrutura de middleware.",
		Run:         runMakeMiddleware,
	})

	RegisterCommand(Command{
		Name:        "make:model",
		Usage:       "kite make:model <Nome>",
		Description: "Gera um modelo de dados para o ORM/Database.",
		Run:         runMakeModel,
	})

	RegisterCommand(Command{
		Name:        "make:migration",
		Usage:       "kite make:migration <nome_da_tabela>",
		Description: "Gera um novo arquivo de migração de banco de dados.",
		Run:         runMakeMigration,
	})

	RegisterCommand(Command{
		Name:        "migrate",
		Usage:       "kite migrate",
		Description: "Executa todas as migrações de banco pendentes.",
		Run:         runMigrate,
	})
}

func printHelp() {
	fmt.Print(Banner)
	fmt.Printf("\n%sUso:%s\n  kite <comando> [argumentos]\n\n", Bold, Reset)
	fmt.Printf("%sComandos disponíveis:%s\n", Bold, Reset)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, cmd := range commands {
		fmt.Fprintf(w, "  %s%s%s\t%s%s%s\n", FgGreen, cmd.Name, Reset, FgGray, cmd.Description, Reset)
	}
	w.Flush()

	fmt.Printf("\n%sOpções:%s\n", Bold, Reset)
	fmt.Printf("  %s-v, --version%s    Exibe a versão atual da CLI.\n", FgGreen, Reset)
	fmt.Printf("  %s-h, --help%s       Exibe este menu de ajuda.\n\n", FgGreen, Reset)
}

func printDotsLine(leftText string, rightText string, success bool) {
	totalWidth := 65
	dotsCount := totalWidth - len(leftText) - len(rightText)
	if dotsCount < 2 {
		dotsCount = 2
	}

	dots := strings.Repeat(".", dotsCount)
	statusColor := FgGreen
	if !success {
		statusColor = FgYellow
	}

	fmt.Printf("  %s%s %s%s %s%s%s\n",
		Bold, leftText,
		FgGray, dots,
		statusColor, rightText, Reset,
	)
}

// ----------------------------------------------------------------------
// Implementação dos Comandos
// ----------------------------------------------------------------------

func runNewProject(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome do projeto. Ex: kite new meu-app")
	}

	projectName := args[0]
	fmt.Printf("\n%s Criando estrutura do projeto [%s]...\n\n", BadgeInfo, projectName)

	dirs := []string{
		projectName,
		filepath.Join(projectName, "app", "controllers"),
		filepath.Join(projectName, "app", "models"),
		filepath.Join(projectName, "app", "middlewares"),
		filepath.Join(projectName, "config"),
		filepath.Join(projectName, "database", "migrations"),
		filepath.Join(projectName, "routes"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("falha ao criar diretório %s: %w", dir, err)
		}
	}

	mainContent := fmt.Sprintf(`package main

import (
	"%s/core"
	"fmt"
)

func main() {
	app := core.NewApp()

	app.Get("/", func(req core.Request, res core.Response) error {
		return res.Json(map[string]string{
			"app":    "%s",
			"status": "online",
		})
	})

	fmt.Println("⚡ Servidor Kite rodando na porta :8080")
	app.Listen(":8080")
}
`, projectName, projectName)

	if err := os.WriteFile(filepath.Join(projectName, "main.go"), []byte(mainContent), 0644); err != nil {
		return fmt.Errorf("falha ao criar main.go: %w", err)
	}

	printDotsLine("Estrutura base", "CRIADA", true)
	printDotsLine("Arquivo main.go", "CRIADO", true)

	fmt.Printf("\n%s Projeto criado com sucesso. Digite %scd %s%s para iniciar.\n\n", BadgeSuccess, Bold, projectName, Reset)
	return nil
}

func runMakeController(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome do controller. Ex: kite make:controller User")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + "_controller.go"
	path := filepath.Join("app", "controllers", fileName)

	content := fmt.Sprintf(`package controllers

import (
	"kite/core"
)

type %sController struct{}

func (c *%sController) Index(req core.Request, res core.Response) error {
	return res.Json(map[string]string{"message": "Lista de %s"})
}
`, name, name, name)

	return generateFileWithArtisanOutput("Controller", path, content)
}

func runMakeMiddleware(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome do middleware. Ex: kite make:middleware Auth")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + ".go"
	path := filepath.Join("app", "middlewares", fileName)

	content := fmt.Sprintf(`package middlewares

import (
	"kite/core"
)

func %s() core.MiddlewareFunc {
	return func(next core.HandlerFunc) core.HandlerFunc {
		return func(req core.Request, res core.Response) error {
			return next(req, res)
		}
	}
}
`, name)

	return generateFileWithArtisanOutput("Middleware", path, content)
}

func runMakeModel(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome da model. Ex: kite make:model Product")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + ".go"
	path := filepath.Join("app", "models", fileName)

	content := fmt.Sprintf(`package models

import "time"

type %s struct {
	ID        uint      ` + "`json:\"id\" db:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\" db:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\" db:\"updated_at\"`" + `
}
`, name, name)

	return generateFileWithArtisanOutput("Model", path, content)
}

/**
 * Gera um arquivo de migration no formato timestamp_create_tableName_table.go.
 */
func runMakeMigration(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome da tabela. Ex: kite make:migration create_users_table")
	}

	rawName := strings.ToLower(args[0])
	timestamp := time.Now().Format("20060102150405")
	fileName := fmt.Sprintf("%s_%s.go", timestamp, rawName)
	path := filepath.Join("database", "migrations", fileName)

	structName := strings.ReplaceAll(strings.Title(strings.ReplaceAll(rawName, "_", " ")), " ", "")

	content := fmt.Sprintf(`package migrations

import (
	"database/sql"
)

/**
 * Migration %s
 */
type %s struct{}

func (m *%s) Up(db *sql.DB) error {
	query := ` + "`" + `
	CREATE TABLE IF NOT EXISTS %s (
		id INT AUTO_INCREMENT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	);
	` + "`" + `
	_, err := db.Exec(query)
	return err
}

func (m *%s) Down(db *sql.DB) error {
	query := ` + "`" + `DROP TABLE IF EXISTS %s;` + "`" + `
	_, err := db.Exec(query)
	return err
}
`, structName, structName, structName, rawName, structName, rawName)

	return generateFileWithArtisanOutput("Migration", path, content)
}

/**
 * Simula a execução das migrações pendentes e exibe a saída no estilo Artisan.
 */
func runMigrate(args []string) error {
	migrationsDir := filepath.Join("database", "migrations")
	files, err := os.ReadDir(migrationsDir)
	if err != nil || len(files) == 0 {
		fmt.Printf("\n%s Nenhuma migração pendente encontrada em [%s].\n\n", BadgeInfo, migrationsDir)
		return nil
	}

	fmt.Printf("\n%s Executando migrações no banco de dados...\n\n", BadgeInfo)

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			migrationName := strings.TrimSuffix(file.Name(), ".go")
			printDotsLine(migrationName, "RAN", true)
		}
	}

	fmt.Printf("\n%s Todas as migrações foram executadas com sucesso.\n\n", BadgeSuccess)
	return nil
}

func generateFileWithArtisanOutput(resourceType string, path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório %s: %w", dir, err)
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("o arquivo [%s] já existe", path)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("erro ao escrever arquivo [%s]: %w", path, err)
	}

	fmt.Println()
	fmt.Printf("%s Created %s [%s].\n", BadgeInfo, resourceType, path)
	fmt.Println()
	printDotsLine(path, "DONE", true)
	fmt.Println()

	return nil
}