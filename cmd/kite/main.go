package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	Reset    = "\033[0m"
	Bold     = "\033[1m"
	Dim      = "\033[2m"
	FgGreen  = "\033[32m"
	FgYellow = "\033[33m"
	FgCyan   = "\033[36m"
	FgGray   = "\033[90m"

	BadgeInfo    = "\033[44m\033[97m\033[1m INFO \033[0m"
	BadgeSuccess = "\033[42m\033[30m\033[1m DONE \033[0m"
	BadgeError   = "\033[41m\033[97m\033[1m FAIL \033[0m"
)

const (
	Version = "1.0.4"
	Banner  = FgCyan + `
  _  ___ _       
 | |/ / (_) |_ ___ 
 | ' /| | | __/ _ \
 | . \| | | ||  __/
 |_|\_|_|_|\__\___|` + Reset + Dim + `  v` + Version + Reset + "\n"
)

/**
 * Command representa um comando registrado na CLI: seu nome de invocação,
 * a linha de uso exibida no help, uma descrição curta e a função executada
 * quando o comando é chamado.
 *
 * @property {string} Name - Nome usado na linha de comando (ex.: "make:model").
 * @property {string} Usage - Exemplo de uso exibido no help.
 * @property {string} Description - Descrição curta exibida na lista de comandos.
 * @property {func(args []string) error} Run - Função executada com os argumentos após o nome do comando.
 */
type Command struct {
	Name        string
	Usage       string
	Description string
	Run         func(args []string) error
}

var commands = map[string]Command{}

/**
 * RegisterCommand adiciona um Command ao índice global de comandos
 * disponíveis, indexado pelo seu Name.
 *
 * @param cmd Command - Comando a ser registrado.
 */
func RegisterCommand(cmd Command) {
	commands[cmd.Name] = cmd
}

/**
 * main é o ponto de entrada da CLI: registra todos os comandos, resolve
 * flags globais (-v/--version, -h/--help), localiza o Command pelo nome
 * informado em os.Args[1] e executa seu Run com o restante dos argumentos.
 */
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

/**
 * registerAllCommands centraliza o registro de todos os comandos
 * conhecidos pela CLI. Chamado uma única vez, no início de main().
 */
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
		Name:        "make:seeder",
		Usage:       "kite make:seeder <Nome>",
		Description: "Gera um novo arquivo de seeder de banco de dados.",
		Run:         runMakeSeeder,
	})

	RegisterCommand(Command{
		Name:        "make:request",
		Usage:       "kite make:request <Nome>",
		Description: "Gera uma classe de validação de request (FormRequest).",
		Run:         runMakeRequest,
	})

	RegisterCommand(Command{
		Name:        "make:test",
		Usage:       "kite make:test <Nome>",
		Description: "Gera um esqueleto de teste HTTP para um controller.",
		Run:         runMakeTest,
	})

	RegisterCommand(Command{
		Name:        "migrate",
		Usage:       "kite migrate",
		Description: "Executa as migrações pendentes da aplicação atual (via database.RunCLI).",
		Run:         runMigrate,
	})

	RegisterCommand(Command{
		Name:        "migrate:status",
		Usage:       "kite migrate:status",
		Description: "Mostra o estado (executada/pendente) de cada migração registrada.",
		Run:         runMigrateStatus,
	})

	RegisterCommand(Command{
		Name:        "migrate:rollback",
		Usage:       "kite migrate:rollback [steps]",
		Description: "Reverte o último lote de migrações (ou os últimos N lotes).",
		Run:         runMigrateRollback,
	})

	RegisterCommand(Command{
		Name:        "seed",
		Usage:       "kite seed",
		Description: "Roda os seeders registrados na aplicação atual (via database.RunCLI).",
		Run:         runSeed,
	})

	RegisterCommand(Command{
		Name:        "serve",
		Usage:       "kite serve [porta]",
		Description: "Roda a aplicação atual com 'go run .' (equivalente ao 'go run .' na mão).",
		Run:         runServe,
	})
}

/**
 * printHelp imprime o banner da CLI, a lista de comandos registrados
 * (alinhados em colunas via tabwriter) e as opções globais disponíveis.
 */
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

