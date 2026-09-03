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
 * Command represents a CLI-registered command: its invocation name,
 * the usage line shown in help, a short description and the function
 * executed when the command is called.
 *
 * @property {string} Name - Name used on the command line (eg: "make:model").
 * @property {string} Usage - Example usage shown on help.
 * @property {string} Description - Short description shown in the command list.
 * @property {func(args []string) error} Run - Function executed with the arguments after the command name.
 */
type Command struct {
	Name        string
	Usage       string
	Description string
	Run         func(args []string) error
}

var commands = map[string]Command{}

/**
 * RegisterCommand adds a Command to the global registry of available commands,
 * indexed by its Name.
 *
 * @param cmd Command - Command to be registered.
 */
func RegisterCommand(cmd Command) {
	commands[cmd.Name] = cmd
}

/**
 * main is the CLI entrypoint: it registers all commands, resolves global flags
 * (-v/--version, -h/--help), finds the Command by os.Args[1] and runs its Run
 * with the remaining arguments.
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
		fmt.Printf("\n%s The command %q was not found.\n\n", BadgeError, commandName)
		printHelp()
		os.Exit(1)
	}

	if err := cmd.Run(os.Args[2:]); err != nil {
		fmt.Printf("\n%s %v\n\n", BadgeError, err)
		os.Exit(1)
	}
}

/**
 * registerAllCommands centralizes registration of all known CLI commands.
 * Called once at the start of main().
 */
func registerAllCommands() {
	RegisterCommand(Command{
		Name:        "new",
		Usage:       "kite new <project-name>",
		Description: "Initializes a new Kite project structure.",
		Run:         runNewProject,
	})

	RegisterCommand(Command{
		Name:        "make:controller",
		Usage:       "kite make:controller <Name>",
		Description: "Generates a new controller/handler file.",
		Run:         runMakeController,
	})

	RegisterCommand(Command{
		Name:        "make:middleware",
		Usage:       "kite make:middleware <Name>",
		Description: "Generates a new middleware scaffold.",
		Run:         runMakeMiddleware,
	})

	RegisterCommand(Command{
		Name:        "make:model",
		Usage:       "kite make:model <Name>",
		Description: "Generates a data model for the ORM/Database.",
		Run:         runMakeModel,
	})

	RegisterCommand(Command{
		Name:        "make:migration",
		Usage:       "kite make:migration <table_name>",
		Description: "Generates a new database migration file.",
		Run:         runMakeMigration,
	})

	RegisterCommand(Command{
		Name:        "make:seeder",
		Usage:       "kite make:seeder <Name>",
		Description: "Generates a new database seeder file.",
		Run:         runMakeSeeder,
	})

	RegisterCommand(Command{
		Name:        "make:request",
		Usage:       "kite make:request <Name>",
		Description: "Generates a request validation class (FormRequest style).",
		Run:         runMakeRequest,
	})

	RegisterCommand(Command{
		Name:        "make:test",
		Usage:       "kite make:test <Name>",
		Description: "Generates an HTTP test skeleton for a controller.",
		Run:         runMakeTest,
	})

	RegisterCommand(Command{
		Name:        "migrate",
		Usage:       "kite migrate",
		Description: "Runs pending migrations for the current application (via database.RunCLI).",
		Run:         runMigrate,
	})

	RegisterCommand(Command{
		Name:        "migrate:status",
		Usage:       "kite migrate:status",
		Description: "Shows the status (ran/pending) of each registered migration.",
		Run:         runMigrateStatus,
	})

	RegisterCommand(Command{
		Name:        "migrate:rollback",
		Usage:       "kite migrate:rollback [steps]",
		Description: "Rolls back the last batch of migrations (or the last N batches).",
		Run:         runMigrateRollback,
	})

	RegisterCommand(Command{
		Name:        "seed",
		Usage:       "kite seed",
		Description: "Runs the seeders registered in the current application (via database.RunCLI).",
		Run:         runSeed,
	})

	RegisterCommand(Command{
		Name:        "serve",
		Usage:       "kite serve [port]",
		Description: "Runs the current application with 'go run .' (equivalent to running it manually).",
		Run:         runServe,
	})
}

/**
 * printHelp prints the CLI banner, the list of registered commands
 * (aligned in columns via tabwriter) and the global available options.
 */
