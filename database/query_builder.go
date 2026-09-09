package database

import (
	"database/sql"
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
	op      string // "=", ">", "IN", "NULL", "NOT NULL", "BETWEEN", "NOT IN"
	value   interface{}
	value2  interface{}
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
	groupBy    []string
	havings    []whereClause
	distinct   bool
	orderBy    string
	desc       bool
	limit      int
	hasLim     bool
	offset     int
	hasOffset  bool
}

/**
 * All é um alias explícito de Get, no estilo Model::all() do Eloquent.
 * Semanticamente idêntico a Get(), existe apenas para deixar o código
 * mais legível quando não há filtros aplicados.
 *
 * Exemplo:
 *  users, err := database.Query[User](db).All()
 *
 * @return ([]T, error)
 */
func (q *QueryBuilder[T]) All() ([]T, error) {
	return q.Get()
}

/**
 * FirstOrFail executa a consulta como First, mas retorna ErrNotFound em
 * vez de (nil, nil) quando nenhum registro é encontrado.
 *
 * Exemplo:
 *  user, err := database.Query[User](db).Where("email", email).FirstOrFail()
 *
 * @return (*T, error)
 */
func (q *QueryBuilder[T]) FirstOrFail() (*T, error) {
	result, err := q.First()
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrNotFound
	}
	return result, nil
}

/**
 * WhereNull adiciona uma condição WHERE column IS NULL à consulta.
 *
 * @param column string
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) WhereNull(column string) *QueryBuilder[T] {
	q.wheres = append(q.wheres, whereClause{column: column, op: "NULL", boolean: "AND"})
	return q
}

/**
 * WhereNotNull adiciona uma condição WHERE column IS NOT NULL à consulta.
 *
 * @param column string
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) WhereNotNull(column string) *QueryBuilder[T] {
	q.wheres = append(q.wheres, whereClause{column: column, op: "NOT NULL", boolean: "AND"})
	return q
}

/**
 * WhereBetween adiciona uma condição WHERE column BETWEEN min AND max.
 *
 * Exemplo:
 *  database.Query[Order](db).WhereBetween("total", 100, 500).Get()
 *
 * @param column string
 * @param min interface{}
 * @param max interface{}
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) WhereBetween(column string, min, max interface{}) *QueryBuilder[T] {
	q.wheres = append(q.wheres, whereClause{column: column, op: "BETWEEN", value: min, value2: max, boolean: "AND"})
	return q
}

/**
 * WhereNotIn adiciona uma restrição WHERE column NOT IN (...) à consulta.
 *
 * @param column string
 * @param values []interface{}
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) WhereNotIn(column string, values []interface{}) *QueryBuilder[T] {
	q.wheres = append(q.wheres, whereClause{column: column, op: "NOT IN", value: values, boolean: "AND"})
	return q
}

/**
 * GroupBy define o agrupamento GROUP BY do resultado por uma ou mais colunas.
 *
 * @param columns ...string
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) GroupBy(columns ...string) *QueryBuilder[T] {
	q.groupBy = append(q.groupBy, columns...)
	return q
}

/**
 * Having adiciona uma condição HAVING, aplicada após o agrupamento (GROUP BY).
 *
 * Exemplo:
 *  database.Query[Order](db).
 *      GroupBy("user_id").
 *      Having("COUNT(*)", ">", 5).
 *      Get()
 *
 * @param column string
 * @param args ...interface{}
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) Having(column string, args ...interface{}) *QueryBuilder[T] {
	op, value := parseWhereArgs(args)
	q.havings = append(q.havings, whereClause{column: column, op: op, value: value, boolean: "AND"})
	return q
}

/**
 * Distinct adiciona o modificador DISTINCT à instrução SELECT, eliminando
 * linhas duplicadas do resultado.
 *
 * @return *QueryBuilder[T]
 */
func (q *QueryBuilder[T]) Distinct() *QueryBuilder[T] {
	q.distinct = true
	return q
}

/**
 * aggregate executa uma função de agregação SQL (SUM, AVG, MIN, MAX) sobre
 * uma coluna, respeitando os filtros WHERE/JOIN já aplicados na consulta.
 */
func (q *QueryBuilder[T]) aggregate(fn, column string) (float64, error) {
	whereSQL, args := q.buildWhereAndJoins()
	query := fmt.Sprintf("SELECT %s(%s)%s", fn, column, whereSQL)

	var result sql.NullFloat64
	if err := q.db.conn.QueryRow(query, args...).Scan(&result); err != nil {
		return 0, fmt.Errorf("orm: erro ao agregar %q: %w", query, err)
	}
	return result.Float64, nil
}

/**
 * Sum retorna a soma dos valores de uma coluna numérica, respeitando os
 * filtros aplicados na consulta. Retorna 0 se não houver registros.
 *
 * @param column string
 * @return (float64, error)
 */