/**
 * printDotsLine imprime uma linha de status no estilo "tarefa .... OK",
 * preenchendo o espaço entre leftText e rightText com pontos até atingir
 * uma largura fixa. A cor de rightText muda conforme success.
 *
 * @param leftText string - Texto à esquerda (ex.: nome do arquivo/tarefa).
 * @param rightText string - Texto à direita (ex.: "CRIADO", "PENDENTE").
 * @param success bool - Define a cor de rightText (verde ou amarelo).
 */
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

/**
 * runNewProject cria a estrutura de diretórios de um novo projeto Kite
 * (controllers, models, middlewares, requests, migrations, seeders,
 * routes, views) e gera um main.go inicial já com uma rota de exemplo.
 *
 * @param args []string - args[0] deve ser o nome do projeto/diretório a criar.
 * @return error
 */
/**
 * runNewProject inicializa um novo projeto Kite completo: cria a
 * estrutura de diretórios (a mesma usada pelos comandos make:*), roda
 * "go mod init" para que o projeto seja um módulo Go válido desde já,
 * adiciona o Kite como dependência via "go get" e gera um main.go que
 * já compila, importando o pacote real do framework.
 *
 * BUG CORRIGIDO: a versão anterior gerava um main.go que importava
 * "%s/core" — um pacote LOCAL que nunca existia dentro do projeto — e
 * chamava core.NewApp(), uma função que não existe no Kite (o construtor
 * real é kite.New()). Também não criava go.mod nenhum. Ou seja, todo
 * projeto gerado por "kite new" nascia sem compilar.
 *
 * @param args []string - args[0] é o nome do diretório OU um module path completo (ex.: "github.com/seu-usuario/minha-app").
 * @return error
 */
func runNewProject(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome do projeto. Ex: kite new minha-app (ou kite new github.com/seu-usuario/minha-app)")
	}

	modulePath := args[0]
	dirName := lastPathSegment(modulePath)

	fmt.Printf("\n%s Criando estrutura do projeto [%s]...\n\n", BadgeInfo, dirName)

	dirs := []string{
		dirName,
		filepath.Join(dirName, "app", "controllers"),
		filepath.Join(dirName, "app", "models"),
		filepath.Join(dirName, "app", "middlewares"),
		filepath.Join(dirName, "app", "requests"),
		filepath.Join(dirName, "config"),
		filepath.Join(dirName, "database", "migrations"),
		filepath.Join(dirName, "database", "seeders"),
		filepath.Join(dirName, "routes"),
		filepath.Join(dirName, "views"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("falha ao criar diretório %s: %w", dir, err)
		}
	}
	printDotsLine("Estrutura de diretórios", "CRIADA", true)

	if err := initGoModule(dirName, modulePath); err != nil {
		return err
	}
	printDotsLine("go.mod", "CRIADO", true)

	if err := addKiteDependency(dirName); err != nil {
		// Não é fatal: o projeto já fica utilizável, só falta rodar
		// "go get github.com/claudiovictors/kite" manualmente depois
		// (provavelmente por falta de acesso à rede neste momento).
		printDotsLine("go get github.com/claudiovictors/kite", "PULADO", false)
		fmt.Printf("  %s%v%s\n", FgYellow, err, Reset)
	} else {
		printDotsLine("Dependência do Kite", "ADICIONADA", true)
	}

	mainContent := fmt.Sprintf(`package main

import (
	"log"

	kite "github.com/claudiovictors/kite/core"
)

func main() {
	app := kite.New()

	app.Get("/", func(req kite.Request, res kite.Response) error {
		return res.Json(kite.Map{
			"app":    %q,
			"status": "online",
		})
	})

	log.Println("⚡ Servidor Kite rodando na porta :8080")
	if err := app.Listen(":8080"); err != nil {
		log.Fatal(err)
	}
}
`, dirName)

	if err := os.WriteFile(filepath.Join(dirName, "main.go"), []byte(mainContent), 0644); err != nil {
		return fmt.Errorf("falha ao criar main.go: %w", err)
	}
	printDotsLine("Arquivo main.go", "CRIADO", true)

	fmt.Printf("\n%s Projeto criado com sucesso. Digite %scd %s%s para começar.\n\n", BadgeSuccess, Bold, dirName, Reset)
	return nil
}

/**
 * lastPathSegment devolve o último segmento de um module path (o trecho
 * depois da última "/"), usado como nome do diretório do projeto. Se
 * modulePath não tiver "/", devolve o próprio valor sem alterações.
 *
 * Exemplo: "github.com/seu-usuario/minha-app" -> "minha-app"
 *
 * @param modulePath string
 * @return string
 */
