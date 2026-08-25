package database

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

// whereClause representa uma condição WHERE ("column op value"),
// combinada com a cláusula anterior via boolean ("AND"/"OR").
type whereClause struct {
	column  string
	op      string
	value   interface{}
	boolean string // "AND" ou "OR"
}

// joinClause representa um JOIN no FROM da query.
type joinClause struct {
	kind     string // "INNER" ou "LEFT"
	table    string
	first    string
	operator string
	second   string
}

// QueryBuilder monta e executa queries SELECT de forma fluente para o
// model T. Uso:
//
//	users, err := orm.Query[User](db).
//	    Where("age", ">", 18).
//	    OrderBy("name").
//	    Limit(10).
//	    Get()
//
// Where aceita omitir o operador quando for "=":
//
//	orm.Query[User](db).Where("email", email).First()
type QueryBuilder[T any] struct {
	db         *DB
	meta       *modelMeta
	wheres     []whereClause
	joins      []joinClause
	selectCols []string
	orderBy    string
	desc       bool
	limit      int
	hasLim     bool
	offset     int
	hasOffset  bool
}

// Query inicia uma query fluente para o model T sobre a conexão db.
func Query[T any](db *DB) *QueryBuilder[T] {
	var zero T
	t := reflect.TypeOf(zero)
	meta := db.registerModel(t)
	return &QueryBuilder[T]{db: db, meta: meta}
}

// parseWhereArgs normaliza os argumentos variádicos de Where/OrWhere:
//
//	parseWhereArgs([]interface{}{18})       -> "=", 18
//	parseWhereArgs([]interface{}{">=", 18}) -> ">=", 18
func parseWhereArgs(args []interface{}) (op string, value interface{}) {
	switch len(args) {
	case 1:
		return "=", args[0]
	case 2:
		if opStr, ok := args[0].(string); ok {
			return opStr, args[1]
		}
		return "=", args[0]
	default:
		return "=", nil
	}
}

// Where adiciona uma condição AND. Aceita duas formas:
//
//	Where("age", 18)       // vira "age = ?"  (operador "=" default)
//	Where("age", ">=", 18) // vira "age >= ?" (operador explícito)
func (q *QueryBuilder[T]) Where(column string, args ...interface{}) *QueryBuilder[T] {
	op, value := parseWhereArgs(args)
	q.wheres = append(q.wheres, whereClause{column: column, op: op, value: value, boolean: "AND"})
	return q
}

// OrWhere funciona como Where, mas combina com a cláusula anterior
// usando OR em vez de AND.
func (q *QueryBuilder[T]) OrWhere(column string, args ...interface{}) *QueryBuilder[T] {
	op, value := parseWhereArgs(args)
	q.wheres = append(q.wheres, whereClause{column: column, op: op, value: value, boolean: "OR"})
	return q
}

// WhereIn adiciona uma condição "column IN (?, ?, ...)".
func (q *QueryBuilder[T]) WhereIn(column string, values []interface{}) *QueryBuilder[T] {
	q.wheres = append(q.wheres, whereClause{column: column, op: "IN", value: values, boolean: "AND"})
	return q
}

// Join adiciona um INNER JOIN. Ex:
//
//	Join("posts", "users.id", "=", "posts.user_id")
func (q *QueryBuilder[T]) Join(table, first, operator, second string) *QueryBuilder[T] {
	q.joins = append(q.joins, joinClause{kind: "INNER", table: table, first: first, operator: operator, second: second})
	return q
}

// LeftJoin adiciona um LEFT JOIN, mesma assinatura de Join.
func (q *QueryBuilder[T]) LeftJoin(table, first, operator, second string) *QueryBuilder[T] {
	q.joins = append(q.joins, joinClause{kind: "LEFT", table: table, first: first, operator: operator, second: second})
	return q
}

// Select sobrescreve as colunas retornadas (por padrão, todas as
// colunas mapeadas do model). Útil com Join, pra evitar ambiguidade
// entre colunas repetidas em tabelas diferentes:
//
//	orm.Query[User](db).
//	    Join("posts", "users.id", "=", "posts.user_id").
//	    Select("users.id", "users.name", "posts.title").
//	    Get()
func (q *QueryBuilder[T]) Select(columns ...string) *QueryBuilder[T] {
	q.selectCols = columns
	return q
}

// OrderBy define a coluna de ordenação (ascendente por padrão).
func (q *QueryBuilder[T]) OrderBy(column string) *QueryBuilder[T] {
	q.orderBy = column
	q.desc = false
	return q
}

// OrderByDesc define a coluna de ordenação descendente.
func (q *QueryBuilder[T]) OrderByDesc(column string) *QueryBuilder[T] {
	q.orderBy = column
	q.desc = true
	return q
}

// Limit define o número máximo de linhas retornadas.
func (q *QueryBuilder[T]) Limit(n int) *QueryBuilder[T] {
	q.limit = n
	q.hasLim = true
	return q
}

// Offset define quantas linhas pular antes de retornar resultados.
// Paginate() já cuida disso automaticamente; use direto só se precisar
// de paginação manual.
func (q *QueryBuilder[T]) Offset(n int) *QueryBuilder[T] {
	q.offset = n
	q.hasOffset = true
	return q
}

// columns retorna as colunas efetivas do SELECT: as escolhidas via
// Select(), ou todas as colunas mapeadas do model por padrão.
func (q *QueryBuilder[T]) columns() []string {
	if len(q.selectCols) > 0 {
		return q.selectCols
	}
	cols := make([]string, len(q.meta.fields))
	for i, f := range q.meta.fields {
		cols[i] = f.column
	}
	return cols
}

