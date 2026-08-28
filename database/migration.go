package database

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

/**
 * Dialect identifica o SGBD alvo para adequação das sintaxes DDL, auto-incremento e tipos nativos.
 */
type Dialect string

const (
	MySQL    Dialect = "mysql"
	Postgres Dialect = "postgres"
	SQLite   Dialect = "sqlite"
)

/**
 * Migration representa uma única unidade de migração de banco de dados no estilo Schema Migrations do Laravel.
 *
 * Name deve ser único e preferencialmente prefixado com timestamp/sequencial para ordenação correta.
 */
type Migration struct {
	Name string
	Up   func(schema *Schema) error
	Down func(schema *Schema) error
}

/**
 * Schema fornece a API de nível superior para operações de DDL (Data Definition Language) de criação, alteração e remoção de tabelas.
 */
type Schema struct {
	db      *DB
	dialect Dialect
}

/**
 * NewSchema instancia o manipulador de Schema para o SGBD e dialeto informados.
 *
 * @param db *DB
 * @param dialect Dialect
 * @return *Schema
 */
func NewSchema(db *DB, dialect Dialect) *Schema {
	return &Schema{db: db, dialect: dialect}
}

/**
 * Create gera e executa a instrução SQL de criação de uma nova tabela com base nas definições do Blueprint.
 *
 * Exemplo:
 *  schema.Create("users", func(t *database.Blueprint) {
 *      t.ID()
 *      t.String("name", 100)
 *      t.String("email", 150).Unique()
 *      t.Boolean("active").Default("true")
 *      t.Timestamps()
 *  })
 *
 * @param table string
 * @param fn func(t *Blueprint)
 * @return error
 */
func (s *Schema) Create(table string, fn func(t *Blueprint)) error {
	bp := &Blueprint{table: table, dialect: s.dialect}
	fn(bp)

	sql := bp.buildCreate()
	if _, err := s.db.conn.Exec(sql); err != nil {
		return fmt.Errorf("migration: erro ao criar tabela %s: %w (sql: %s)", table, err, sql)
	}
	return nil
}

/**
 * Table altera a estrutura de uma tabela existente no banco de dados.
 *
 * Exemplo:
 *  schema.Table("users", func(t *database.Blueprint) {
 *      t.String("phone", 20).Nullable()
 *  })
 *
 * @param table string
 * @param fn func(t *Blueprint)
 * @return error
 */
func (s *Schema) Table(table string, fn func(t *Blueprint)) error {
	bp := &Blueprint{table: table, dialect: s.dialect, alter: true}
	fn(bp)

	for _, stmt := range bp.buildAlter() {
		if _, err := s.db.conn.Exec(stmt); err != nil {
			return fmt.Errorf("migration: erro ao alterar tabela %s: %w (sql: %s)", table, err, stmt)
		}
	}
	return nil
}

/**
 * Drop remove uma tabela do banco de dados, retornando um erro caso ela não exista.
 *
 * @param table string
 * @return error
 */
func (s *Schema) Drop(table string) error {
	_, err := s.db.conn.Exec("DROP TABLE " + table)
	if err != nil {
		return fmt.Errorf("migration: erro ao remover tabela %s: %w", table, err)
	}
	return nil
}

/**
 * DropIfExists remove uma tabela do banco de dados de forma segura, ignorando a operação se a tabela não existir.
 *
 * @param table string
 * @return error
 */
func (s *Schema) DropIfExists(table string) error {
	_, err := s.db.conn.Exec("DROP TABLE IF EXISTS " + table)
	if err != nil {
		return fmt.Errorf("migration: erro ao remover tabela %s: %w", table, err)
	}
	return nil
}

/* ---------------------------------------------------------------------- */
/* Blueprint: construtor de colunas, ao estilo do Blueprint do Laravel     */
/* ---------------------------------------------------------------------- */

/**
 * Blueprint acumula a lista de colunas e constraints a serem processadas na montagem das instruções SQL DDL.
 */
type Blueprint struct {
	table   string
	dialect Dialect
	alter   bool
	columns []*columnDef
}

/**
 * columnDef armazena as características, atributos e qualificadores de tipo de uma coluna de banco de dados.
 */
type columnDef struct {
	name       string
	sqlType    string
	nullable   bool
	unique     bool
	primaryKey bool
	hasDefault bool
	defaultVal string
	references string
}

func (b *Blueprint) addColumn(name, sqlType string) *columnDef {
	c := &columnDef{name: name, sqlType: sqlType}
	b.columns = append(b.columns, c)
	return c
}

