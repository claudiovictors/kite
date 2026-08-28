package database

import (
	"errors"
	"fmt"
	"reflect"
)

/**
 * ErrNotFound é retornado por FindOrFail quando nenhum registro correspondente ao ID é encontrado.
 */
var ErrNotFound = errors.New("orm: registro não encontrado")

/**
 * Find busca um registro do tipo T pela sua chave primária "id".
 *
 * Retorna o ponteiro do elemento se encontrado, ou nil (sem erro) caso não exista.
 *
 * Exemplo:
 *  user, err := database.Find[User](db, 1)
 *
 * @tparam T tipo da struct do modelo.
 * @param db *DB
 * @param id interface{}
 * @return (*T, error)
 */
func Find[T any](db *DB, id interface{}) (*T, error) {
	return Query[T](db).Where("id", id).First()
}

/**
 * FindOrFail busca um registro do tipo T pelo seu "id" e retorna ErrNotFound se o registro não for localizado.
 *
 * Exemplo:
 *  user, err := database.FindOrFail[User](db, 1)
 *
 * @tparam T tipo da struct do modelo.
 * @param db *DB
 * @param id interface{}
 * @return (*T, error)
 */
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

/**
 * Raw executa uma instrução SQL arbitrária (raw) e mapeia automaticamente os resultados para uma slice de structs T.
 *
 * Exemplo:
 *  users, err := database.Raw[User](db, "SELECT * FROM users WHERE status = ? AND age > ?", "active", 18)
 *
 * @tparam T tipo da struct do modelo.
 * @param db *DB
 * @param query string
 * @param args ...interface{}
 * @return ([]T, error)
 */
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

/**
 * HasMany resolve um relacionamento 1:N buscando todos os registros de R que possuem a chave estrangeira igual ao ID do pai.
 *
 * Exemplo:
 *  posts, err := database.HasMany[Post](db, "user_id", user.ID)
 *
 * @tparam R tipo da struct do modelo filho (relacionado).
 * @param db *DB
 * @param foreignKey string
 * @param parentID interface{}
 * @return ([]R, error)
 */
func HasMany[R any](db *DB, foreignKey string, parentID interface{}) ([]R, error) {
	return Query[R](db).Where(foreignKey, parentID).Get()
}

/**
 * BelongsTo resolve um relacionamento N:1 buscando o registro do tipo T associado pelo valor da chave estrangeira.
 *
 * Exemplo:
 *  user, err := database.BelongsTo[User](db, post.UserID)
 *
 * @tparam T tipo da struct do modelo pai (relacionado).
 * @param db *DB
 * @param foreignKeyValue interface{}
 * @return (*T, error)
 */
func BelongsTo[T any](db *DB, foreignKeyValue interface{}) (*T, error) {
	return Find[T](db, foreignKeyValue)
}