func lastPathSegment(modulePath string) string {
	trimmed := strings.TrimRight(modulePath, "/")
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}

/**
 * initGoModule roda "go mod init <modulePath>" dentro do diretório do
 * projeto recém-criado, tornando-o um módulo Go válido imediatamente —
 * sem isso, o main.go gerado não tinha como compilar (não existia
 * go.mod nenhum no projeto).
 *
 * @param dir string - Diretório do projeto já criado.
 * @param modulePath string - Module path a gravar no go.mod.
 * @return error
 */
func initGoModule(dir, modulePath string) error {
	cmd := exec.Command("go", "mod", "init", modulePath)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("falha ao rodar 'go mod init %s': %w\n%s", modulePath, err, output)
	}
	return nil
}

/**
 * addKiteDependency roda "go get github.com/claudiovictors/kite" dentro
 * do diretório do projeto, já que o main.go gerado importa
 * "github.com/claudiovictors/kite/core" e precisa dessa dependência
 * registrada no go.mod/go.sum para compilar. Depende de acesso à rede;
 * se falhar, o erro é devolvido para ser reportado como aviso (não é
 * motivo para abortar a criação do projeto).
 *
 * @param dir string - Diretório do projeto.
 * @return error
 */
func addKiteDependency(dir string) error {
	cmd := exec.Command("go", "get", "github.com/claudiovictors/kite")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, output)
	}
	return nil
}

/**
 * runMakeController gera um novo arquivo de controller em
 * app/controllers/<nome>_controller.go, com um método Index de exemplo.
 *
 * @param args []string - args[0] deve ser o nome do controller (ex.: "User").
 * @return error
 */
func runMakeController(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome do controller. Ex: kite make:controller User")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + "_controller.go"
	path := filepath.Join("app", "controllers", fileName)

	content := fmt.Sprintf(`package controllers

import (
	"github.com/claudiovictors/kite/core"
)

type %sController struct{}

func (c *%sController) Index(req core.Request, res core.Response) error {
	return res.Json(map[string]string{"message": "Lista de %s"})
}
`, name, name, name)

	return generateFileWithArtisanOutput("Controller", path, content)
}

/**
 * runMakeMiddleware gera um novo arquivo de middleware em
 * app/middlewares/<nome>.go, já no formato core.MiddlewareFunc esperado
 * por app.Use/Route.Middleware.
 *
 * @param args []string - args[0] deve ser o nome do middleware (ex.: "Auth").
 * @return error
 */
func runMakeMiddleware(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome do middleware. Ex: kite make:middleware Auth")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + ".go"
	path := filepath.Join("app", "middlewares", fileName)

	content := fmt.Sprintf(`package middlewares

import (
	"github.com/claudiovictors/kite/core"
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

/**
 * runMakeModel gera um novo arquivo de model em app/models/<nome>.go,
 * com os campos ID, CreatedAt e UpdatedAt já mapeados via tags db/json.
 *
 * @param args []string - args[0] deve ser o nome da model (ex.: "Product").
 * @return error
 */
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
	ID        uint      `+"`json:\"id\" db:\"id\"`"+`
	CreatedAt time.Time `+"`json:\"created_at\" db:\"created_at\"`"+`
	UpdatedAt time.Time `+"`json:\"updated_at\" db:\"updated_at\"`"+`
}
`, name)

	return generateFileWithArtisanOutput("Model", path, content)
}

/**
 * runMakeMigration gera um arquivo de migration em
 * database/migrations/<timestamp>_<nome>.go, prefixado por timestamp para
 * garantir a ordem de execução. O conteúdo gerado é uma database.Migration
 * de verdade (Up/Down recebendo *database.Schema), pronta para ser
 * registrada num database.Migrator sem precisar editar a assinatura.
 *
 * @param args []string - args[0] deve ser o nome da tabela/migration (ex.: "create_users_table").
 * @return error
 */