func (q *QueryBuilder[T]) Sum(column string) (float64, error) { return q.aggregate("SUM", column) }

/**
 * Avg retorna a média dos valores de uma coluna numérica.
 *
 * @param column string
 * @return (float64, error)
 */
func (q *QueryBuilder[T]) Avg(column string) (float64, error) { return q.aggregate("AVG", column) }

/**
 * Min retorna o menor valor de uma coluna.
 *
 * @param column string
 * @return (float64, error)
 */
func (q *QueryBuilder[T]) Min(column string) (float64, error) { return q.aggregate("MIN", column) }

/**
 * Max retorna o maior valor de uma coluna.
 *
 * @param column string
 * @return (float64, error)
 */
func (q *QueryBuilder[T]) Max(column string) (float64, error) { return q.aggregate("MAX", column) }

/**
 * Pluck executa a consulta projetando apenas a coluna informada e retorna
 * todos os seus valores como uma slice, sem materializar a struct inteira.
 *
 * Exemplo:
 *  emails, err := database.Query[User](db).Where("active", true).Pluck("email")
 *
 * @param column string
 * @return ([]interface{}, error)
 */
func (q *QueryBuilder[T]) Pluck(column string) ([]interface{}, error) {
	whereSQL, args := q.buildWhereAndJoins()
	query := fmt.Sprintf("SELECT %s%s", column, whereSQL)

	rows, err := q.db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("orm: erro ao executar pluck %q: %w", query, err)
	}
	defer rows.Close()

	var out []interface{}
	for rows.Next() {
		var v interface{}
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

/**
 * Value retorna o valor de uma única coluna do primeiro registro
 * correspondente à consulta, ou nil se não houver resultado.
 *
 * Exemplo:
 *  name, err := database.Query[User](db).Where("id", 1).Value("name")
 *
 * @param column string
 * @return (interface{}, error)
 */
func (q *QueryBuilder[T]) Value(column string) (interface{}, error) {
	q.Limit(1)
	values, err := q.Pluck(column)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values[0], nil
}

/**
 * Chunk executa a consulta em lotes de tamanho size, chamando fn para cada
 * lote, evitando carregar todos os registros em memória de uma só vez.
 * A execução para na primeira chamada de fn que retornar erro.
 *
 * Exemplo:
 *  err := database.Query[User](db).Chunk(100, func(batch []User) error {
 *      for _, u := range batch { ... }
 *      return nil
 *  })
 *
 * @param size int
 * @param fn func([]T) error
 * @return error
 */
func (q *QueryBuilder[T]) Chunk(size int, fn func([]T) error) error {
	page := 0
	for {
		batch := *q
		batch.Limit(size)
		batch.Offset(page * size)

		results, err := batch.Get()
		if err != nil {
			return err
		}
		if len(results) == 0 {
			return nil
		}
		if err := fn(results); err != nil {
			return err
		}
		if len(results) < size {
			return nil
		}
		page++
	}
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
			switch w.op {
			case "NULL":
				fmt.Fprintf(&sb, "%s IS NULL", w.column)
			case "NOT NULL":
				fmt.Fprintf(&sb, "%s IS NOT NULL", w.column)
			case "BETWEEN":
				fmt.Fprintf(&sb, "%s BETWEEN ? AND ?", w.column)
				args = append(args, w.value, w.value2)
			case "IN", "NOT IN":
				values, _ := w.value.([]interface{})
				placeholders := make([]string, len(values))
				for j, v := range values {
					placeholders[j] = "?"
					args = append(args, v)
				}
				verb := "IN"
				if w.op == "NOT IN" {
					verb = "NOT IN"
				}
				fmt.Fprintf(&sb, "%s %s (%s)", w.column, verb, strings.Join(placeholders, ", "))
			default:
				fmt.Fprintf(&sb, "%s %s ?", w.column, w.op)
				args = append(args, w.value)
			}
		}
	}

	if len(q.groupBy) > 0 {
		fmt.Fprintf(&sb, " GROUP BY %s", strings.Join(q.groupBy, ", "))
	}

	if len(q.havings) > 0 {
		sb.WriteString(" HAVING ")
		for i, h := range q.havings {
			if i > 0 {
				fmt.Fprintf(&sb, " %s ", h.boolean)
			}
			fmt.Fprintf(&sb, "%s %s ?", h.column, h.op)
			args = append(args, h.value)
		}
	}

	return sb.String(), args
}

func (q *QueryBuilder[T]) buildSelect() (string, []interface{}) {
	whereSQL, args := q.buildWhereAndJoins()

	selectKeyword := "SELECT"
	if q.distinct {
		selectKeyword = "SELECT DISTINCT"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %s%s", selectKeyword, strings.Join(q.columns(), ", "), whereSQL)

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