func printHelp() {
	fmt.Print(Banner)
	fmt.Printf("\n%sUsage:%s\n  kite <command> [arguments]\n\n", Bold, Reset)
	fmt.Printf("%sAvailable commands:%s\n", Bold, Reset)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, cmd := range commands {
		fmt.Fprintf(w, "  %s%s%s\t%s%s%s\n", FgGreen, cmd.Name, Reset, FgGray, cmd.Description, Reset)
	}
	w.Flush()

	fmt.Printf("\n%sOptions:%s\n", Bold, Reset)
	fmt.Printf("  %s-v, --version%s    Shows the current CLI version.\n", FgGreen, Reset)
	fmt.Printf("  %s-h, --help%s       Shows this help menu.\n\n", FgGreen, Reset)
}

/**
 * printDotsLine prints a status line like "task .... OK",
 * filling the space between leftText and rightText with dots until a fixed width.
 * The color of rightText changes according to success.
 *
 * @param leftText string - Left text (eg: filename/task name).
 * @param rightText string - Right text (eg: "CREATED", "PENDING").
 * @param success bool - Defines the color of rightText (green or yellow).
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
// Command implementations
// ----------------------------------------------------------------------

/**
 * runNewProject creates the directory structure of a new Kite project
 * (controllers, models, middlewares, requests, migrations, seeders,
 * routes, views) and generates a starter main.go with an example route.
 *
 * runNewProject also runs "go mod init" so the project is a valid Go module,
 * attempts to add the Kite dependency via "go get", and writes a main.go that
 * imports the framework package.
 *
 * Note: args[0] may be either the directory name or a full module path
 * (eg: "github.com/your-user/my-app").
 *
 * @param args []string - args[0] is the directory name OR a module path.
 * @return error
 */
func runNewProject(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide the project name. Ex: kite new my-app (or kite new github.com/your-user/my-app)")
	}

	modulePath := args[0]
	dirName := lastPathSegment(modulePath)

	fmt.Printf("\n%s Creating project structure [%s]...\n\n", BadgeInfo, dirName)

	dirs := []string{
		dirName,
		filepath.Join(dirName, "controllers"),
		filepath.Join(dirName, "models"),
		filepath.Join(dirName, "database"),
		filepath.Join(dirName, "views"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	printDotsLine("Directory structure", "CREATED", true)

	if err := initGoModule(dirName, modulePath); err != nil {
		return err
	}
	printDotsLine("go.mod", "CREATED", true)

	if err := addKiteDependency(dirName); err != nil {
		// Not fatal: the project is still usable, it only requires running
		// "go get github.com/claudiovictors/kite" manually later (maybe no network).
		printDotsLine("go get github.com/claudiovictors/kite", "SKIPPED", false)
		fmt.Printf("  %s%v%s\n", FgYellow, err, Reset)
	} else {
		printDotsLine("Kite dependency", "ADDED", true)
	}

	mainContent := `package main

import (
	"log"

	kite "github.com/claudiovictors/kite/core"
)

func main() {
	app := kite.New()
	app.Use(kite.CORS())

	app.Get("/", func(req kite.Request, res kite.Response) error {
		return res.Send("Hello, World!")
	})

	log.Fatal(app.Listen(":3000"))
}
`

	if err := os.WriteFile(filepath.Join(dirName, "main.go"), []byte(mainContent), 0644); err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}
	printDotsLine("File main.go", "CREATED", true)

	fmt.Printf("\n%s Project created successfully. Type %scd %s%s to get started.\n\n", BadgeSuccess, Bold, dirName, Reset)
	return nil
}

/**
 * lastPathSegment returns the last segment of a module path (the part
 * after the last "/"), used as the project directory name. If the
 * modulePath has no "/", returns the value unchanged.
 *
 * Example: "github.com/your-user/my-app" -> "my-app"
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
 * initGoModule runs "go mod init <modulePath>" inside the newly-created
 * project directory, turning it into a valid Go module.
 *
 * @param dir string - Project directory.
 * @param modulePath string - Module path to write in go.mod.
 * @return error
 */
func initGoModule(dir, modulePath string) error {
	cmd := exec.Command("go", "mod", "init", modulePath)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run 'go mod init %s': %w\n%s", modulePath, err, output)
	}
	return nil
}

/**
 * addKiteDependency runs "go get github.com/claudiovictors/kite" inside
 * the project directory, since the generated main.go imports
 * "github.com/claudiovictors/kite/core" and needs the dependency registered.
 * This requires network access; if it fails the error is returned so the
 * caller can show a warning (creation is not aborted).
 *
 * @param dir string - Project directory.
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
 * runMakeController generates a new controller file in controllers/<name>_controller.go
 * with an example Index method.
 *
 * @param args []string - args[0] should be the controller name (eg: "User").
 * @return error
 */
