package database

import "fmt"

/**
 * Seeder é a interface que qualquer seeder deve implementar, no estilo do
 * Seeder do Laravel: recebe a conexão já aberta e popula dados iniciais
 * ou de teste na tabela correspondente.
 */
type Seeder interface {
	Run(db *DB) error
}

/**
 * NamedSeeder é implementado opcionalmente por seeders que queiram um nome
 * amigável na saída do SeederRunner. Quando um Seeder não implementa esta
 * interface, o SeederRunner usa o nome do tipo em Go (via %T) como fallback.
 */
type NamedSeeder interface {
	Seeder
	Name() string
}

/**
 * SeederRunner orquestra a execução de uma lista de seeders registrados, na
 * ordem em que foram adicionados — equivalente ao DatabaseSeeder do Laravel
 * (o "call([...])" dentro do run()).
 */
type SeederRunner struct {
	db      *DB
	seeders []Seeder
}

/**
 * NewSeederRunner cria um SeederRunner associado à conexão informada.
 *
 * @param db *DB
 * @return *SeederRunner
 */
func NewSeederRunner(db *DB) *SeederRunner {
	return &SeederRunner{db: db}
}

/**
 * Register adiciona um ou mais seeders à fila de execução. A ordem de
 * registro é a ordem de execução em Run().
 *
 *	seeder := database.NewSeederRunner(db)
 *	seeder.Register(&seeders.RolesSeeder{}, &seeders.UsersSeeder{})
 *
 * @param seeders ...Seeder
 */
func (r *SeederRunner) Register(seeders ...Seeder) {
	r.seeders = append(r.seeders, seeders...)
}

/**
 * Run executa, em ordem, todos os seeders registrados. Interrompe e
 * devolve o erro assim que um seeder falhar (os seeders anteriores já
 * rodaram e não são desfeitos automaticamente).
 *
 * @return error
 */
func (r *SeederRunner) Run() error {
	if len(r.seeders) == 0 {
		fmt.Println("nenhum seeder registrado")
		return nil
	}

	for _, s := range r.seeders {
		name := seederName(s)
		fmt.Printf("seeding: %s\n", name)
		if err := s.Run(r.db); err != nil {
			return fmt.Errorf("seeder: falha ao rodar %q: %w", name, err)
		}
		fmt.Printf("seeded:  %s\n", name)
	}
	return nil
}

/**
 * seederName resolve o nome de exibição de um seeder: usa Name() quando o
 * seeder implementa NamedSeeder, senão cai para o nome do tipo Go (%T).
 *
 * @param s Seeder
 * @return string
 */
func seederName(s Seeder) string {
	if n, ok := s.(NamedSeeder); ok {
		return n.Name()
	}
	return fmt.Sprintf("%T", s)
}