// buildWhereAndJoins monta o trecho compartilhado entre SELECT e COUNT
// (FROM + JOINs + WHERE) e os argumentos posicionais correspondentes.
func (q *QueryBuilder[T]) buildWhereAndJoins() (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}

	fmt.Fprintf(&sb, " FROM %s", q.meta.tableName)

	for _, j := range q.joins {
		fmt.Fprintf(&sb, " %s JOIN %s ON %s %s %s", j.kind, j.table, j.first, j.operator, j.second)
	}

	if len(q.wheres) > 0 {
		sb.WriteString(" WHERE ")
		for i, w := range q.wheres {
			if i > 0 {
				fmt.Fprintf(&sb, " %s ", w.boolean)
			}
			if w.op == "IN" {
				values, _ := w.value.([]interface{})
				placeholders := make([]string, len(values))
				for j, v := range values {
					placeholders[j] = "?"
					args = append(args, v)
				}
				fmt.Fprintf(&sb, "%s IN (%s)", w.column, strings.Join(placeholders, ", "))
			} else {
				fmt.Fprintf(&sb, "%s %s ?", w.column, w.op)
				args = append(args, w.value)
			}
		}
	}

	return sb.String(), args
}

// buildSelect monta o SQL final + os argumentos posicionais para
// database/sql (usa placeholders "?", ajustar para "$1" em Postgres
// numa camada de dialect na v2).
func (q *QueryBuilder[T]) buildSelect() (string, []interface{}) {
	whereSQL, args := q.buildWhereAndJoins()

	var sb strings.Builder
	fmt.Fprintf(&sb, "SELECT %s%s", strings.Join(q.columns(), ", "), whereSQL)

	if q.orderBy != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(q.orderBy)
		if q.desc {
			sb.WriteString(" DESC")
		}
	}

	if q.hasLim {
		fmt.Fprintf(&sb, " LIMIT %d", q.limit)
	}
	if q.hasOffset {
		fmt.Fprintf(&sb, " OFFSET %d", q.offset)
	}

	return sb.String(), args
}

// buildCount monta "SELECT COUNT(*) FROM ..." reaproveitando JOINs e
// WHEREs do builder (ORDER BY/LIMIT/OFFSET não fazem sentido aqui).
func (q *QueryBuilder[T]) buildCount() (string, []interface{}) {
	whereSQL, args := q.buildWhereAndJoins()
	return "SELECT COUNT(*)" + whereSQL, args
}

// Get executa a query e retorna todos os resultados como []T.
func (q *QueryBuilder[T]) Get() ([]T, error) {
	query, args := q.buildSelect()

	rows, err := q.db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("orm: erro ao executar query %q: %w", query, err)
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var item T
		if err := scanInto(&item, q.meta, rows); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

// First executa a query com LIMIT 1 e retorna o primeiro resultado
// (ou nil se não houver nenhum).
func (q *QueryBuilder[T]) First() (*T, error) {
	q.Limit(1)
	results, err := q.Get()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &results[0], nil
}

// Count retorna o número de registros que batem com os WHEREs/JOINs
// configurados.
func (q *QueryBuilder[T]) Count() (int64, error) {
	query, args := q.buildCount()

	var count int64
	if err := q.db.conn.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("orm: erro ao contar registros %q: %w", query, err)
	}
	return count, nil
}

// Exists retorna true se existir pelo menos um registro que bate com
// os WHEREs configurados (Count() > 0, com nome mais expressivo):
//
//	if orm.Query[User](db).Where("email", email).Exists() { ... }
func (q *QueryBuilder[T]) Exists() (bool, error) {
	count, err := q.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Paginator é o retorno de Paginate(): os dados da página atual mais
// os metadados prontos pra serializar em JSON na resposta da API.
type Paginator[T any] struct {
	Data     []T   `json:"data"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PerPage  int   `json:"per_page"`
	LastPage int   `json:"last_page"`
}

// Paginate calcula o total de registros (Count), aplica LIMIT/OFFSET
// de acordo com page/perPage, e devolve os dados + metadados prontos:
//
//	page, _ := strconv.Atoi(req.Query("page"))
//	result, err := orm.Query[User](db).OrderBy("name").Paginate(page, 20)
//	return res.Json(result)
func (q *QueryBuilder[T]) Paginate(page, perPage int) (*Paginator[T], error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}

	total, err := q.Count()
	if err != nil {
		return nil, err
	}

	q.Limit(perPage)
	q.Offset((page - 1) * perPage)

	data, err := q.Get()
	if err != nil {
		return nil, err
	}

	lastPage := int(math.Ceil(float64(total) / float64(perPage)))
	if lastPage < 1 {
		lastPage = 1
	}

	return &Paginator[T]{
		Data:     data,
		Total:    total,
		Page:     page,
		PerPage:  perPage,
		LastPage: lastPage,
	}, nil
}

// scanner é a interface mínima que *sql.Rows satisfaz, usada para
// permitir testes/mocks sem depender do tipo concreto.
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanInto usa reflection para popular os campos de dest (um ponteiro
// para struct model) a partir da linha atual do resultado.
func scanInto[T any](dest *T, meta *modelMeta, row scanner) error {
	v := reflect.ValueOf(dest).Elem()
	pointers := make([]interface{}, len(meta.fields))

	for i, f := range meta.fields {
		field := v.Field(f.structIndex)
		// Campo embutido orm.Model: aponta direto para o subcampo ID,
		// já que Model em si não implementa sql.Scanner.
		if field.Kind() == reflect.Struct && field.Type() == reflect.TypeOf(Model{}) {
			pointers[i] = field.FieldByName("ID").Addr().Interface()
			continue
		}
		pointers[i] = field.Addr().Interface()
	}

	return row.Scan(pointers...)
}