package gormx

import (
	"fmt"
	"reflect"

	"gorm.io/gorm/clause"
)

func buildSQLWhere(where interface{}) (expression clause.Expression, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = packPanicError(r)
		}
	}()

	rv, rt, err := getValueAndType(where)
	if err != nil {
		return nil, err
	}

	sqlType, err := parseStructType(rt)
	if err != nil {
		return nil, err
	}

	return buildClauseExpression(rv, sqlType, true)
}

func buildClauseExpression(rv reflect.Value, sqlType *structType, joinAnd bool) (result clause.Expression, err error) {
	expressions := []clause.Expression{}
	for _, name := range sqlType.Names {
		column := sqlType.Fields[name] // 前置步骤检查过，一定存在

		// 计算字段的值
		data := rv.FieldByName(column.Name)
		// 字段的值是 nil 直接忽略 不做处理
		if isEmptyValue(data) {
			continue
		}
		if data.Kind() == reflect.Ptr {
			data = data.Elem()
		}
		inter := data.Interface()

		queryExprBuilder, err := getQueryExpr(column.QueryExpr)
		if err != nil {
			return nil, err
		}

		if column.OrType != nil {
			orType, err := parseStructType(column.OrType)
			if err != nil {
				return nil, err
			}
			if data.Kind() == reflect.Slice {
				list := []clause.Expression{}
				for i := 0; i < data.Len(); i++ {
					or, err := buildClauseExpression(data.Index(i), orType, false)
					if err != nil {
						return nil, err
					} else if or != nil {
						list = append(list, or)
					}
				}
				if len(list) > 0 {
					expressions = append(expressions, joinExpression(list, false))
				}
			} else {
				or, err := buildClauseExpression(data, orType, false)
				if err != nil {
					return nil, err
				} else if or != nil {
					expressions = append(expressions, or)
				}
			}
		} else {
			and := queryExprBuilder(column.Column, inter)
			if and != nil {
				expressions = append(expressions, and)
			}
		}
	}

	return joinExpression(expressions, joinAnd), nil
}

func joinExpression(exprs []clause.Expression, joinAnd bool) clause.Expression {
	if len(exprs) == 1 {
		return exprs[0]
	}
	if joinAnd {
		return clause.And(exprs...)
	}
	return clause.Or(exprs...)
}

const (
	operatorOr   = "or"     // clause.OrConditions
	operatorIn   = "in"     // clause.IN
	operatorNin  = "not in" // notIn // 无 clause.NIN
	operatorGt   = ">"      // clause.Gt
	operatorGte  = ">="     // clause.Gte
	operatorLt   = "<"      // clause.Lt
	operatorLte  = "<="     // clause.Lte
	operatorEq   = "="      // clause.Eq
	operatorNeq  = "!="     // clause.Neq
	operatorLike = "like"   // clause.Like
	operatorNull = "null"   // clause.Null
)

type (
	buildExpression func(field string, data interface{}) clause.Expression
)

var queryExprMap = map[string]struct {
	build buildExpression
}{
	operatorLt: {
		build: func(field string, data interface{}) clause.Expression {
			return clause.Lt{
				Column: clause.Column{Name: field},
				Value:  data,
			}
		},
	},
	operatorLte: {
		build: func(field string, data interface{}) clause.Expression {
			return clause.Lte{
				Column: clause.Column{Name: field},
				Value:  data,
			}
		},
	},
	operatorEq: {
		build: func(field string, data interface{}) clause.Expression {
			return clause.Eq{
				Column: clause.Column{Name: field},
				Value:  data,
			}
		},
	},
	"": {
		build: func(field string, data interface{}) clause.Expression {
			return clause.Eq{
				Column: clause.Column{Name: field},
				Value:  data,
			}
		},
	},
	operatorNeq: {
		build: func(field string, data interface{}) clause.Expression {
			return clause.Neq{
				Column: clause.Column{Name: field},
				Value:  data,
			}
		},
	},
	operatorGt: {
		build: func(field string, data interface{}) clause.Expression {
			return clause.Gt{
				Column: clause.Column{Name: field},
				Value:  data,
			}
		},
	},
	operatorGte: {
		build: func(field string, data interface{}) clause.Expression {
			return clause.Gte{
				Column: clause.Column{Name: field},
				Value:  data,
			}
		},
	},
	operatorNull: {
		build: func(field string, data interface{}) clause.Expression {
			switch v := data.(type) {
			case bool:
				if v {
					return clause.Eq{
						Column: clause.Column{Name: field},
						Value:  nil,
					}
				} else {
					return clause.Neq{
						Column: clause.Column{Name: field},
						Value:  nil,
					}
				}
			}
			return nil
		},
	},
	operatorIn: {
		build: func(field string, data interface{}) clause.Expression {
			return clause.IN{
				Column: clause.Column{Name: field},
				Values: interfaceToSlice(data),
			}
		},
	},
	operatorNin: {
		build: func(field string, data interface{}) clause.Expression {
			return notIn{clause.IN{
				Column: clause.Column{Name: field},
				Values: interfaceToSlice(data),
			}}
		},
	},
	operatorLike: {
		build: func(field string, data interface{}) clause.Expression {
			return clause.Like{
				Column: clause.Column{Name: field},
				Value:  data.(string),
			}
		},
	},
	operatorOr: {},
}

func getQueryExpr(queryExprString string) (buildExpression, error) {
	queryExpr, ok := queryExprMap[queryExprString]
	if !ok {
		return nil, fmt.Errorf("query_expr '%s' invalid", queryExprString)
	}
	return queryExpr.build, nil
}
