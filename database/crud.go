package database

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

/**
 * ErrEmptyUpdate é retornado por Update ou QueryBuilder.Update quando o map de valores fornecido está vazio,
 * prevenindo a geração de uma instrução SQL UPDATE inválida.
 */
var ErrEmptyUpdate = errors.New("orm: nenhum campo informado para atualizar")

/**
 * Create insere um novo registro no banco de dados a partir do ponteiro da estrutura fornecida.
 *
 * Colunas identificadas como chave primária ("id") são ignoradas na instrução INSERT e preenchidas
 * automaticamente na struct via LastInsertId.
 *
 * Exemplo:
 *  user := &User{Name: "Ana", Email: "ana@ex.com"}
 *  err := database.Create(db, user)
 *  // user.ID é preenchido automaticamente após a inserção
 *
 * @param db *DB
 * @param data *T
 * @return error
 */
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

/**
 * Update atualiza todas as colunas mapeadas de data (exceto "id"), utilizando o valor de data.ID como cláusula WHERE.
 *
 * Exemplo:
 *  user.Name = "Ana Paula"
 *  err := database.Update(db, user)
 *
 * @param db *DB
 * @param data *T
 * @return error
 */
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

/**
 * Save alterna de forma transparente entre Create e Update inspecionando o campo ID de data.
 *
 * Se ID == 0, executa uma inserção (Create); caso contrário, executa uma atualização (Update).
 *
 * Exemplo:
 *  user := &User{Name: "Ana"}
 *  database.Save(db, user) // insere (ID igual a 0)
 *  user.Name = "Ana Paula"
 *  database.Save(db, user) // atualiza (ID maior que 0)
 *
 * @param db *DB
 * @param data *T
 * @return error
 */
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

/**
 * Delete remove a entrada correspondente ao tipo T cuja chave primária id coincida com o parâmetro informado.
 *
 * Exemplo:
 *  err := database.Delete[User](db, 42)
 *
 * @param db *DB
 * @param id interface{}
 * @return error
 */
func Delete[T any](db *DB, id interface{}) error {
	var zero T
	meta := db.registerModel(reflect.TypeOf(zero))

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", meta.tableName)
	if _, err := db.conn.Exec(query, id); err != nil {
		return fmt.Errorf("orm: erro ao deletar de %s: %w", meta.tableName, err)
	}
	return nil
}

/**
 * DeleteModel remove o registro representado pela instância data, extraindo o identificador diretamente do struct.
 *
 * Exemplo:
 *  err := database.DeleteModel(db, user)
 *
 * @param db *DB
 * @param data *T
 * @return error
 */
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

/**
 * fieldValueByColumn busca e retorna o reflect.Value correspondente ao campo associado à coluna informada.
 *
 * @param v reflect.Value
 * @param meta *modelMeta
 * @param column string
 * @return reflect.Value
 */
func fieldValueByColumn(v reflect.Value, meta *modelMeta, column string) reflect.Value {
	for _, f := range meta.fields {
		if f.column == column {
			return fieldValue(v, f)
		}
	}
	return reflect.Value{}
}

/* ---------------------------------------------------------------------- */
/* Update/Delete em massa via QueryBuilder                                */
/* ---------------------------------------------------------------------- */

/**
 * Update executa uma atualização em massa nos registros que satisfazem as condições registradas no QueryBuilder.
 *
 * Recebe um mapa de valores onde a chave representa a coluna da tabela. Retorna a quantidade de linhas afetadas.
 *
 * Exemplo:
 *  affected, err := database.Query[User](db).
 *      Where("active", false).
 *      Update(map[string]interface{}{"active": true})
 *
 * @param values map[string]interface{}
 * @return (int64, error)
 */
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

/**
 * Delete executa a remoção em massa de registros correspondentes aos critérios estabelecidos no QueryBuilder.
 *
 * Retorna a quantidade total de linhas removidas do banco de dados.
 *
 * Exemplo:
 *  affected, err := database.Query[Session](db).
 *      Where("expires_at", "<", time.Now()).
 *      Delete()
 *
 * @return (int64, error)
 */
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