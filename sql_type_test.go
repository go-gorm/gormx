package gormx

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseType(t *testing.T) {
	as := assert.New(t)

	type AStruct struct {
		A *int `gorm:"column:a; query_expr:>"`
	}
	type BStruct struct {
		B *string `gorm:"column:b; query_expr:!="`
	}
	type CStruct struct {
		B *int `gorm:"column:b; query_expr:!="`
		BStruct
	}

	tests := []struct {
		name    string
		args    reflect.Type
		want    *structType
		wantErr assert.ErrorAssertionFunc
	}{
		{"err - invalid update_expr", reflect.TypeOf(struct {
			A *int `gorm:"column:a; update_expr:x"`
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("field(A) update_expr(x) invalid", err.Error())
			return false
		}},
		{"err - invalid query_expr", reflect.TypeOf(struct {
			A *int `gorm:"column:a; query_expr:x"`
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("field(A) query_expr(x) invalid", err.Error())
			return false
		}},
		{"err - invalid anonymous type", reflect.TypeOf(struct {
			io.Reader
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("field's type must be struct/slice, but got interface", err.Error())
			return false
		}},
		{"err - invalid anonymous column name not empty", reflect.TypeOf(struct {
			BStruct `gorm:"column:b"`
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("field BStruct is anonymous that can not have column tag", err.Error())
			return false
		}},
		{"err - invalid or column name not empty", reflect.TypeOf(struct {
			Or BStruct `gorm:"column:b; query_expr:or"`
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("struct field(Or) with query_expr(or) cannot set column tag", err.Error())
			return false
		}},
		{"err - invalid or must be struct/slice", reflect.TypeOf(struct {
			Or int `gorm:"column:b; query_expr:or"`
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("struct field(Or) with query_expr(or) cannot set column tag", err.Error())
			return false
		}},
		{"err - invalid or must be struct/slice", reflect.TypeOf(struct {
			Or []int `gorm:"query_expr:or"`
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("struct field(Or) with query_expr(or) must be struct or it's list", err.Error())
			return false
		}},
		{"err - invalid or must be struct/slice", reflect.TypeOf(struct {
			Or int `gorm:"query_expr:or"`
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("struct field(Or) with query_expr(or) must be struct or it's list", err.Error())
			return false
		}},
		{"err - invalid must set column name", reflect.TypeOf(struct {
			A string `gorm:"query_expr:="`
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("struct field(A) need column tag", err.Error())
			return false
		}},
		{"err - in with not slice type", reflect.TypeOf(struct {
			A string `gorm:"column:a; query_expr:in"`
		}{}), &structType{}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.NotNil(err)
			as.Equal("struct field(A) with in query_expr must be slice/array", err.Error())
			return false
		}},

		{"empty - basetype", reflect.TypeOf(struct {
			A int `json:"a"`
		}{}), &structType{
			Names: []string{},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},
		{"empty - pointer", reflect.TypeOf(struct {
			A *int `json:"a"`
		}{}), &structType{
			Names: []string{},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},

		{"ok - one field - basetype", reflect.TypeOf(struct {
			A int `gorm:"column:a"`
		}{}), &structType{
			Names: []string{"A"},
			Fields: map[string]*fieldType{
				"A": {Name: "A", Column: "a", Kind: reflect.Int, Tag: map[string]string{"COLUMN": "a"}},
			},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},

		{"ok - one field - pointer", reflect.TypeOf(struct {
			A *int `gorm:"column:a"`
		}{}), &structType{
			Names: []string{"A"},
			Fields: map[string]*fieldType{
				"A": {Name: "A", Column: "a", Kind: reflect.Pointer, Tag: map[string]string{"COLUMN": "a"}},
			},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},
		{"ok - one field - struct is pointer", reflect.TypeOf(&struct {
			A int `gorm:"column:a"`
		}{}), &structType{
			Names: []string{"A"},
			Fields: map[string]*fieldType{
				"A": {Name: "A", Column: "a", Kind: reflect.Int, Tag: map[string]string{"COLUMN": "a"}},
			},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},
		{"ok - two field", reflect.TypeOf(struct {
			A int     `gorm:"column:a"`
			B *string `gorm:"column:b"`
		}{}), &structType{
			Names: []string{"A", "B"},
			Fields: map[string]*fieldType{
				"A": {Name: "A", Column: "a", Kind: reflect.Int, Tag: map[string]string{"COLUMN": "a"}},
				"B": {Name: "B", Column: "b", Kind: reflect.Pointer, Tag: map[string]string{"COLUMN": "b"}},
			},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},
		{"ok - rewrite field", reflect.TypeOf(struct {
			A *int `gorm:"column:a"`
			AStruct
		}{}), &structType{
			Names: []string{"A"},
			Fields: map[string]*fieldType{
				"A": {
					Name: "A", Column: "a", Kind: reflect.Pointer,
					QueryExpr: ">", Tag: map[string]string{
						"COLUMN": "a", "QUERY_EXPR": ">",
					},
				},
			},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},
		{"ok - anonymous", reflect.TypeOf(struct {
			BStruct
		}{}), &structType{
			Names: []string{"B"},
			Fields: map[string]*fieldType{
				"B": {
					Name: "B", Column: "b",
					Kind:        reflect.Ptr,
					IsAnonymous: false,
					QueryExpr:   "!=", Tag: map[string]string{"COLUMN": "b", "QUERY_EXPR": "!="},
				},
			},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},
		{"ok - anonymous - pointer", reflect.TypeOf(struct {
			B int `gorm:"column:b; query_expr:!="`
			*BStruct
		}{}), &structType{
			Names: []string{"B"},
			Fields: map[string]*fieldType{
				"B": {
					Name: "B", Column: "b",
					Kind:        reflect.Ptr,
					IsAnonymous: false,
					QueryExpr:   "!=", Tag: map[string]string{"COLUMN": "b", "QUERY_EXPR": "!="},
				},
			},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},
		{"ok - anonymous'2s", reflect.TypeOf(struct {
			B int `gorm:"column:b; query_expr:!="`
			*CStruct
		}{}), &structType{
			Names: []string{"B"},
			Fields: map[string]*fieldType{
				"B": {
					Name: "B", Column: "b",
					Kind:        reflect.Ptr,
					IsAnonymous: false,
					QueryExpr:   "!=", Tag: map[string]string{"COLUMN": "b", "QUERY_EXPR": "!="},
				},
			},
		}, func(t assert.TestingT, err error, i ...interface{}) bool {
			as.Nil(err)
			return true
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name+"_parseTypeNoCache", func(t *testing.T) {
			got, err := parseStructTypeNoCache(tt.args)
			if !tt.wantErr(t, err, fmt.Sprintf("parseStructTypeNoCache(%v)", tt.args)) {
				return
			}
			assertStructTypeEqual(t, got, tt.want, fmt.Sprintf("parseStructTypeNoCache(%v)", tt.args))
		})
		t.Run(tt.name+"_parseType", func(t *testing.T) {
			got, err := parseStructType(tt.args)
			if !tt.wantErr(t, err, fmt.Sprintf("parseStructTypeNoCache(%v)", tt.args)) {
				return
			}
			assertStructTypeEqual(t, got, tt.want, fmt.Sprintf("parseStructTypeNoCache(%v)", tt.args))
		})
	}
}

func assertStructTypeEqual(t assert.TestingT, a, b *structType, msg string) {
	as := assert.New(t)

	if a == nil {
		as.Nil(b, msg)
		return
	}

	as.Equal(len(a.Names), len(b.Names), msg)

	sort.Strings(a.Names)
	sort.Strings(b.Names)
	as.Equal(a.Names, b.Names, msg)

	as.Equal(len(a.Fields), len(b.Fields), msg)
	for _, name := range a.Names {
		as.NotNil(a.Fields[name], msg)
		as.NotNil(b.Fields[name], msg)
		assertFieldTypeEqual(t, a.Fields[name], b.Fields[name], fmt.Sprintf("name:%s, val:%v; name:%s, val:%v; %s", name, a.Fields[name], name, b.Fields[name], msg))
	}
}

func assertFieldTypeEqual(t assert.TestingT, a, b *fieldType, msg string) {
	as := assert.New(t)

	as.NotNil(a)
	as.NotNil(b)

	as.Equal(a.Name, b.Name, msg)
	as.Equal(a.Column, b.Column, msg)
	as.Equal(a.QueryExpr, b.QueryExpr, msg)
	as.Equal(a.UpdateExpr, b.UpdateExpr, msg)
	as.Equal(a.IsAnonymous, b.IsAnonymous, msg)
	as.Equal(a.Kind, b.Kind, msg)
	if a.OrType == nil {
		as.Nil(b.OrType)
	} else {
		as.NotNil(b.OrType)
		as.Equal(a.OrType.String(), b.OrType.String())
	}

	as.Equal(len(a.Tag), len(b.Tag), msg)
	for k, v := range a.Tag {
		as.Equal(v, b.Tag[k], msg)
	}
}
