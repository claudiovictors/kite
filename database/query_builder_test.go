package database

import (
	"reflect"
	"testing"
)

/**
 * testUser é o modelo utilizado exclusivamente nos testes de geração de
 * SQL do QueryBuilder. Como estes testes verificam apenas a string SQL
 * e os argumentos montados — nunca executam contra um banco de dados
 * real — não é necessário nenhum driver (nem sqlite, nem nada externo).
 */
type testUser struct {
	Model
	Name   string `db:"name"`
	Email  string `db:"email"`
	Age    int    `db:"age"`
	Active bool   `db:"active"`
	Bio    string `db:"bio"`
}

/**
 * newTestBuilder cria um QueryBuilder[testUser] apoiado num *DB "vazio"
 * (sem conexão real). Isto funciona porque registerModel só usa
 * reflection, e buildSelect/buildWhereAndJoins/buildCount nunca tocam
 * db.conn — são funções puras de montagem de string SQL. O mapa
 * metadata precisa estar inicializado, senão registerModel entra em
 * panic ao tentar escrever numa map nil.
 *
 * @param t *testing.T
 * @return *QueryBuilder[testUser]
 */
func newTestBuilder(t *testing.T) *QueryBuilder[testUser] {
	t.Helper()
	db := &DB{metadata: make(map[reflect.Type]*modelMeta)}
	return Query[testUser](db)
}

func TestQueryBuilder_BuildSelect_Basic(t *testing.T) {
	q := newTestBuilder(t)
	sql, args := q.buildSelect()

	wantSQL := "SELECT id, name, email, age, active, bio FROM test_users"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado:\n got:  %q\n want: %q", sql, wantSQL)
	}
	if len(args) != 0 {
		t.Fatalf("esperava 0 args, veio %v", args)
	}
}

func TestQueryBuilder_Where_ImplicitEquals(t *testing.T) {
	q := newTestBuilder(t).Where("name", "Ana")
	sql, args := q.buildSelect()

	wantSQL := "SELECT id, name, email, age, active, bio FROM test_users WHERE name = ?"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado:\n got:  %q\n want: %q", sql, wantSQL)
	}
	if !reflect.DeepEqual(args, []interface{}{"Ana"}) {
		t.Fatalf("args inesperados: %v", args)
	}
}

func TestQueryBuilder_Where_ExplicitOperator(t *testing.T) {
	q := newTestBuilder(t).Where("age", ">=", 18)
	sql, args := q.buildSelect()

	wantSQL := "SELECT id, name, email, age, active, bio FROM test_users WHERE age >= ?"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado: %q", sql)
	}
	if !reflect.DeepEqual(args, []interface{}{18}) {
		t.Fatalf("args inesperados: %v", args)
	}
}

func TestQueryBuilder_OrWhere(t *testing.T) {
	q := newTestBuilder(t).Where("role", "admin").OrWhere("role", "editor")
	sql, args := q.buildSelect()

	wantSQL := "SELECT id, name, email, age, active, bio FROM test_users WHERE role = ? OR role = ?"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado: %q", sql)
	}
	if !reflect.DeepEqual(args, []interface{}{"admin", "editor"}) {
		t.Fatalf("args inesperados: %v", args)
	}
}

func TestQueryBuilder_WhereIn(t *testing.T) {
	q := newTestBuilder(t).WhereIn("id", []interface{}{1, 2, 3})
	sql, args := q.buildSelect()

	wantSQL := "SELECT id, name, email, age, active, bio FROM test_users WHERE id IN (?, ?, ?)"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado: %q", sql)
	}
	if !reflect.DeepEqual(args, []interface{}{1, 2, 3}) {
		t.Fatalf("args inesperados: %v", args)
	}
}

func TestQueryBuilder_WhereNotIn(t *testing.T) {
	q := newTestBuilder(t).WhereNotIn("role", []interface{}{"banned", "suspended"})
	sql, _ := q.buildSelect()

	wantSQL := "SELECT id, name, email, age, active, bio FROM test_users WHERE role NOT IN (?, ?)"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado: %q", sql)
	}
}

