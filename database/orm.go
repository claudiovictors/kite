package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

/**
 * DB representa a instância central do ORM, mantendo o pool de conexões com o banco de dados
 * e o cache thread-safe dos metadados das structs/models registradas.
 */
type DB struct {
	conn *sql.DB

	mu       sync.RWMutex
	metadata map[reflect.Type]*modelMeta
}

/**
 * Connect estabelece a conexão com o banco de dados utilizando o driver e a DSN (Data Source Name) fornecidos.
 *
 * Executa um Ping() inicial para garantir que a conexão física está ativa e responsiva.
 *
 * @param driver string
 * @param dsn string
 * @return (*DB, error)
 */
func Connect(driver, dsn string) (*DB, error) {
	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("orm: falha ao abrir conexão: %w", err)
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("orm: falha ao conectar: %w", err)
	}
	return &DB{conn: conn, metadata: make(map[reflect.Type]*modelMeta)}, nil
}

/**
 * Model é a struct base embutida (embedded) que fornece a chave primária padrão "id" para os modelos do ORM.
 */
type Model struct {
	ID uint `db:"id"`
}

/**
 * modelMeta armazena as informações mapeadas por reflexão de um modelo Go para otimizar a montagem de queries.
 */
type modelMeta struct {
	tableName string
	fields    []fieldMeta
}

/**
 * fieldMeta associa o índice de um campo na struct Go ao seu respectivo nome de coluna na tabela SQL.
 */
type fieldMeta struct {
	structIndex int
	column      string
}

/**
 * registerModel inspeciona o reflect.Type da struct informada e armazena em cache o mapeamento de campos e tabela.
 *
 * Utiliza RLock/Lock para garantir acesso seguro concorrente e evitar inspeções repetidas via reflexão.
 *
 * @param t reflect.Type
 * @return *modelMeta
 */
func (db *DB) registerModel(t reflect.Type) *modelMeta {
	db.mu.RLock()
	if meta, ok := db.metadata[t]; ok {
		db.mu.RUnlock()
		return meta
	}
	db.mu.RUnlock()

	meta := &modelMeta{tableName: tableNameFor(t)}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Anonymous && field.Type == reflect.TypeOf(Model{}) {
			meta.fields = append(meta.fields, fieldMeta{structIndex: i, column: "id"})
			continue
		}

		tag := field.Tag.Get("db")
		if tag == "-" {
			continue
		}
		column := tag
		if column == "" {
			column = toSnakeCase(field.Name)
		}
		meta.fields = append(meta.fields, fieldMeta{structIndex: i, column: column})
	}

	db.mu.Lock()
	db.metadata[t] = meta
	db.mu.Unlock()

	return meta
}

/**
 * tableNameFor converte o nome da struct Go em snake_case e aplica a regra simples de pluralização ("s").
 *
 * @param t reflect.Type
 * @return string
 */
func tableNameFor(t reflect.Type) string {
	name := toSnakeCase(t.Name())
	if strings.HasSuffix(name, "s") {
		return name
	}
	return name + "s"
}

/**
 * toSnakeCase converte uma string no formato PascalCase ou camelCase para a convenção snake_case SQL.
 *
 * Exemplo: "FirstName" -> "first_name"
 *
 * @param s string
 * @return string
 */
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}