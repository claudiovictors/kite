package database

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

/**
 * whereClause representa uma condição WHERE individual na instrução SQL.
 */
type whereClause struct {
	column  string
	op      string
	value   interface{}
	boolean string
}

/**
 * joinClause representa uma cláusula de junção de tabelas (JOIN / LEFT JOIN).
 */
type joinClause struct {
	kind     string
	table    string
	first    string
	operator string
	second   string
}

/**
 * QueryBuilder provê uma interface fluente e genericamente tipada para construção e execução de consultas SQL.
 *
 * @tparam T Tipo da struct que representa o modelo no banco de dados.
 */
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

/**
 * Query inicializa um novo QueryBuilder fortemente tipado para o modelo informado.
 *
 * Exemplo:
 *  users, err := database.Query[User](db).Where("active", true).Get()
 *
 * @tparam T tipo da struct do modelo.
 * @param db *DB
 * @return *QueryBuilder[T]
 */
func Query[T any](db *DB) *QueryBuilder[T] {
	var zero T
	t := reflect.TypeOf(zero)
	meta := db.registerModel(t)
	return &QueryBuilder[T]{db: db, meta: meta}
}

/**
 * parseWhereArgs analisa os argumentos variádicos para permitir sintaxes simplificadas como Where("age", 18) ou Where("age", ">=", 18).
 *
 * @param args []interface{}
 * @return (string, interface{})
 */
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

/**
 * Where adiciona uma condição AND WHERE à consulta.
 *
 * Exemplos:
 *  q.Where("status", "active")
 *  q.Where("age", ">=", 18)
 *
 * @param column string
 * @param args ...interface{}
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) Where(column string, args ...interface{}) *QueryBuilder[T] {
	op, value := parseWhereArgs(args)
	q.wheres = append(q.wheres, whereClause{column: column, op: op, value: value, boolean: "AND"})
	return q
}

/**
 * OrWhere adiciona uma condição OR WHERE à consulta.
 *
 * @param column string
 * @param args ...interface{}
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) OrWhere(column string, args ...interface{}) *QueryBuilder[T] {
	op, value := parseWhereArgs(args)
	q.wheres = append(q.wheres, whereClause{column: column, op: op, value: value, boolean: "OR"})
	return q
}

/**
 * WhereIn adiciona uma restrição WHERE column IN (...) à consulta.
 *
 * @param column string
 * @param values []interface{}
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) WhereIn(column string, values []interface{}) *QueryBuilder[T] {
	q.wheres = append(q.wheres, whereClause{column: column, op: "IN", value: values, boolean: "AND"})
	return q
}

/**
 * Join adiciona uma junção interna (INNER JOIN) à consulta.
 *
 * @param table string
 * @param first string
 * @param operator string
 * @param second string
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) Join(table, first, operator, second string) *QueryBuilder[T] {
	q.joins = append(q.joins, joinClause{kind: "INNER", table: table, first: first, operator: operator, second: second})
	return q
}

/**
 * LeftJoin adiciona uma junção à esquerda (LEFT JOIN) à consulta.
 *
 * @param table string
 * @param first string
 * @param operator string
 * @param second string
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) LeftJoin(table, first, operator, second string) *QueryBuilder[T] {
	q.joins = append(q.joins, joinClause{kind: "LEFT", table: table, first: first, operator: operator, second: second})
	return q
}

/**
 * Select projeta colunas específicas na instrução SELECT. Se omitido, todas as colunas do modelo são projetadas.
 *
 * @param columns ...string
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) Select(columns ...string) *QueryBuilder[T] {
	q.selectCols = columns
	return q
}

/**
 * OrderBy define a ordenação ascendente (ASC) do resultado por uma coluna específica.
 *
 * @param column string
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) OrderBy(column string) *QueryBuilder[T] {
	q.orderBy = column
	q.desc = false
	return q
}

/**
 * OrderByDesc define a ordenação descendente (DESC) do resultado por uma coluna específica.
 *
 * @param column string
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) OrderByDesc(column string) *QueryBuilder[T] {
	q.orderBy = column
	q.desc = true
	return q
}

/**
 * Limit estabelece o número máximo de registros retornados (LIMIT).
 *
 * @param n int
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) Limit(n int) *QueryBuilder[T] {
	q.limit = n
	q.hasLim = true
	return q
}

/**
 * Offset define o deslocamento de registros na consulta (OFFSET).
 *
 * @param n int
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) Offset(n int) *QueryBuilder[T] {
	q.offset = n
	q.hasOffset = true
	return q
}

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

func (q *QueryBuilder[T]) buildCount() (string, []interface{}) {
	whereSQL, args := q.buildWhereAndJoins()
	return "SELECT COUNT(*)" + whereSQL, args
}

/**
 * Get executa a consulta SELECT montada e retorna uma slice contendo todas as instâncias encontradas do tipo T.
 *
 * @return ([]T, error)
 */
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

/**
 * First executa a consulta limitando o resultado a 1 registro e retorna o ponteiro para a primeira instância encontrada ou nil.
 *
 * @return (*T, error)
 */
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

/**
 * Count retorna a quantidade total de registros que satisfazem os critérios de filtro da consulta.
 *
 * @return (int64, error)
 */
func (q *QueryBuilder[T]) Count() (int64, error) {
	query, args := q.buildCount()

	var count int64
	if err := q.db.conn.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("orm: erro ao contar registros %q: %w", query, err)
	}
	return count, nil
}

/**
 * Exists verifica se existe pelo menos um registro correspondente aos filtros da consulta.
 *
 * @return (bool, error)
 */
func (q *QueryBuilder[T]) Exists() (bool, error) {
	count, err := q.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

/**
 * Paginator contém a estrutura de resposta para resultados paginados, incluindo os dados e metadados de paginação.
 */
type Paginator[T any] struct {
	Data     []T   `json:"data"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PerPage  int   `json:"per_page"`
	LastPage int   `json:"last_page"`
}

/**
 * Paginate executa a consulta paginada para a página e limite por página especificados.
 *
 * @param page int
 * @param perPage int
 * @return (*Paginator[T], error)
 */
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

type scanner interface {
	Scan(dest ...interface{}) error
}

/**
 * fieldValue retorna o reflect.Value endereçável do campo correspondente a fieldMeta em v.
 * Trata o caso do campo embutido orm.Model apontando direto para o subcampo ID.
 *
 * @param v reflect.Value
 * @param f fieldMeta
 * @return reflect.Value
 */
func fieldValue(v reflect.Value, f fieldMeta) reflect.Value {
	field := v.Field(f.structIndex)
	if field.Kind() == reflect.Struct && field.Type() == reflect.TypeOf(Model{}) {
		return field.FieldByName("ID")
	}
	return field
}

/**
 * scanInto preenche iterativamente os ponteiros dos campos da struct de destino utilizando o scanner das linhas do banco.
 *
 * @tparam T tipo da struct do modelo.
 * @param dest *T
 * @param meta *modelMeta
 * @param row scanner
 * @return error
 */
func scanInto[T any](dest *T, meta *modelMeta, row scanner) error {
	v := reflect.ValueOf(dest).Elem()
	pointers := make([]interface{}, len(meta.fields))

	for i, f := range meta.fields {
		pointers[i] = fieldValue(v, f).Addr().Interface()
	}

	return row.Scan(pointers...)
}