/**
 * ID declara a coluna de chave primária autoincremental chamada "id" adaptada ao dialeto do SGBD ativo.
 *
 * @return *columnDef
 */
func (b *Blueprint) ID() *columnDef {
	sqlType := "BIGINT"
	switch b.dialect {
	case Postgres:
		sqlType = "BIGSERIAL"
	case SQLite:
		sqlType = "INTEGER"
	default: // MySQL
		sqlType = "BIGINT AUTO_INCREMENT"
	}
	c := b.addColumn("id", sqlType)
	c.primaryKey = true
	return c
}

/**
 * String adiciona uma coluna de tipo texto de tamanho variável (VARCHAR). Se omitido, o tamanho padrão é 255.
 *
 * @param name string
 * @param length ...int
 * @return *columnDef
 */
func (b *Blueprint) String(name string, length ...int) *columnDef {
	n := 255
	if len(length) > 0 {
		n = length[0]
	}
	return b.addColumn(name, fmt.Sprintf("VARCHAR(%d)", n))
}

/**
 * Text adiciona uma coluna de texto longo sem limitação estrita de comprimento (TEXT).
 *
 * @param name string
 * @return *columnDef
 */
func (b *Blueprint) Text(name string) *columnDef {
	return b.addColumn(name, "TEXT")
}

/**
 * Integer adiciona uma coluna de inteiros padrão (INT).
 *
 * @param name string
 * @return *columnDef
 */
func (b *Blueprint) Integer(name string) *columnDef {
	return b.addColumn(name, "INT")
}

/**
 * BigInteger adiciona uma coluna de inteiros de 64 bits (BIGINT).
 *
 * @param name string
 * @return *columnDef
 */
func (b *Blueprint) BigInteger(name string) *columnDef {
	return b.addColumn(name, "BIGINT")
}

/**
 * Float adiciona uma coluna de ponto flutuante de precisão dupla (DOUBLE PRECISION).
 *
 * @param name string
 * @return *columnDef
 */
func (b *Blueprint) Float(name string) *columnDef {
	return b.addColumn(name, "DOUBLE PRECISION")
}

/**
 * Decimal adiciona uma coluna de precisão exata DECIMAL(precision, scale) para valores monetários.
 *
 * @param name string
 * @param precision int
 * @param scale int
 * @return *columnDef
 */
func (b *Blueprint) Decimal(name string, precision, scale int) *columnDef {
	return b.addColumn(name, fmt.Sprintf("DECIMAL(%d,%d)", precision, scale))
}

/**
 * Boolean adiciona uma coluna do tipo booleano.
 *
 * @param name string
 * @return *columnDef
 */
func (b *Blueprint) Boolean(name string) *columnDef {
	sqlType := "BOOLEAN"
	if b.dialect == SQLite {
		sqlType = "INTEGER"
	}
	return b.addColumn(name, sqlType)
}

/**
 * Date adiciona uma coluna do tipo data (DATE).
 *
 * @param name string
 * @return *columnDef
 */
func (b *Blueprint) Date(name string) *columnDef {
	return b.addColumn(name, "DATE")
}

/**
 * Timestamp adiciona uma coluna do tipo data e hora (TIMESTAMP).
 *
 * @param name string
 * @return *columnDef
 */
func (b *Blueprint) Timestamp(name string) *columnDef {
	return b.addColumn(name, "TIMESTAMP")
}

/**
 * Timestamps adiciona as colunas padronizadas created_at e updated_at na tabela.
 */
func (b *Blueprint) Timestamps() {
	createdAt := b.addColumn("created_at", "TIMESTAMP")
	createdAt.hasDefault = true
	createdAt.defaultVal = "CURRENT_TIMESTAMP"

	b.addColumn("updated_at", "TIMESTAMP").Nullable()
}

/**
 * SoftDeletes adiciona a coluna anulável deleted_at destinada ao suporte de exclusão lógica.
 *
 * @return *columnDef
 */
func (b *Blueprint) SoftDeletes() *columnDef {
	return b.addColumn("deleted_at", "TIMESTAMP").Nullable()
}

/**
 * ForeignID declara uma coluna BIGINT para vínculo de chave estrangeira.
 *
 * Exemplo:
 *  t.ForeignID("user_id").References("id").On("users")
 *
 * @param name string
 * @return *columnDef
 */