func runMakeMigration(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome da tabela. Ex: kite make:migration create_users_table")
	}

	rawName := strings.ToLower(args[0])
	timestamp := time.Now().Format("20060102150405")
	migrationName := fmt.Sprintf("%s_%s", timestamp, rawName)
	fileName := migrationName + ".go"
	path := filepath.Join("database", "migrations", fileName)

	structName := strings.ReplaceAll(strings.Title(strings.ReplaceAll(rawName, "_", " ")), " ", "")
	tableName := strings.TrimPrefix(rawName, "create_")
	tableName = strings.TrimSuffix(tableName, "_table")
	if tableName == "" {
		tableName = rawName
	}

	content := fmt.Sprintf(`package migrations

import "github.com/claudiovictors/kite/database"

/**
 * %s cria a tabela "%s". Ajuste as colunas conforme o que sua aplicação
 * precisa e registre esta migration num database.Migrator no seu main():
 *
 *	migrator.Register(migrations.%s)
 */
var %s = database.Migration{
	Name: %q,
	Up: func(schema *database.Schema) error {
		return schema.Create(%q, func(t *database.Blueprint) {
			t.ID()
			t.Timestamps()
		})
	},
	Down: func(schema *database.Schema) error {
		return schema.DropIfExists(%q)
	},
}
`, structName, tableName, structName, structName, migrationName, tableName, tableName)

	return generateFileWithArtisanOutput("Migration", path, content)
}

/**
 * runMakeSeeder gera um arquivo de seeder em
 * database/seeders/<nome>_seeder.go, já implementando database.Seeder
 * (método Run(db *database.DB) error) — pronto para ser registrado num
 * database.SeederRunner.
 *
 * @param args []string - args[0] deve ser o nome do seeder (ex.: "Users").
 * @return error
 */
func runMakeSeeder(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome do seeder. Ex: kite make:seeder Users")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + "_seeder.go"
	path := filepath.Join("database", "seeders", fileName)

	content := fmt.Sprintf(`package seeders

import (
	"github.com/claudiovictors/kite/database"
)

/**
 * %sSeeder popula a tabela correspondente com dados iniciais/de teste.
 * Registre-o num database.SeederRunner e chame seeder.Run() (via
 * database.RunCLI ou manualmente no seu main()).
 */
type %sSeeder struct{}

func (s *%sSeeder) Run(db *database.DB) error {
	// Exemplo:
	//
	// user := &models.User{Name: "Admin", Email: "admin@example.com"}
	// return database.Create(db, user)

	return nil
}
`, name, name, name)

	return generateFileWithArtisanOutput("Seeder", path, content)
}

/**
 * runMakeRequest gera um arquivo de validação de request em
 * app/requests/<nome>_request.go, no estilo FormRequest do Laravel,
 * usando o pacote validation para as regras.
 *
 * @param args []string - args[0] deve ser o nome do request (ex.: "CreateUser").
 * @return error
 */
func runMakeRequest(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome do request. Ex: kite make:request CreateUser")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + "_request.go"
	path := filepath.Join("app", "requests", fileName)

	content := fmt.Sprintf(`package requests

import (
	"github.com/claudiovictors/kite/core"
	"github.com/claudiovictors/kite/validation"
)

/**
 * %sRequest concentra as regras de validação de entrada para essa ação,
 * no estilo do FormRequest do Laravel. Chame Validate(req) no início do
 * handler e devolva 422 se Fails() for true.
 */
type %sRequest struct{}

/**
 * Rules define as regras de validação por campo. Ajuste conforme os
 * campos reais esperados pela sua rota.
 *
 * @return map[string]string
 */
func (r *%sRequest) Rules() map[string]string {
	return map[string]string{
		// "nome":  "required|min:3",
		// "email": "required|email",
	}
}

/**
 * Validate roda o validador sobre os dados de entrada da requisição
 * (JSON, formulário ou query string, via req.All()).
 *
 * @param req core.Request
 * @return *validation.Validator
 */
func (r *%sRequest) Validate(req core.Request) *validation.Validator {
	return validation.Make(req.All(), r.Rules())
}
`, name, name, name, name)

	return generateFileWithArtisanOutput("Request", path, content)
}

/**
 * runMakeTest gera um esqueleto de teste HTTP para um controller em
 * app/controllers/<nome>_test.go, usando httptest para simular a
 * requisição sem precisar subir um servidor real.
 *
 * @param args []string - args[0] deve ser o nome do controller (ex.: "User").
 * @return error
 */
