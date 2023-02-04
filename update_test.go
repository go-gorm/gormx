package gormx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestBuildSQLUpdate(t *testing.T) {
	as := assert.New(t)
	db := newDB()
	as.Nil(db.Migrator().AutoMigrate(&User{}))

	testBuildSQLUpdate := func(opt interface{}, check func(result map[string]interface{}, sql string, err error)) {
		result, err := buildSQLUpdate(opt)
		sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Table("user").Where(Query(struct {
				ID int `gorm:"column:id"`
			}{ID: 1})).Updates(Update(opt))
		})
		check(result, sql, err)
	}

	t.Run("invalid", func(t *testing.T) {
		testBuildSQLUpdate(struct {
			Age *int `gorm:"column:age; update_expr:invalid"`
		}{
			Age: ptr[int](1),
		}, func(m map[string]interface{}, sql string, err error) {
			as.NotNil(err)
			as.Equal("", sql)

			as.Equal("field(Age) update_expr(invalid) invalid", err.Error())
		})
	})

	t.Run("empty", func(t *testing.T) {
		testBuildSQLUpdate(struct {
			Name *string `gorm:"column:name"`
		}{}, func(m map[string]interface{}, sql string, err error) {
			as.Nil(err)
			as.Equal("", sql)

			as.Len(m, 0)
		})
	})

	t.Run("direct set", func(t *testing.T) {
		t.Run("one field", func(t *testing.T) {
			name := "bob"

			testBuildSQLUpdate(struct {
				Name *string `gorm:"column:name"`
				Age  *int    `gorm:"column:age"`
			}{
				Name: &name,
			}, func(m map[string]interface{}, sql string, err error) {
				as.Nil(err)
				as.Equal("UPDATE `user` SET `name`='bob' WHERE `id` = 1", sql)

				as.Len(m, 1)
				as.Equal(name, m["name"])
			})
		})

		t.Run("field 名称不同", func(t *testing.T) {
			name := "bob"
			age := 20
			testBuildSQLUpdate(struct {
				Name *string `gorm:"column:name_jjj"`
				Age  *int    `gorm:"column:age_hhh"`
			}{
				Name: &name,
				Age:  &age,
			}, func(m map[string]interface{}, sql string, err error) {
				as.Nil(err)
				as.Equal("UPDATE `user` SET `age_hhh`=20,`name_jjj`='bob' WHERE `id` = 1", sql)

				as.Len(m, 2)
				as.Equal(name, m["name_jjj"])
				as.Equal(age, m["age_hhh"])
			})
		})
	})

	t.Run("+", func(t *testing.T) {
		testBuildSQLUpdate(struct {
			Age *int `gorm:"column:age; update_expr:+"`
		}{
			Age: ptr[int](1),
		}, func(m map[string]interface{}, sql string, err error) {
			as.Nil(err)
			as.Equal("UPDATE `user` SET `age`=age + 1 WHERE `id` = 1", sql)

			as.Len(m, 1)
			as.Equal(clause.Expr{SQL: "age + ?", Vars: []interface{}{1}}, m["age"])
		})
	})

	t.Run("-", func(t *testing.T) {
		testBuildSQLUpdate(struct {
			Age *int `gorm:"column:age; update_expr:-"`
		}{
			Age: ptr[int](1),
		}, func(m map[string]interface{}, sql string, err error) {
			as.Nil(err)
			as.Equal("UPDATE `user` SET `age`=age - 1 WHERE `id` = 1", sql)

			as.Len(m, 1)
			as.Equal(clause.Expr{SQL: "age - ?", Vars: []interface{}{1}}, m["age"])
		})
	})

	t.Run("merge_json", func(t *testing.T) {
		t.Run("nil", func(t *testing.T) {
			testBuildSQLUpdate(struct {
				Data *map[string]interface{} `gorm:"column:data; update_expr:merge_json"`
			}{
				Data: nil,
			}, func(m map[string]interface{}, sql string, err error) {
				as.Nil(err)
				as.Equal("", sql)

				as.Len(m, 0)
			})
		})

		t.Run("empty", func(t *testing.T) {
			testBuildSQLUpdate(struct {
				Data string `gorm:"column:data; update_expr:merge_json"`
			}{
				Data: "",
			}, func(m map[string]interface{}, sql string, err error) {
				as.Nil(err)
				as.Equal("", sql)

				as.Len(m, 0)
			})
		})

		t.Run("map", func(t *testing.T) {
			m := map[string]interface{}{"a": "a", "b": 2, "c": false}
			testBuildSQLUpdate(struct {
				Data *map[string]interface{} `gorm:"column:data; update_expr:merge_json"`
			}{
				Data: &m,
			}, func(m map[string]interface{}, sql string, err error) {
				as.Nil(err)
				as.Equal("UPDATE `user` SET `data`=CASE WHEN (`data` IS NULL OR `data` = '') THEN CAST('{\"a\":\"a\",\"b\":2,\"c\":false}' AS JSON) ELSE JSON_MERGE_PATCH(`data`, CAST('{\"a\":\"a\",\"b\":2,\"c\":false}' AS JSON)) END WHERE `id` = 1", sql)

				as.Len(m, 1)
				as.Equal(clause.Expr{
					SQL:  "CASE WHEN (`data` IS NULL OR `data` = '') THEN CAST(? AS JSON) ELSE JSON_MERGE_PATCH(`data`, CAST(? AS JSON)) END",
					Vars: []interface{}{"{\"a\":\"a\",\"b\":2,\"c\":false}", "{\"a\":\"a\",\"b\":2,\"c\":false}"},
				}, m["data"])
			})
		})

		t.Run("struct-no-nil", func(t *testing.T) {
			type data struct {
				A string `json:"a"`
				B int    `json:"b"`
				C bool   `json:"c"`
				D string `json:"-"`
			}
			testBuildSQLUpdate(struct {
				Data *data `gorm:"column:data; update_expr:merge_json"`
			}{
				Data: &data{
					A: "a",
					B: 2,
					C: false,
				},
			}, func(m map[string]interface{}, sql string, err error) {
				as.Nil(err)
				as.Equal("UPDATE `user` SET `data`=CASE WHEN (`data` IS NULL OR `data` = '') THEN CAST('{\"a\":\"a\",\"b\":2,\"c\":false}' AS JSON) ELSE JSON_MERGE_PATCH(`data`, CAST('{\"a\":\"a\",\"b\":2,\"c\":false}' AS JSON)) END WHERE `id` = 1", sql)

				as.Len(m, 1)
				as.Equal(clause.Expr{
					SQL:  "CASE WHEN (`data` IS NULL OR `data` = '') THEN CAST(? AS JSON) ELSE JSON_MERGE_PATCH(`data`, CAST(? AS JSON)) END",
					Vars: []interface{}{"{\"a\":\"a\",\"b\":2,\"c\":false}", "{\"a\":\"a\",\"b\":2,\"c\":false}"},
				}, m["data"])
			})
		})

		t.Run("struct-no-nil", func(t *testing.T) {
			type data struct {
				A interface{} `json:"a"`
			}
			testBuildSQLUpdate(struct {
				Data *data `gorm:"column:data; update_expr:merge_json"`
			}{
				Data: &data{
					A: nil,
				},
			}, func(m map[string]interface{}, sql string, err error) {
				as.Nil(err)
				as.Equal("UPDATE `user` SET `data`=CASE WHEN (`data` IS NULL OR `data` = '') THEN CAST('{\"a\":null}' AS JSON) ELSE JSON_MERGE_PATCH(`data`, CAST('{\"a\":null}' AS JSON)) END WHERE `id` = 1", sql)

				as.Len(m, 1)
				as.Equal(clause.Expr{
					SQL:  "CASE WHEN (`data` IS NULL OR `data` = '') THEN CAST(? AS JSON) ELSE JSON_MERGE_PATCH(`data`, CAST(? AS JSON)) END",
					Vars: []interface{}{"{\"a\":null}", "{\"a\":null}"},
				}, m["data"])
			})
		})

		t.Run("struct-no-nil - pointer of pointer", func(t *testing.T) {
			type data struct {
				A string `json:"a"`
				B int    `json:"b"`
				C bool   `json:"c"`
			}
			dataX := &data{
				A: "a",
				B: 2,
				C: false,
			}
			testBuildSQLUpdate(struct {
				Data **data `gorm:"column:data; update_expr:merge_json"`
			}{
				Data: &dataX,
			}, func(m map[string]interface{}, sql string, err error) {
				as.Nil(err)
				as.Equal("UPDATE `user` SET `data`=CASE WHEN (`data` IS NULL OR `data` = '') THEN CAST('{\"a\":\"a\",\"b\":2,\"c\":false}' AS JSON) ELSE JSON_MERGE_PATCH(`data`, CAST('{\"a\":\"a\",\"b\":2,\"c\":false}' AS JSON)) END WHERE `id` = 1", sql)

				as.Len(m, 1)
				as.Equal(clause.Expr{
					SQL:  "CASE WHEN (`data` IS NULL OR `data` = '') THEN CAST(? AS JSON) ELSE JSON_MERGE_PATCH(`data`, CAST(? AS JSON)) END",
					Vars: []interface{}{"{\"a\":\"a\",\"b\":2,\"c\":false}", "{\"a\":\"a\",\"b\":2,\"c\":false}"},
				}, m["data"])
			})
		})

		t.Run("struct-nil", func(t *testing.T) {
			type data struct {
				A string `json:"a"`
				B int    `json:"b"`
				C bool   `json:"c"`
			}
			testBuildSQLUpdate(struct {
				Data *data `gorm:"column:data; update_expr:merge_json"`
			}{
				Data: nil,
			}, func(m map[string]interface{}, sql string, err error) {
				as.Nil(err)
				as.Equal("", sql)

				as.Len(m, 0)
			})
		})
	})
}