func (b *Blueprint) ForeignID(name string) *columnDef {
	return b.addColumn(name, "BIGINT")
}

/**
 * References define o campo de destino da chave estrangeira. Deve ser utilizado em conjunto com On.
 *
 * @param column string
 * @return *columnDef
 */
func (c *columnDef) References(column string) *columnDef {
	c.references = column
	return c
}

/**
 * On define a tabela de destino associada à chave estrangeira.
 *
 * @param table string
 * @return *columnDef
 */
func (c *columnDef) On(table string) *columnDef {
	c.references = table + "(" + c.references + ")"
	return c
}

/**
 * Nullable configura a coluna para aceitar valores nulos (NULL).
 *
 * @return *columnDef
 */
func (c *columnDef) Nullable() *columnDef {
	c.nullable = true
	return c
}

/**
 * Unique adiciona a restrição de unicidade (UNIQUE) à coluna.
 *
 * @return *columnDef
 */
func (c *columnDef) Unique() *columnDef {
	c.unique = true
	return c
}

/**
 * Default estabelece um valor padrão a ser atribuído à coluna em operações de inserção.
 *
 * @param value string
 * @return *columnDef
 */
func (c *columnDef) Default(value string) *columnDef {
	c.hasDefault = true
	c.defaultVal = value
	return c
}

/* ---------------------------------------------------------------------- */
/* Geração de SQL                                                         */
/* ---------------------------------------------------------------------- */

func (c *columnDef) toSQL() string {
	var sb strings.Builder
	sb.WriteString(c.name)
	sb.WriteString(" ")
	sb.WriteString(c.sqlType)

	if c.primaryKey {
		sb.WriteString(" PRIMARY KEY")
	}
	if !c.nullable && !c.primaryKey {
		sb.WriteString(" NOT NULL")
	}
	if c.hasDefault {
		sb.WriteString(" DEFAULT ")
		sb.WriteString(c.defaultVal)
	}
	if c.unique {
		sb.WriteString(" UNIQUE")
	}
	return sb.String()
}

func (b *Blueprint) buildCreate() string {
	var parts []string
	var foreignKeys []string

	for _, c := range b.columns {
		parts = append(parts, c.toSQL())
		if c.references != "" {
			foreignKeys = append(foreignKeys, fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s", c.name, c.references))
		}
	}
	parts = append(parts, foreignKeys...)

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)", b.table, strings.Join(parts, ",\n  "))
}

func (b *Blueprint) buildAlter() []string {
	var stmts []string
	for _, c := range b.columns {
		stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", b.table, c.toSQL()))
	}
	return stmts
}

/* ---------------------------------------------------------------------- */
/* Migrator: orquestração de migrations                                   */
/* ---------------------------------------------------------------------- */

/**
 * Migrator gerencia e orquestra a execução sequencial de migrações, mantendo o histórico de lotes (batches) no banco de dados.
 */
type Migrator struct {
	db         *DB
	schema     *Schema
	dialect    Dialect
	migrations []Migration
}

/**
 * NewMigrator cria uma nova instância de Migrator para o dialeto de banco de dados indicado.
 *
 * @param db *DB
 * @param dialect Dialect
 * @return *Migrator
 */
func NewMigrator(db *DB, dialect Dialect) *Migrator {
	return &Migrator{db: db, schema: NewSchema(db, dialect), dialect: dialect}
}

/**
 * Register adiciona uma lista de migrações à fila de execução do Migrator.
 *
 * @param migrations ...Migration
 */
func (m *Migrator) Register(migrations ...Migration) {
	m.migrations = append(m.migrations, migrations...)
}

func (m *Migrator) ensureMigrationsTable() error {
	_, err := m.db.conn.Exec(`CREATE TABLE IF NOT EXISTS migrations (
        id ` + m.autoIncrementType() + ` PRIMARY KEY,
        name VARCHAR(255) NOT NULL UNIQUE,
        batch INT NOT NULL,
        executed_at TIMESTAMP NOT NULL
    )`)
	return err
}

func (m *Migrator) autoIncrementType() string {
	switch m.dialect {
	case Postgres:
		return "BIGSERIAL"
	case SQLite:
		return "INTEGER"
	default:
		return "BIGINT AUTO_INCREMENT"
	}
}

func (m *Migrator) executedNames() (map[string]bool, error) {
	rows, err := m.db.conn.Query("SELECT name FROM migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	done := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		done[name] = true
	}
	return done, rows.Err()
}

