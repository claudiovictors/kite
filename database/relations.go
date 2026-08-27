package database

import (
	"errors"
	"fmt"
	"reflect"
)

var ErrNotFound = errors.New("orm: registro não encontrado")

func Find[T any](db *DB, id interface{}) (*T, error) {
	return Query[T](db).Where("id", id).First()
}

func FindOrFail[T any](db *DB, id interface{}) (*T, error) {
	result, err := Find[T](db, id)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrNotFound
	}
	return result, nil
}

func Raw[T any](db *DB, query string, args ...interface{}) ([]T, error) {
	var zero T
	meta := db.registerModel(reflect.TypeOf(zero))

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("orm: erro ao executar query raw %q: %w", query, err)
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var item T
		if err := scanInto(&item, meta, rows); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func HasMany[R any](db *DB, foreignKey string, parentID interface{}) ([]R, error) {
	return Query[R](db).Where(foreignKey, parentID).Get()
}

func BelongsTo[T any](db *DB, foreignKeyValue interface{}) (*T, error) {
	return Find[T](db, foreignKeyValue)
}