func runMakeController(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide the controller name. Ex: kite make:controller User")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + "_controller.go"
	path := filepath.Join("controllers", fileName)

	content := fmt.Sprintf(`package controllers

import (
	"github.com/claudiovictors/kite/core"
)

type %sController struct{}

func (c *%sController) Index(req core.Request, res core.Response) error {
	return res.Json(map[string]string{"message": "List of %s"})
}
`, name, name, name)

	return generateFileWithArtisanOutput("Controller", path, content)
}

/**
 * runMakeMiddleware generates a new middleware file in middlewares/<name>.go,
 * returning core.MiddlewareFunc expected by app.Use/Route.Middleware.
 *
 * @param args []string - args[0] should be the middleware name (eg: "Auth").
 * @return error
 */
func runMakeMiddleware(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide the middleware name. Ex: kite make:middleware Auth")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + ".go"
	path := filepath.Join("middlewares", fileName)

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
 * runMakeModel generates a new model file in models/<name>.go,
 * with ID, CreatedAt and UpdatedAt fields already mapped via tags db/json.
 *
 * @param args []string - args[0] should be the model name (eg: "Product").
 * @return error
 */
func runMakeModel(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide the model name. Ex: kite make:model Product")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + ".go"
	path := filepath.Join("models", fileName)

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
 * runMakeMigration generates a migration file in
 * database/migrations/<timestamp>_<name>.go, prefixed by a timestamp to
 * guarantee execution order. The generated content is a real
 * database.Migration (Up/Down receiving *database.Schema), ready to be
 * registered in a database.Migrator without changing the signature.
 *
 * @param args []string - args[0] should be the table/migration name (eg: "create_users_table").
 * @return error
 */
func runMakeMigration(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide the table name. Ex: kite make:migration create_users_table")
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
 * %s creates the table "%s". Adjust columns as your application needs
 * and register this migration in a database.Migrator in your main():
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
 * runMakeSeeder generates a seeder file in database/seeders/<name>_seeder.go,
 * already implementing database.Seeder (Run(db *database.DB) error).
 *
 * @param args []string - args[0] should be the seeder name (eg: "Users").
 * @return error
 */
func runMakeSeeder(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide the seeder name. Ex: kite make:seeder Users")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + "_seeder.go"
	path := filepath.Join("database", "seeders", fileName)

	content := fmt.Sprintf(`package seeders

import (
	"github.com/claudiovictors/kite/database"
)

/**
 * %sSeeder populates the corresponding table with initial/test data.
 * Register it in a database.SeederRunner and call seeder.Run() (via
 * database.RunCLI or manually in your main()).
 */
type %sSeeder struct{}

func (s *%sSeeder) Run(db *database.DB) error {
	// Example:
	//
	// user := &models.User{Name: "Admin", Email: "admin@example.com"}
	// return database.Create(db, user)

	return nil
}
`, name, name, name)

	return generateFileWithArtisanOutput("Seeder", path, content)
}

/**
 * runMakeRequest generates a request validation file in
 * requests/<name>_request.go, FormRequest-style, using the validation package.
 *
 * @param args []string - args[0] should be the request name (eg: "CreateUser").
 * @return error
 */
func runMakeRequest(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide the request name. Ex: kite make:request CreateUser")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + "_request.go"
	path := filepath.Join("requests", fileName)

	content := fmt.Sprintf(`package requests

import (
	"github.com/claudiovictors/kite/core"
	"github.com/claudiovictors/kite/validation"
)

/**
 * %sRequest concentrates the validation rules for this action,
 * in Laravel FormRequest style. Call Validate(req) at the start of the
 * handler and return 422 if Fails() is true.
 */
type %sRequest struct{}

/**
 * Rules defines validation rules per field. Adjust for the real
 * fields expected by your route.
 *
 * @return map[string]string
 */
func (r *%sRequest) Rules() map[string]string {
	return map[string]string{
		// "name":  "required|min:3",
		// "email": "required|email",
	}
}

/**
 * Validate runs the validator over the request input
 * (JSON, form or query string, via req.All()).
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
 * runMakeTest generates an HTTP test skeleton for a controller in
 * controllers/<name>_test.go, using httptest to simulate the request.
 *
 * @param args []string - args[0] should be the controller name (eg: "User").
 * @return error
 */
func runMakeTest(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide the test name. Ex: kite make:test User")
	}

	name := strings.Title(strings.ToLower(args[0]))
	fileName := strings.ToLower(name) + "_test.go"
	path := filepath.Join("controllers", fileName)
	route := strings.ToLower(name)

	content := fmt.Sprintf(`package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/claudiovictors/kite/core"
)

/**
 * TestKite%sIndex is an HTTP test skeleton for the %s controller.
 * Adjust the route, method and assertions according to the real handler.
 */
func TestKite%sIndex(t *testing.T) {
	app := core.NewApp()
	controller := &%sController{}
	app.Get("/%s", controller.Index)

	req := httptest.NewRequest(http.MethodGet, "/%s", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %%d", rec.Code)
	}
}
`, name, name, name, name, route, route)

	return generateFileWithArtisanOutput("Test", path, content)
}

/**
 * runMigrate executes pending migrations for the current application. The
 * global CLI has no access to the DB connection nor to migrations registered
 * in memory — only your application's binary has. Therefore this command
 * forwards the call via runAppCommand, which runs "go run . migrate".
 *
 * @param args []string
 * @return error
 */
func runMigrate(args []string) error {
	return runAppCommand("migrate", args)
}

/**
 * runMigrateStatus shows the status (ran/pending) of each migration
 * registered in the current application. See runMigrate for why this is forwarded.
 *
 * @param args []string
 * @return error
 */
func runMigrateStatus(args []string) error {
	return runAppCommand("migrate:status", args)
}

/**
 * runMigrateRollback rolls back the last batch of migrations in the current
 * application (or the last N batches, if a number is provided). See runMigrate.
 *
 * @param args []string
 * @return error
 */
func runMigrateRollback(args []string) error {
	return runAppCommand("migrate:rollback", args)
}

/**
 * runSeed executes seeders registered in the current application via
 * database.SeederRunner. See runMigrate for the forwarding rationale.
 *
 * @param args []string
 * @return error
 */
func runSeed(args []string) error {
	return runAppCommand("seed", args)
}

/**
 * runAppCommand runs the current application with "go run . <command> [args...]",
 * forwarding stdout/stderr/stdin directly to the terminal. The application
 * must call database.RunCLI(os.Args[1:], migrator, seeder) in its main() so
 * the command is recognized and executed — otherwise "go run ." will simply
 * start the server and ignore the extra argument.
 *
 * @param command string - Name of the command recognized by database.RunCLI.
 * @param args []string - Extra arguments (eg: rollback steps).
 * @return error
 */
func runAppCommand(command string, args []string) error {
	fmt.Printf("\n%s Forwarding to the application: go run . %s\n\n", BadgeInfo, strings.TrimSpace(command+" "+strings.Join(args, " ")))

	cmdArgs := append([]string{"run", "."}, append([]string{command}, args...)...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %q in the application: %w (did you call database.RunCLI(os.Args[1:], ...) in your main()?)", command, err)
	}
	return nil
}

/**
 * runServe runs the current application with "go run .", forwarding
 * stdout/stderr/stdin directly to the terminal (equivalent to running
 * the command manually). An optional port argument is exposed via the
 * PORT environment variable, in case your app reads kite.Env("PORT", "8080") on boot.
 *
 * @param args []string - args[0], if present, sets the port (via env PORT).
 * @return error
 */
func runServe(args []string) error {
	port := "8080"
	if len(args) > 0 {
		port = args[0]
	}

	fmt.Printf("\n%s Starting application with 'go run .' (PORT=%s)...\n\n", BadgeInfo, port)

	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(), "PORT="+port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run the application: %w", err)
	}
	return nil
}

/**
 * generateFileWithArtisanOutput creates necessary directories and writes a
 * new file with the provided content, refusing to overwrite an existing file,
 * and prints Artisan-style status output.
 *
 * @param resourceType string - Display name of the resource (eg: "Controller").
 * @param path string - Full path of the file to create.
 * @param content string - Content to write to the file.
 * @return error
 */
func generateFileWithArtisanOutput(resourceType string, path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("the file [%s] already exists", path)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("error writing file [%s]: %w", path, err)
	}

	fmt.Println()
	fmt.Printf("%s Created %s [%s].\n", BadgeInfo, resourceType, path)
	fmt.Println()
	printDotsLine(path, "DONE", true)
	fmt.Println()

	return nil
}