func runMakeTest(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("informe o nome do teste. Ex: kite make:test User")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + "_test.go"
	path := filepath.Join("app", "controllers", fileName)
	route := strings.ToLower(name)

	content := fmt.Sprintf(`package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/claudiovictors/kite/core"
)

/**
 * TestKite%sIndex é um esqueleto de teste HTTP para o controller %s.
 * Ajuste a rota, o método e as asserções conforme o handler real.
 */
func TestKite%sIndex(t *testing.T) {
	app := core.NewApp()
	controller := &%sController{}
	app.Get("/%s", controller.Index)

	req := httptest.NewRequest(http.MethodGet, "/%s", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava status 200, recebeu %%d", rec.Code)
	}
}
`, name, name, name, name, route, route)

	return generateFileWithArtisanOutput("Test", path, content)
}

/**
 * runMigrate executa as migrações pendentes da aplicação atual. A CLI
 * global não tem acesso à conexão do banco nem às Migration registradas em
 * memória — só o binário da sua aplicação tem. Por isso este comando só
 * repassa a chamada via runAppCommand, que roda "go run . migrate" no
 * diretório atual; a execução de verdade acontece dentro do seu main(),
 * através de database.RunCLI (veja kite/database/cli.go).
 *
 * @param args []string
 * @return error
 */
func runMigrate(args []string) error {
	return runAppCommand("migrate", args)
}

/**
 * runMigrateStatus mostra o estado (executada/pendente) de cada migração
 * registrada na aplicação atual. Veja runMigrate para o porquê do repasse.
 *
 * @param args []string
 * @return error
 */
func runMigrateStatus(args []string) error {
	return runAppCommand("migrate:status", args)
}

/**
 * runMigrateRollback reverte o último lote de migrações da aplicação atual
 * (ou os últimos N lotes, se um número for passado como argumento). Veja
 * runMigrate para o porquê do repasse.
 *
 * @param args []string
 * @return error
 */
func runMigrateRollback(args []string) error {
	return runAppCommand("migrate:rollback", args)
}

/**
 * runSeed executa os seeders registrados na aplicação atual, através de
 * database.SeederRunner. Veja runMigrate para o porquê do repasse.
 *
 * @param args []string
 * @return error
 */
func runSeed(args []string) error {
	return runAppCommand("seed", args)
}

/**
 * runAppCommand roda a aplicação atual com "go run . <command> [args...]",
 * repassando stdout/stderr/stdin diretamente para o terminal. A aplicação
 * precisa chamar database.RunCLI(os.Args[1:], migrator, seeder) no início
 * do seu main() para que o comando seja reconhecido e executado de verdade
 * — sem isso, "go run ." simplesmente sobe o servidor normalmente e ignora
 * o argumento extra.
 *
 * @param command string - Nome do comando reconhecido por database.RunCLI.
 * @param args []string - Argumentos extras (ex.: número de steps do rollback).
 * @return error
 */
func runAppCommand(command string, args []string) error {
	fmt.Printf("\n%s Repassando para a aplicação: go run . %s\n\n", BadgeInfo, strings.TrimSpace(command+" "+strings.Join(args, " ")))

	cmdArgs := append([]string{"run", "."}, append([]string{command}, args...)...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falha ao executar %q na aplicação: %w (você chamou database.RunCLI(os.Args[1:], ...) no seu main()?)", command, err)
	}
	return nil
}

/**
 * runServe roda a aplicação atual com "go run .", repassando
 * stdout/stderr/stdin diretamente para o terminal (equivalente a rodar o
 * comando na mão, mas com uma saída de boot consistente com o resto da
 * CLI). Um argumento opcional de porta é exposto via variável de
 * ambiente PORT, caso sua aplicação leia kite.Env("PORT", "8080") no boot.
 *
 * @param args []string - args[0], se presente, define a porta (via env PORT).
 * @return error
 */
func runServe(args []string) error {
	port := "8080"
	if len(args) > 0 {
		port = args[0]
	}

	fmt.Printf("\n%s Iniciando aplicação com 'go run .' (PORT=%s)...\n\n", BadgeInfo, port)

	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(), "PORT="+port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falha ao rodar a aplicação: %w", err)
	}
	return nil
}

/**
 * generateFileWithArtisanOutput cria os diretórios necessários e grava um
 * novo arquivo com o conteúdo informado, recusando sobrescrever um
 * arquivo já existente, e imprime uma saída de status no estilo Artisan.
 *
 * @param resourceType string - Nome exibido do tipo de recurso (ex.: "Controller").
 * @param path string - Caminho completo do arquivo a criar.
 * @param content string - Conteúdo a ser escrito no arquivo.
 * @return error
 */
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