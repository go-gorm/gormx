package gormx

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	Name string `gorm:"column:name"`
}

func (User) TableName() string {
	return "user"
}

func TestBuildSQLWhere(t *testing.T) {
	as := assert.New(t)
	db := newDB()

	type AOrInvalid struct {
		In string `gorm:"column:in; query_expr:in"`
	}
	type BOrOfOrInvalid struct {
		Or struct {
			Or AOrInvalid `gorm:"query_expr:or"`
		} `gorm:"query_expr:or"`
	}

	testBuildSQLWhere := func(opt interface{}, check func(expression clause.Expression, sql string, err error)) {
		expression, err := buildSQLWhere(opt)
		sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Where(Query(opt)).Find(&[]User{}) })
		check(expression, sql, err)
	}

	t.Run("invalid", func(t *testing.T) {
		t.Run("basetype", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Name string `gorm:"column:name; query_expr:invalid"`
			}{}, func(expression clause.Expression, sql string, err error) {
				as.NotNil(err)
				as.Contains(err.Error(), `field(Name) query_expr(invalid) invalid`)
			})
		})

		t.Run("pointer", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Name *string `gorm:"column:name; query_expr:invalid"`
			}{}, func(expression clause.Expression, sql string, err error) {
				as.NotNil(err)
				as.Contains(err.Error(), `field(Name) query_expr(invalid) invalid`)
			})
		})

		t.Run("invalid data", func(t *testing.T) {
			_, err := buildSQLWhere(nil)
			as.NotNil(err)
			as.Equal("gormx's data is invalid", err.Error())
		})

		t.Run("in[or] with invalid datatype", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Or AOrInvalid `gorm:"query_expr:or"`
			}{}, func(expression clause.Expression, sql string, err error) {
				as.NotNil(err)
				as.Contains(err.Error(), `struct field(In) with in query_expr must be slice/array`)
			})
		})

		t.Run("in[or of or] with invalid datatype", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Or BOrOfOrInvalid `gorm:"query_expr:or"`
			}{}, func(expression clause.Expression, sql string, err error) {
				as.NotNil(err)
				as.Contains(err.Error(), `struct field(In) with in query_expr must be slice/array`)
			})
		})
	})

	t.Run("empty-expression", func(t *testing.T) {
		t.Run("basetype", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Name string `gorm:"column:name"`
			}{}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user`", sql)
				as.Nil(expression)
			})
		})

		t.Run("pointer", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Name *string `gorm:"column:name"`
			}{}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user`", sql)
				as.Nil(expression)
			})
		})
	})

	t.Run("one field", func(t *testing.T) {
		t.Run("basetype", func(t *testing.T) {
			name := "bob"
			testBuildSQLWhere(struct {
				Name string `gorm:"column:name"`
			}{
				Name: name,
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `name` = 'bob'", sql)
				assertExprEq[clause.Eq](t, expression, "name", name)
			})
		})

		t.Run("pointer", func(t *testing.T) {
			name := "bob"
			testBuildSQLWhere(struct {
				Name *string `gorm:"column:name"`
			}{
				Name: &name,
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `name` = 'bob'", sql)
				assertExprEq[clause.Eq](t, expression, "name", name)
			})
		})
	})

	t.Run("two field", func(t *testing.T) {
		t.Run("not empty", func(t *testing.T) {
			name := "bob"
			age := 0

			testBuildSQLWhere(struct {
				Name string `gorm:"column:name_jjj"`
				Age  *int   `gorm:"column:age_hhh"`
			}{
				Name: name,
				Age:  &age,
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE (`name_jjj` = 'bob' AND `age_hhh` = 0)", sql)
				exprs := assertExprList[clause.AndConditions](t, expression, 2)
				assertExprEq[clause.Eq](t, exprs[0], "name_jjj", name)
				assertExprEq[clause.Eq](t, exprs[1], "age_hhh", age)
			})
		})

		t.Run("empty", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Name string `gorm:"column:name_jjj"`
				Age  *int   `gorm:"column:age_hhh"`
			}{
				Name: "",
				Age:  nil,
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user`", sql)
				as.Nil(expression)
			})
		})
	})

	t.Run("in", func(t *testing.T) {
		ids := []int64{1, 2, 3}
		names := []string{"a", "b"}

		idsEmpty := []int64{}
		namesEmpty := []string{}

		t.Run("in && not int", func(t *testing.T) {
			testBuildSQLWhere(struct {
				IDs   *[]int64  `gorm:"column:id; query_expr:in"`
				Names *[]string `gorm:"column:name; query_expr:not in"`
			}{
				IDs:   &ids,
				Names: &names,
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE (`id` IN (1,2,3) AND `name` NOT IN ('a','b'))", sql)
				exprs := assertExprList[clause.AndConditions](t, expression, 2)
				assertExprIn(t, exprs[0], "id", toAnySlice(ids))
				assertExprNotIn(t, exprs[1], "name", toAnySlice(names))
			})
		})

		t.Run("empty slice", func(t *testing.T) {
			t.Run("basetype", func(t *testing.T) {
				testBuildSQLWhere(struct {
					IDs   []int64  `gorm:"column:id; query_expr:in"`
					Names []string `gorm:"column:name; query_expr:not in"`
				}{
					IDs:   []int64{},
					Names: []string{},
				}, func(expression clause.Expression, sql string, err error) {
					as.Nil(err)
					as.Equal("SELECT * FROM `user`", sql)
					assertExprList[clause.AndConditions](t, expression, 0)
				})
			})

			t.Run("pointer", func(t *testing.T) {
				testBuildSQLWhere(struct {
					IDs   *[]int64  `gorm:"column:id; query_expr:in"`
					Names *[]string `gorm:"column:name; query_expr:not in"`
				}{
					IDs:   &idsEmpty,
					Names: &namesEmpty,
				}, func(expression clause.Expression, sql string, err error) {
					as.Nil(err)
					as.Equal("SELECT * FROM `user` WHERE (`id` IN (NULL) AND `name` IS NOT NULL)", sql)
					exprs := assertExprList[clause.AndConditions](t, expression, 2)
					assertExprIn(t, exprs[0], "id", []any{})
					assertExprNotIn(t, exprs[1], "name", []any{})
				})
			})
		})

		t.Run("slice", func(t *testing.T) {
			testBuildSQLWhere(struct {
				IDs   []int64  `gorm:"column:id; query_expr:in"`
				Names []string `gorm:"column:name; query_expr:not in"`
			}{
				IDs:   ids,
				Names: names,
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE (`id` IN (1,2,3) AND `name` NOT IN ('a','b'))", sql)
				exprs := assertExprList[clause.AndConditions](t, expression, 2)
				assertExprIn(t, exprs[0], "id", toAnySlice(ids))
				assertExprNotIn(t, exprs[1], "name", toAnySlice(names))
			})
		})

		t.Run("invalid", func(t *testing.T) {
			testBuildSQLWhere(struct {
				IDs []int64 `gorm:"column:id; query_expr:="`
			}{
				IDs: ids,
			}, func(expression clause.Expression, sql string, err error) {
				as.NotNil(err)
				as.Equal("struct field(IDs) with eq query_expr can not be slice/array", err.Error())
			})
		})
	})

	t.Run("like", func(t *testing.T) {
		t.Run("like", func(t *testing.T) {
			name := "%name%"
			testBuildSQLWhere(struct {
				NameLike *string `gorm:"column:name; query_expr:like"`
			}{
				NameLike: &name,
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `name` LIKE '%name%'", sql)
				assertExprEq[clause.Like](t, expression, "name", name)
			})
		})
	})

	t.Run("compare", func(t *testing.T) {
		t.Run(">", func(t *testing.T) {
			testBuildSQLWhere(struct {
				ID   *int  `gorm:"column:id; query_expr:>"`
				Name *bool `gorm:"column:name; query_expr:null"`
			}{
				ID: ptr[int](1),
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `id` > 1", sql)
				assertExprEq[clause.Gt](t, expression, "id", 1)
			})
		})

		t.Run(">=", func(t *testing.T) {
			testBuildSQLWhere(struct {
				ID   *int  `gorm:"column:id; query_expr:>="`
				Name *bool `gorm:"column:name; query_expr:null"`
			}{
				ID: ptr[int](1),
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `id` >= 1", sql)
				assertExprEq[clause.Gte](t, expression, "id", 1)
			})
		})

		t.Run("=", func(t *testing.T) {
			testBuildSQLWhere(struct {
				ID   *int  `gorm:"column:id; query_expr:="`
				Name *bool `gorm:"column:name; query_expr:null"`
			}{
				ID: ptr[int](1),
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `id` = 1", sql)
				assertExprEq[clause.Eq](t, expression, "id", 1)
			})
		})

		t.Run("<", func(t *testing.T) {
			testBuildSQLWhere(struct {
				ID   *int  `gorm:"column:id; query_expr:<"`
				Name *bool `gorm:"column:name; query_expr:null"`
			}{
				ID: ptr[int](1),
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `id` < 1", sql)
				assertExprEq[clause.Lt](t, expression, "id", 1)
			})
		})

		t.Run("<=", func(t *testing.T) {
			testBuildSQLWhere(struct {
				ID   *int  `gorm:"column:id; query_expr:<="`
				Name *bool `gorm:"column:name; query_expr:null"`
			}{
				ID: ptr[int](1),
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `id` <= 1", sql)
				assertExprEq[clause.Lte](t, expression, "id", 1)
			})
		})

		t.Run("!=", func(t *testing.T) {
			testBuildSQLWhere(struct {
				ID   *int  `gorm:"column:id; query_expr:!="`
				Name *bool `gorm:"column:name; query_expr:null"`
			}{
				ID: ptr[int](1),
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `id` <> 1", sql)
				assertExprEq[clause.Neq](t, expression, "id", 1)
			})
		})
	})

	t.Run("null", func(t *testing.T) {
		t.Run("is null - empty", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Name *bool `gorm:"column:name; query_expr:null"`
			}{
				Name: nil,
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user`", sql)
				as.Nil(expression)
			})
		})

		t.Run("is null - is null", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Name *bool `gorm:"column:name; query_expr:null"`
			}{
				Name: ptr(true),
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `name` IS NULL", sql)
				assertExprEq[clause.Eq](t, expression, "name", nil)
			})
		})

		t.Run("is null - is not null", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Name *bool `gorm:"column:name; query_expr:null"`
			}{
				Name: ptr(false),
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `name` IS NOT NULL", sql)
				assertExprEq[clause.Neq](t, expression, "name", nil)
			})
		})

		t.Run("is null - not bool", func(t *testing.T) {
			testBuildSQLWhere(struct {
				Name string `gorm:"column:name; query_expr:null"`
			}{
				Name: "x",
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user`", sql)
				as.Nil(expression)
			})
		})
	})

	t.Run("anonymous struct", func(t *testing.T) {
		type WhereUser struct {
			UserID *int64 `gorm:"column:user_id"`
		}
		testBuildSQLWhere(struct {
			WhereUser
			ParentID *int64 `gorm:"column:parent_id"`
		}{
			ParentID: ptr[int64](1),
			WhereUser: WhereUser{
				UserID: ptr[int64](2),
			},
		}, func(expression clause.Expression, sql string, err error) {
			as.Nil(err)
			as.Equal("SELECT * FROM `user` WHERE (`user_id` = 2 AND `parent_id` = 1)", sql)

			exprs := assertExprList[clause.AndConditions](t, expression, 2)
			assertExprEq[clause.Eq](t, exprs[0], "user_id", int64(2))
			assertExprEq[clause.Eq](t, exprs[1], "parent_id", int64(1))
		})
	})

	t.Run("or", func(t *testing.T) {
		type WhereUser struct {
			UserID     *int64      `gorm:"column:user_id"`
			UserName   *string     `gorm:"column:user_name"`
			UserAge    *int64      `gorm:"column:user_age"`
			OrClauses1 []WhereUser `gorm:"query_expr:or"`
			OrClauses2 *WhereUser  `gorm:"query_expr:or"`
		}

		t.Run("or with slice", func(t *testing.T) {
			testBuildSQLWhere(WhereUser{
				OrClauses1: []WhereUser{
					{
						UserName: ptr("dirac"),
					},
					{
						UserAge: ptr[int64](18),
					},
				},
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE (`user_name` = 'dirac' OR `user_age` = 18)", sql)

				exprs := assertExprList[clause.OrConditions](t, expression, 2)
				assertExprEq[clause.Eq](t, exprs[0], "user_name", "dirac")
				assertExprEq[clause.Eq](t, exprs[1], "user_age", int64(18))
			})
		})

		t.Run("and + or", func(t *testing.T) {
			testBuildSQLWhere(WhereUser{
				UserID: ptr[int64](1),
				OrClauses1: []WhereUser{
					{
						UserName: ptr("dirac"),
					},
					{
						UserAge: ptr[int64](18),
					},
				},
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE (`user_id` = 1 AND (`user_name` = 'dirac' OR `user_age` = 18))", sql)

				exprs := assertExprList[clause.AndConditions](t, expression, 2)
				assertExprEq[clause.Eq](t, exprs[0], "user_id", int64(1))
				exprs2 := assertExprList[clause.OrConditions](t, exprs[1], 2)
				assertExprEq[clause.Eq](t, exprs2[0], "user_name", "dirac")
				assertExprEq[clause.Eq](t, exprs2[1], "user_age", int64(18))
			})
		})

		t.Run("empty or", func(t *testing.T) {
			testBuildSQLWhere(WhereUser{
				UserID: ptr[int64](1),
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `user_id` = 1", sql)
				assertExprEq[clause.Eq](t, expression, "user_id", int64(1))
			})
		})

		t.Run("empty or", func(t *testing.T) {
			testBuildSQLWhere(WhereUser{
				UserID: ptr[int64](1),
				OrClauses1: []WhereUser{
					{},
					{},
				},
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE `user_id` = 1", sql)
				assertExprEq[clause.Eq](t, expression, "user_id", int64(1))
			})
		})

		t.Run("or of or", func(t *testing.T) {
			testBuildSQLWhere(WhereUser{
				UserID: ptr[int64](1),
				OrClauses1: []WhereUser{
					{
						UserAge: ptr[int64](18),
						OrClauses1: []WhereUser{
							{
								UserName: ptr("bob"),
							},
							{
								UserName: ptr("dirac"),
							},
						},
					},
					{
						UserAge: ptr[int64](19),
					},
				},
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE (`user_id` = 1 AND ((`user_age` = 18 OR (`user_name` = 'bob' OR `user_name` = 'dirac')) OR `user_age` = 19))", sql)

				exprs := assertExprList[clause.AndConditions](t, expression, 2)
				assertExprEq[clause.Eq](t, exprs[0], "user_id", int64(1))
				exprs2 := assertExprList[clause.OrConditions](t, exprs[1], 2)

				exprs3 := assertExprList[clause.OrConditions](t, exprs2[0], 2)
				assertExprEq[clause.Eq](t, exprs2[1], "user_age", int64(19))

				assertExprEq[clause.Eq](t, exprs3[0], "user_age", int64(18))
				exprs4 := assertExprList[clause.OrConditions](t, exprs3[1], 2)

				assertExprEq[clause.Eq](t, exprs4[0], "user_name", "bob")
				assertExprEq[clause.Eq](t, exprs4[1], "user_name", "dirac")
			})
		})

		t.Run("multi or", func(t *testing.T) {
			testBuildSQLWhere(WhereUser{
				UserID: ptr[int64](1),
				OrClauses1: []WhereUser{
					{
						UserAge: ptr[int64](18),
					},
					{
						UserAge: ptr[int64](19),
					},
				},
				OrClauses2: &WhereUser{
					UserName: ptr("dirac"),
				},
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE (`user_id` = 1 AND (`user_age` = 18 OR `user_age` = 19) AND `user_name` = 'dirac')", sql)

				exprs := assertExprList[clause.AndConditions](t, expression, 3)
				assertExprEq[clause.Eq](t, exprs[0], "user_id", int64(1))
				exprs2 := assertExprList[clause.OrConditions](t, exprs[1], 2)
				assertExprEq[clause.Eq](t, exprs[2], "user_name", "dirac")

				assertExprEq[clause.Eq](t, exprs2[0], "user_age", int64(18))
				assertExprEq[clause.Eq](t, exprs2[1], "user_age", int64(19))
			})
		})

		t.Run("column=-, query_expr=or", func(t *testing.T) {
			type WhereUser struct {
				UserID    *int64      `gorm:"column:user_id"`
				UserName  *string     `gorm:"column:user_name"`
				UserAge   *int64      `gorm:"column:user_age"`
				OrClauses []WhereUser `gorm:"column:-; query_expr:or"`
			}

			testBuildSQLWhere(WhereUser{
				UserAge: ptr[int64](18),
				OrClauses: []WhereUser{
					{
						UserID:   ptr[int64](123),
						UserName: ptr("bob"),
					},
					{
						UserID:   ptr[int64](234),
						UserName: ptr("dirac"),
					},
				},
			}, func(expression clause.Expression, sql string, err error) {
				as.Nil(err)
				as.Equal("SELECT * FROM `user` WHERE (`user_age` = 18 AND ((`user_id` = 123 OR `user_name` = 'bob') OR (`user_id` = 234 OR `user_name` = 'dirac')))", sql)

				exprs := assertExprList[clause.AndConditions](t, expression, 2)
				assertExprEq[clause.Eq](t, exprs[0], "user_age", int64(18))
				exprs2 := assertExprList[clause.OrConditions](t, exprs[1], 2)

				expr3 := assertExprList[clause.OrConditions](t, exprs2[0], 2)
				expr4 := assertExprList[clause.OrConditions](t, exprs2[1], 2)

				assertExprEq[clause.Eq](t, expr3[0], "user_id", int64(123))
				assertExprEq[clause.Eq](t, expr3[1], "user_name", "bob")

				assertExprEq[clause.Eq](t, expr4[0], "user_id", int64(234))
				assertExprEq[clause.Eq](t, expr4[1], "user_name", "dirac")
			})
		})
	})

	t.Run("json_contains", func(t *testing.T) {
		testBuildSQLWhere(struct {
			JSONField *string `gorm:"column:json_field; json_contains:json_key"`
		}{}, func(expression clause.Expression, sql string, err error) {
		})
	})
}

func assertExprIn(t *testing.T, expression clause.Expression, column string, value any) {
	as := assert.New(t)

	eq, ok := expression.(clause.IN)
	as.True(ok)

	_, ok = eq.Column.(clause.Column)
	as.True(ok)

	as.Equal(column, eq.Column.(clause.Column).Name)
	as.Equal(value, eq.Values)
}

func assertExprEq[T any](t *testing.T, expression clause.Expression, column string, value any) {
	as := assert.New(t)

	_, ok := expression.(T)
	if !ok {
		t.Errorf("expression(%T) is not %T", expression, new(T))
		return
	}

	eqType := reflect.TypeOf(clause.Eq{})
	ev := reflect.ValueOf(expression)
	if !ev.CanConvert(eqType) {
		t.Errorf("expression can not convert to clause.Eq")
		return
	}
	eq := ev.Convert(eqType).Interface().(clause.Eq)

	_, ok = eq.Column.(clause.Column)
	as.True(ok)

	as.Equal(column, eq.Column.(clause.Column).Name)
	as.Equal(value, eq.Value)
}

func assertExprNotIn(t *testing.T, expression clause.Expression, column string, value any) {
	as := assert.New(t)

	eq, ok := expression.(notIn)
	as.True(ok)

	as.Equal(column, eq.in.Column.(clause.Column).Name)
	as.Equal(value, eq.in.Values)
}

func assertExprList[T any](t *testing.T, expression clause.Expression, length int) []clause.Expression {
	as := assert.New(t)

	if length == 0 {
		as.Nil(expression)
		return nil
	}

	if _, ok := expression.(T); !ok {
		t.Errorf("expression(%T) is not %T", expression, new(T))
		return nil
	}

	rv := reflect.ValueOf(expression)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	exprs := rv.FieldByName("Exprs").Interface().([]clause.Expression)

	// eq, ok := expression.(clause.AndConditions)
	// as.True(ok)

	as.Equal(length, len(exprs))

	return exprs
}

func toAnySlice[T any](data []T) []any {
	var result []any
	for _, v := range data {
		result = append(result, v)
	}
	return result
}

func Test_getQueryExpr(t *testing.T) {
	as := assert.New(t)

	t.Run("not found", func(t *testing.T) {
		_, err := getQueryExpr("not found")
		as.NotNil(err)
		as.Equal("query_expr 'not found' invalid", err.Error())
	})
}

func Test_buildClauseExpression(t *testing.T) {
	as := assert.New(t)

	t.Run("invalid query_expr", func(t *testing.T) {
		_, err := buildClauseExpression(reflect.ValueOf(struct {
			Name string `query_expr:"invalid"`
		}{Name: "str"}), &structType{
			Names: []string{"Name"},
			Fields: map[string]*fieldType{
				"Name": {
					Name:      "Name",
					QueryExpr: "invalid",
				},
			},
		}, true)
		as.NotNil(err)
		as.Equal("query_expr 'invalid' invalid", err.Error())
	})
}