func (m *Migrator) lastBatch() (int, error) {
	var batch *int
	err := m.db.conn.QueryRow("SELECT MAX(batch) FROM migrations").Scan(&batch)
	if err != nil {
		return 0, err
	}
	if batch == nil {
		return 0, nil
	}
	return *batch, nil
}

/**
 * Run executa todas as migrações registradas que ainda não foram aplicadas, agrupando-as sob um novo número de lote (batch).
 *
 * @return error
 */
func (m *Migrator) Run() error {
	if err := m.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("migration: erro ao preparar tabela de controle: %w", err)
	}

	done, err := m.executedNames()
	if err != nil {
		return fmt.Errorf("migration: erro ao ler migrations executadas: %w", err)
	}

	batch, err := m.lastBatch()
	if err != nil {
		return fmt.Errorf("migration: erro ao ler última batch: %w", err)
	}
	batch++

	ran := 0
	for _, mig := range m.migrations {
		if done[mig.Name] {
			continue
		}

		fmt.Printf("migrating: %s\n", mig.Name)
		if err := mig.Up(m.schema); err != nil {
			return fmt.Errorf("migration: falha ao rodar %q: %w", mig.Name, err)
		}

		_, err := m.db.conn.Exec(
			"INSERT INTO migrations (name, batch, executed_at) VALUES (?, ?, ?)",
			mig.Name, batch, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("migration: falha ao registrar %q: %w", mig.Name, err)
		}
		fmt.Printf("migrated:  %s\n", mig.Name)
		ran++
	}

	if ran == 0 {
		fmt.Println("nenhuma migration pendente")
	}
	return nil
}

/**
 * Rollback reverte a execução das migrações aplicadas no último lote (ou na quantidade de lotes especificada em steps).
 *
 * @param steps ...int
 * @return error
 */
func (m *Migrator) Rollback(steps ...int) error {
	n := 1
	if len(steps) > 0 {
		n = steps[0]
	}

	byName := make(map[string]Migration, len(m.migrations))
	for _, mig := range m.migrations {
		byName[mig.Name] = mig
	}

	for i := 0; i < n; i++ {
		batch, err := m.lastBatch()
		if err != nil {
			return err
		}
		if batch == 0 {
			fmt.Println("nada para reverter")
			return nil
		}

		rows, err := m.db.conn.Query("SELECT name FROM migrations WHERE batch = ? ORDER BY id DESC", batch)
		if err != nil {
			return err
		}
		var names []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				rows.Close()
				return err
			}
			names = append(names, name)
		}
		rows.Close()

		for _, name := range names {
			mig, ok := byName[name]
			if !ok {
				return fmt.Errorf("migration: %q está no banco mas não foi registrada no código", name)
			}
			fmt.Printf("rolling back: %s\n", name)
			if err := mig.Down(m.schema); err != nil {
				return fmt.Errorf("migration: falha ao reverter %q: %w", name, err)
			}
			if _, err := m.db.conn.Exec("DELETE FROM migrations WHERE name = ?", name); err != nil {
				return err
			}
			fmt.Printf("rolled back:  %s\n", name)
		}
	}
	return nil
}

/**
 * MigrationStatus representa o estado individual de uma migração registrada.
 */
type MigrationStatus struct {
	Name  string
	Ran   bool
	Batch int
}

/**
 * Status retorna uma lista contendo o estado atual de execução de cada migração cadastrada no sistema.
 *
 * @return ([]MigrationStatus, error)
 */
func (m *Migrator) Status() ([]MigrationStatus, error) {
	if err := m.ensureMigrationsTable(); err != nil {
		return nil, err
	}

	rows, err := m.db.conn.Query("SELECT name, batch FROM migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	batches := map[string]int{}
	for rows.Next() {
		var name string
		var batch int
		if err := rows.Scan(&name, &batch); err != nil {
			return nil, err
		}
		batches[name] = batch
	}

	var out []MigrationStatus
	for _, mig := range m.migrations {
		batch, ran := batches[mig.Name]
		out = append(out, MigrationStatus{Name: mig.Name, Ran: ran, Batch: batch})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

/**
 * parseTimestampPrefix extrai o prefixo numérico (timestamp) e o nome legível de uma migração.
 *
 * Exemplo: "20260101120000_create_users_table" -> (20260101120000, "create_users_table")
 *
 * @param name string
 * @return (int64, string)
 */
func parseTimestampPrefix(name string) (int64, string) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return 0, name
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, name
	}
	return ts, parts[1]
}