package database

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ErrEmptyUpdate é retornado por Update/QueryBuilder.Update quando o
// map de valores vem vazio — evita gerar um "UPDATE table SET " inválido.
var ErrEmptyUpdate = errors.New("orm: nenhum campo informado para atualizar")

// Create insere um novo registro a partir de data, no estilo do
// Model::create() do Eloquent. Colunas geradas pelo banco (id
// autoincrement) são ignoradas no INSERT e preenchidas de volta em
// data.ID a partir do LastInsertId.
//
//	user := &User{Name: "Ana", Email: "ana@ex.com"}
//	err := database.Create(db, user)
//	// user.ID já vem preenchido depois disso
func Create[T any](db *DB, data *T) error {
	t := reflect.TypeOf(*data)
	meta := db.registerModel(t)
	v := reflect.ValueOf(data).Elem()

	var columns []string
	var placeholders []string
	var args []interface{}

	for _, f := range meta.fields {
		if f.column == "id" {
			continue // autoincrement: não entra no INSERT
		}
		columns = append(columns, f.column)
		placeholders = append(placeholders, "?")
		args = append(args, fieldValue(v, f).Interface())
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		meta.tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "),
	)

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("orm: erro ao inserir em %s: %w", meta.tableName, err)
	}

	if id, err := result.LastInsertId(); err == nil && id > 0 {
		idField := fieldValueByColumn(v, meta, "id")
		if idField.IsValid() && idField.CanSet() {
			idField.SetUint(uint64(id))
		}
	}

	return nil
}

// Update atualiza todas as colunas mapeadas de data (exceto "id"),
// usando data.ID como condição WHERE. Espelha o $model->save() do
// Eloquent quando o model já existe.
//
//	user.Name = "Ana Paula"
//	err := database.Update(db, user)
func Update[T any](db *DB, data *T) error {
	t := reflect.TypeOf(*data)
	meta := db.registerModel(t)
	v := reflect.ValueOf(data).Elem()

	var setClauses []string
	var args []interface{}
	var idValue interface{}

	for _, f := range meta.fields {
		if f.column == "id" {
			idValue = fieldValue(v, f).Interface()
			continue
		}
		setClauses = append(setClauses, f.column+" = ?")
		args = append(args, fieldValue(v, f).Interface())
	}

	if len(setClauses) == 0 {
		return ErrEmptyUpdate
	}

	args = append(args, idValue)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = ?",
		meta.tableName, strings.Join(setClauses, ", "),
	)

	if _, err := db.conn.Exec(query, args...); err != nil {
		return fmt.Errorf("orm: erro ao atualizar %s: %w", meta.tableName, err)
	}
	return nil
}

// Save decide entre Create e Update olhando pro ID de data: ID == 0
// insere um registro novo, ID != 0 atualiza o existente — igual o
// comportamento do $model->save() do Eloquent, sem precisar escolher
// manualmente qual função chamar.
//
//	user := &User{Name: "Ana"}
//	database.Save(db, user) // insere (ID era 0)
//	user.Name = "Ana Paula"
//	database.Save(db, user) // atualiza (ID já existe)
func Save[T any](db *DB, data *T) error {
	t := reflect.TypeOf(*data)
	meta := db.registerModel(t)
	v := reflect.ValueOf(data).Elem()

	id := fieldValueByColumn(v, meta, "id")
	if id.IsValid() && id.Uint() == 0 {
		return Create(db, data)
	}
	return Update(db, data)
}

// Delete remove o registro de tipo T cujo id bate com o informado.
//
//	err := database.Delete[User](db, 42)
func Delete[T any](db *DB, id interface{}) error {
	var zero T
	meta := db.registerModel(reflect.TypeOf(zero))

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", meta.tableName)
	if _, err := db.conn.Exec(query, id); err != nil {
		return fmt.Errorf("orm: erro ao deletar de %s: %w", meta.tableName, err)
	}
	return nil
}

// DeleteModel remove o registro representado por data, lendo o ID
// direto da instância — equivalente ao $model->delete() do Eloquent.
//
//	err := database.DeleteModel(db, user)
func DeleteModel[T any](db *DB, data *T) error {
	t := reflect.TypeOf(*data)
	meta := db.registerModel(t)
	v := reflect.ValueOf(data).Elem()

	id := fieldValueByColumn(v, meta, "id").Interface()
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", meta.tableName)
	if _, err := db.conn.Exec(query, id); err != nil {
		return fmt.Errorf("orm: erro ao deletar de %s: %w", meta.tableName, err)
	}
	return nil
}

// fieldValueByColumn localiza o reflect.Value de um campo pelo nome da
// coluna mapeada (ex: "id"), reaproveitando fieldValue.
func fieldValueByColumn(v reflect.Value, meta *modelMeta, column string) reflect.Value {
	for _, f := range meta.fields {
		if f.column == column {
			return fieldValue(v, f)
		}
	}
	return reflect.Value{}
}

// --- Update/Delete em massa via QueryBuilder --------------------------
//
// Equivalentes a User::where(...)->update([...]) e
// User::where(...)->delete() do Eloquent: aplicam sobre TODAS as linhas
// que baterem com os Where()/WhereIn() configurados no builder.

// Update roda um UPDATE em massa sobre as linhas que batem com os
// WHEREs do builder, usando os valores do map (chave = nome da coluna).
// Retorna quantas linhas foram afetadas.
//
//	affected, err := database.Query[User](db).
//	    Where("active", false).
//	    Update(map[string]interface{}{"active": true})
func (q *QueryBuilder[T]) Update(values map[string]interface{}) (int64, error) {
	if len(values) == 0 {
		return 0, ErrEmptyUpdate
	}

	setClauses := make([]string, 0, len(values))
	setArgs := make([]interface{}, 0, len(values))
	for column, value := range values {
		setClauses = append(setClauses, column+" = ?")
		setArgs = append(setArgs, value)
	}

	whereSQL, whereArgs := q.buildWhereAndJoins()
	// buildWhereAndJoins gera " FROM table ...", mas UPDATE não usa FROM.
	whereSQL = strings.Replace(whereSQL, " FROM "+q.meta.tableName, "", 1)

	query := fmt.Sprintf("UPDATE %s SET %s%s", q.meta.tableName, strings.Join(setClauses, ", "), whereSQL)
	args := append(setArgs, whereArgs...)

	result, err := q.db.conn.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("orm: erro ao atualizar em massa %q: %w", query, err)
	}
	return result.RowsAffected()
}

// Delete roda um DELETE em massa sobre as linhas que batem com os
// WHEREs do builder. Retorna quantas linhas foram removidas.
//
//	affected, err := database.Query[Session](db).
//	    Where("expires_at", "<", time.Now()).
//	    Delete()
func (q *QueryBuilder[T]) Delete() (int64, error) {
	whereSQL, args := q.buildWhereAndJoins()
	whereSQL = strings.Replace(whereSQL, " FROM "+q.meta.tableName, "", 1)

	query := fmt.Sprintf("DELETE FROM %s%s", q.meta.tableName, whereSQL)

	result, err := q.db.conn.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("orm: erro ao deletar em massa %q: %w", query, err)
	}
	return result.RowsAffected()
}