func Test_StructHelper(t *testing.T) {
	as := assert.New(t)

	{
		_, err := mergeJSONStructToJSONMap(0)
		as.NotNil(err)
		as.Equal("update(JSON_MERGE_PATCH) need struct type", err.Error())
	}

	{
		_, err := mergeJSONStructToJSONMap(ptr[int64](1))
		as.NotNil(err)
		as.Equal("update(JSON_MERGE_PATCH) need struct type", err.Error())
	}

	{

		m, err := mergeJSONStructToJSONMap(struct {
			Name *string `json:"name"`
		}{})
		as.Nil(err)
		as.Equal(map[string]interface{}(map[string]interface{}{}), m)
	}

	{

		m, err := mergeJSONStructToJSONMap(struct {
			Name *string `json:"name"`
		}{
			Name: ptr("name1"),
		})
		as.Nil(err)
		as.Equal(map[string]interface{}(map[string]interface{}{"name": "name1"}), m)
	}

	{

		m, err := mergeJSONStructToJSONMap(struct {
			Name *string `json:"name"`
			Age  *int    `json:"age"`
		}{
			Name: ptr("name1"),
			Age:  nil,
		})
		as.Nil(err)
		as.Equal(map[string]interface{}(map[string]interface{}{"name": "name1"}), m)
	}

	{

		m, err := mergeJSONStructToJSONMap(struct {
			Name *string `json:"name"`
			Age  int32   `json:"age"`
		}{
			Name: ptr("name1"),
			Age:  0,
		})
		as.Nil(err)
		as.Equal(map[string]interface{}(map[string]interface{}{"name": "name1", "age": int32(0)}), m)
	}
}

func Test_buildSQLUpdate(t *testing.T) {
	as := assert.New(t)

	t.Run("", func(t *testing.T) {
		_, err := buildSQLUpdate(nil)
		as.NotNil(err)
		as.Equal("gormx's data is invalid", err.Error())
	})
}
