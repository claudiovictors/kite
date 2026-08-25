// Package orm implementa um ORM estilo Eloquent (Active Record) sobre
// database/sql.
//
// Roadmap:
//  1. [feito]     Connect() + registro de metadados de Model via reflection
//  2. [feito]     QueryBuilder fluente (Where, OrWhere, WhereIn, Join, OrderBy, Limit/Offset, Get, First)
//  3. [feito]     Find/FindOrFail, Count, Exists, Paginate, Raw
//  4. [pendente]  Active Record: model.Save() / model.Delete()
//  5. [pendente]  Migrations (up/down, versionamento de schema)
//  6. [parcial]   Relacionamentos: HasMany/BelongsTo manuais prontos em
//                 relations.go; BelongsToMany e resolução automática via
//                 tags/reflection ainda pendente
//  7. [pendente]  Eager loading (.With("Posts")) pra evitar N+1
package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// DB envolve *sql.DB e mantém metadados de todos os models registrados.
type DB struct {
	conn *sql.DB

	mu       sync.RWMutex
	metadata map[reflect.Type]*modelMeta
}

// Connect abre a conexão usando o driver e DSN informados.
// O driver precisa estar importado em algum lugar do programa
// (ex: import _ "github.com/mattn/go-sqlite3").
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

// Model é a struct base que todo model do usuário deve embutir.
// Ex:
//
//	type User struct {
//	    orm.Model
//	    Name  string `db:"name"`
//	    Email string `db:"email"`
//	}
type Model struct {
	ID uint `db:"id"`
}

// modelMeta guarda os metadados extraídos via reflection de um model:
// nome da tabela, colunas mapeadas e o índice de cada campo na struct.
type modelMeta struct {
	tableName string
	fields    []fieldMeta
}

type fieldMeta struct {
	structIndex int
	column      string
}

// registerModel inspeciona a struct via reflection e monta o modelMeta,
// cacheando o resultado (a reflection só roda uma vez por tipo).
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

		// Campo embutido orm.Model: extrai a coluna "id" dele também.
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

// tableNameFor deriva o nome da tabela a partir do nome da struct,
// convertendo para snake_case e pluralizando de forma simples.
// Ex: User -> users, OrderItem -> order_items
func tableNameFor(t reflect.Type) string {
	name := toSnakeCase(t.Name())
	if strings.HasSuffix(name, "s") {
		return name
	}
	return name + "s"
}

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