func TestQueryBuilder_WhereNullAndNotNull(t *testing.T) {
	nullSQL, nullArgs := newTestBuilder(t).WhereNull("bio").buildSelect()
	wantNull := "SELECT id, name, email, age, active, bio FROM test_users WHERE bio IS NULL"
	if nullSQL != wantNull {
		t.Fatalf("SQL inesperado (WhereNull): %q", nullSQL)
	}
	if len(nullArgs) != 0 {
		t.Fatalf("WhereNull não deveria gerar args, veio %v", nullArgs)
	}

	notNullSQL, _ := newTestBuilder(t).WhereNotNull("bio").buildSelect()
	wantNotNull := "SELECT id, name, email, age, active, bio FROM test_users WHERE bio IS NOT NULL"
	if notNullSQL != wantNotNull {
		t.Fatalf("SQL inesperado (WhereNotNull): %q", notNullSQL)
	}
}

func TestQueryBuilder_WhereBetween(t *testing.T) {
	q := newTestBuilder(t).WhereBetween("age", 18, 30)
	sql, args := q.buildSelect()

	wantSQL := "SELECT id, name, email, age, active, bio FROM test_users WHERE age BETWEEN ? AND ?"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado: %q", sql)
	}
	if !reflect.DeepEqual(args, []interface{}{18, 30}) {
		t.Fatalf("args inesperados: %v", args)
	}
}

func TestQueryBuilder_GroupByAndHaving(t *testing.T) {
	q := newTestBuilder(t).GroupBy("active").Having("COUNT(*)", ">", 1)
	sql, args := q.buildSelect()

	wantSQL := "SELECT id, name, email, age, active, bio FROM test_users GROUP BY active HAVING COUNT(*) > ?"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado: %q", sql)
	}
	if !reflect.DeepEqual(args, []interface{}{1}) {
		t.Fatalf("args inesperados: %v", args)
	}
}

func TestQueryBuilder_Distinct(t *testing.T) {
	q := newTestBuilder(t).Distinct().Select("active")
	sql, _ := q.buildSelect()

	wantSQL := "SELECT DISTINCT active FROM test_users"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado: %q", sql)
	}
}

func TestQueryBuilder_OrderByAndLimitOffset(t *testing.T) {
	q := newTestBuilder(t).OrderByDesc("age").Limit(10).Offset(5)
	sql, _ := q.buildSelect()

	wantSQL := "SELECT id, name, email, age, active, bio FROM test_users ORDER BY age DESC LIMIT 10 OFFSET 5"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado: %q", sql)
	}
}

func TestQueryBuilder_JoinAndLeftJoin(t *testing.T) {
	inner := newTestBuilder(t).Join("posts", "posts.user_id", "=", "test_users.id")
	innerSQL, _ := inner.buildSelect()
	wantInner := "SELECT id, name, email, age, active, bio FROM test_users INNER JOIN posts ON posts.user_id = test_users.id"
	if innerSQL != wantInner {
		t.Fatalf("SQL inesperado (Join): %q", innerSQL)
	}

	left := newTestBuilder(t).LeftJoin("posts", "posts.user_id", "=", "test_users.id")
	leftSQL, _ := left.buildSelect()
	wantLeft := "SELECT id, name, email, age, active, bio FROM test_users LEFT JOIN posts ON posts.user_id = test_users.id"
	if leftSQL != wantLeft {
		t.Fatalf("SQL inesperado (LeftJoin): %q", leftSQL)
	}
}

func TestQueryBuilder_BuildCount(t *testing.T) {
	q := newTestBuilder(t).Where("active", true)
	sql, args := q.buildCount()

	wantSQL := "SELECT COUNT(*) FROM test_users WHERE active = ?"
	if sql != wantSQL {
		t.Fatalf("SQL inesperado: %q", sql)
	}
	if !reflect.DeepEqual(args, []interface{}{true}) {
		t.Fatalf("args inesperados: %v", args)
	}
}

func TestQueryBuilder_ParseWhereArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		wantOp  string
		wantVal interface{}
	}{
		{"apenas valor", []interface{}{"Ana"}, "=", "Ana"},
		{"operador explícito", []interface{}{">=", 18}, ">=", 18},
		{"vazio", nil, "=", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, val := parseWhereArgs(tt.args)
			if op != tt.wantOp || val != tt.wantVal {
				t.Fatalf("got (%q, %v), want (%q, %v)", op, val, tt.wantOp, tt.wantVal)
			}
		})
	}
}