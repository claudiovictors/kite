package database

import (
	"errors"
	"fmt"
	"reflect"
)

// ErrNotFound é retornado por FindOrFail quando nenhum registro bate
// com o ID informado. Compare com errors.Is(err, database.ErrNotFound)
// pra devolver um 404 direto no handler.
var ErrNotFound = errors.New("orm: registro não encontrado")

// Find busca um registro pela coluna "id" (a chave primária padrão,
// vinda do campo embutido orm.Model). Retorna (nil, nil) quando não
// encontra nada — use FindOrFail se preferir tratar isso como erro.
//
//	user, err := database.Find[User](db, 42)
func Find[T any](db *DB, id interface{}) (*T, error) {
	return Query[T](db).Where("id", id).First()
}

// FindOrFail é igual a Find, mas devolve ErrNotFound em vez de
// (nil, nil) quando o registro não existe — evita checar nil na mão em
// handlers que só querem devolver 404:
//
//	user, err := database.FindOrFail[User](db, req.Param("id"))
//	if errors.Is(err, database.ErrNotFound) {
//	    return res.Status(404).WithJson(kite.ErrorResponse{Error: "usuário não encontrado", Status: 404})
//	}
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

// Raw executa uma query SQL arbitrária e faz o scan dos resultados em
// []T, pra casos que o QueryBuilder ainda não cobre (subqueries,
// agregações customizadas, UNIONs, etc).
//
// Importante: a ordem das colunas retornadas pela query precisa bater
// com a ordem dos campos mapeados em T. Prefira "SELECT *" ou liste as
// colunas na mesma ordem dos campos da struct.
//
//	users, err := database.Raw[User](db, "SELECT * FROM users WHERE age > ?", 18)
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

// --- Relacionamentos (v1: helpers manuais) --------------------------
//
// Ainda não há eager loading automático via tags/reflection (item 7 do
// roadmap, .With("Posts") pra evitar N+1). Por enquanto os
// relacionamentos são carregados explicitamente com estas duas funções.

// HasMany busca todos os registros de R cuja coluna foreignKey aponta
// pro ID informado. Uso típico — um Post que tem muitos Comment:
//
//	comments, err := database.HasMany[Comment](db, "post_id", post.ID)
func HasMany[R any](db *DB, foreignKey string, parentID interface{}) ([]R, error) {
	return Query[R](db).Where(foreignKey, parentID).Get()
}

// BelongsTo busca o registro "pai" de T a partir do valor de uma FK
// guardada no filho. Uso típico — um Comment que pertence a um Post:
//
//	post, err := database.BelongsTo[Post](db, comment.PostID)
func BelongsTo[T any](db *DB, foreignKeyValue interface{}) (*T, error) {
	return Find[T](db, foreignKeyValue)
}