package database

import (
	"fmt"
	"strconv"
)

/**
 * RunCLI inspeciona os argumentos de linha de comando da sua aplicação
 * (normalmente os.Args[1:]) e executa a ação de migração/seed
 * correspondente, se houver uma reconhecida. Existe porque só o binário da
 * sua aplicação tem a conexão de banco aberta e sabe quais Migration/Seeder
 * foram registrados em memória — a CLI global do Kite (cmd/kite) não tem
 * acesso a nada disso, então ela só repassa a chamada via "go run .".
 *
 * Uso recomendado, no topo do seu main(), antes de montar as rotas e
 * chamar app.Listen:
 *
 *	db, _ := database.Connect("mysql", dsn)
 *
 *	migrator := database.NewMigrator(db, database.MySQL)
 *	migrator.Register(migrations.All()...)
 *
 *	seeder := database.NewSeederRunner(db)
 *	seeder.Register(&seeders.UsersSeeder{})
 *
 *	if handled, err := database.RunCLI(os.Args[1:], migrator, seeder); handled {
 *	    if err != nil {
 *	        log.Fatal(err)
 *	    }
 *	    return // não sobe o servidor HTTP quando um comando de CLI foi tratado
 *	}
 *
 *	app := kite.New()
 *	// ... rotas ...
 *	app.Listen(":8080")
 *
 * A partir daí, os comandos abaixo funcionam de verdade quando chamados
 * pela CLI global (que só faz "go run . <comando>" por baixo dos panos):
 *
 *	kite migrate
 *	kite migrate:status
 *	kite migrate:rollback [steps]
 *	kite seed
 *
 * @param args []string - Tipicamente os.Args[1:] do seu main().
 * @param migrator *Migrator - Pode ser nil se sua aplicação não usa migrations.
 * @param seeder *SeederRunner - Pode ser nil se sua aplicação não usa seeders.
 * @return (handled bool, err error) - handled=true indica que um comando foi
 *         reconhecido e executado; nesse caso a aplicação deve encerrar sem
 *         subir o servidor HTTP.
 */
func RunCLI(args []string, migrator *Migrator, seeder *SeederRunner) (handled bool, err error) {
	if len(args) == 0 {
		return false, nil
	}

	switch args[0] {
	case "migrate":
		if migrator == nil {
			return true, fmt.Errorf("kite: nenhum Migrator configurado — passe um em database.RunCLI")
		}
		return true, migrator.Run()

	case "migrate:rollback":
		if migrator == nil {
			return true, fmt.Errorf("kite: nenhum Migrator configurado — passe um em database.RunCLI")
		}
		if len(args) > 1 {
			steps, convErr := strconv.Atoi(args[1])
			if convErr != nil {
				return true, fmt.Errorf("kite: número de steps inválido: %q", args[1])
			}
			return true, migrator.Rollback(steps)
		}
		return true, migrator.Rollback()

	case "migrate:status":
		if migrator == nil {
			return true, fmt.Errorf("kite: nenhum Migrator configurado — passe um em database.RunCLI")
		}
		statuses, statusErr := migrator.Status()
		if statusErr != nil {
			return true, statusErr
		}
		printMigrationStatuses(statuses)
		return true, nil

	case "seed":
		if seeder == nil {
			return true, fmt.Errorf("kite: nenhum SeederRunner configurado — passe um em database.RunCLI")
		}
		return true, seeder.Run()

	default:
		return false, nil
	}
}

/**
 * printMigrationStatuses imprime a lista de MigrationStatus num formato
 * legível de tabela simples, usado por RunCLI no comando "migrate:status".
 *
 * @param statuses []MigrationStatus
 */
func printMigrationStatuses(statuses []MigrationStatus) {
	if len(statuses) == 0 {
		fmt.Println("nenhuma migration registrada")
		return
	}

	for _, s := range statuses {
		state := "pendente"
		if s.Ran {
			state = fmt.Sprintf("executada (batch %d)", s.Batch)
		}
		fmt.Printf("  %-45s %s\n", s.Name, state)
	}
}