package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type DB struct {
	conn *sql.DB

	mu       sync.RWMutex
	metadata map[reflect.Type]*modelMeta
}

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

type Model struct {
	ID uint `db:"id"`
}

type modelMeta struct {
	tableName string
	fields    []fieldMeta
}

type fieldMeta struct {
	structIndex int
	column      string
}

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