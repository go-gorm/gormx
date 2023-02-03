package gormx

import (
	"fmt"
	"reflect"
	"sync"

	"gorm.io/gorm/schema"
)

const (
	tagColumn = "COLUMN"
	tagQuery  = "QUERY_EXPR"
	tagUpdate = "UPDATE_EXPR"
)

var structTypeCacheMap sync.Map

type structType struct {
	Names  []string
	Fields map[string]*fieldType
}

type fieldType struct {
	Name        string            // field name
	Column      string            // tag sql_field
	QueryExpr   string            // tag query_expr
	UpdateExpr  string            // tag update_expr
	IsAnonymous bool              // field 是否是匿名字段
	Kind        reflect.Kind      // field Kind
	OrType      reflect.Type      // field OrType
	Tag         map[string]string // key: COLUMN etc.
}

func parseStructType(t reflect.Type) (*structType, error) {
	structType := loadStructTypeFromCache(t)
	if structType != nil {
		return structType, nil
	}
	parsedType, err := parseStructTypeNoCache(t)
	if err != nil {
		return nil, err
	}
	structTypeCacheMap.Store(t, parsedType)
	return parsedType, nil
}

func parseStructTypeNoCache(t reflect.Type) (_ *structType, err error) {
	return parseStructTypeRev(t, nil, false)
}

func parseStructTypeRev(t reflect.Type, sType *structType, isField bool) (_ *structType, err error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if isField && (t.Kind() != reflect.Struct && t.Kind() != reflect.Slice) {
		return nil, fmt.Errorf("field's type must be struct/slice, but got %s", t.Kind())
	}

	if sType == nil {
		sType = &structType{
			Names:  []string{},
			Fields: map[string]*fieldType{},
		}
	}
	for i := 0; i < t.NumField(); i++ {
		structField := t.Field(i)
		tag := schema.ParseTagSetting(structField.Tag.Get("gorm"), ";")
		columnName := tag[tagColumn]
		queryExprString := tag[tagQuery]
		updateExprString := tag[tagUpdate]
		ft := structField.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		isOr := isColumnEmpty(columnName) && queryExprString == operatorOr
		if isColumnEmpty(columnName) {
			if structField.Anonymous {
				// 匿名字段，不跳过
			} else if queryExprString == "" && updateExprString == "" {
				// 没有 query_expr 和 update_expr，跳过
				continue
			}
		}
		if err := checkField(structField, columnName, queryExprString, ft); err != nil {
			return nil, err
		}

		// 匿名字段, 将匿名字段的字段加入到当前结构体中
		if structField.Anonymous {
			if err := parseAnonymousStructField(structField, sType); err != nil {
				return nil, err
			}
		} else {
			var fieldStructType reflect.Type
			if isOr {
				if ft.Kind() == reflect.Slice {
					ft = ft.Elem()
				}
				fieldStructType = ft
			}
			if err := parseNormalStructField(structField, queryExprString, updateExprString, columnName, tag, fieldStructType, sType); err != nil {
				return nil, err
			}
		}
	}
	return sType, nil
}

func reOrderNames(sType *structType, name string) {
	for idx, n := range sType.Names {
		if n == name {
			copy(sType.Names[idx:], sType.Names[idx+1:])
			sType.Names[len(sType.Names)-1] = name
			break
		}
	}
}

func checkField(structField reflect.StructField, columnName, queryExprString string, ft reflect.Type) error {
	// 匿名
	if structField.Anonymous {
		if !isColumnEmpty(columnName) {
			return fmt.Errorf("field %s is anonymous that can not have column tag", structField.Name)
		}
	}

	// or
	if queryExprString == operatorOr {
		if !isColumnEmpty(columnName) {
			return fmt.Errorf("struct field(%s) with query_expr(%s) cannot set column tag", structField.Name, queryExprString)
		}
		if ft.Kind() == reflect.Struct {
		} else if ft.Kind() == reflect.Slice || ft.Kind() == reflect.Array {
			if ft.Elem().Kind() != reflect.Struct {
				return fmt.Errorf("struct field(%s) with query_expr(%s) must be struct or it's list", structField.Name, queryExprString)
			}
		} else {
			return fmt.Errorf("struct field(%s) with query_expr(%s) must be struct or it's list", structField.Name, queryExprString)
		}
	}

	// 非匿名
	if !structField.Anonymous {
		if isColumnEmpty(columnName) && queryExprString != operatorOr {
			return fmt.Errorf("struct field(%s) need column tag", structField.Name)
		}
	}

	// op 和 类型对应
	rt := structField.Type
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	switch queryExprString {
	case operatorIn:
		if rt.Kind() != reflect.Slice && rt.Kind() != reflect.Array {
			return fmt.Errorf("struct field(%s) with in query_expr must be slice/array", structField.Name)
		}
	case operatorEq:
		if rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array {
			return fmt.Errorf("struct field(%s) with eq query_expr can not be slice/array", structField.Name)
		}
	}

	return nil
}

func checkQueryExpr(field reflect.StructField, q string) error {
	if q != "" {
		if _, ok := queryExprMap[q]; !ok {
			return fmt.Errorf("field(%s) query_expr(%s) invalid", field.Name, q)
		}
	}

	return nil
}

func checkUpdateExpr(field reflect.StructField, q string) error {
	if q != "" {
		if _, ok := updaterMap[q]; !ok {
			return fmt.Errorf("field(%s) update_expr(%s) invalid", field.Name, q)
		}
	}
	return nil
}

func loadStructTypeFromCache(t reflect.Type) *structType {
	v, ok := structTypeCacheMap.Load(t)
	if ok {
		sqlType := v.(*structType)
		return sqlType
	}
	return nil
}

func isColumnEmpty(column string) bool {
	return column == "" || column == "-"
}

func parseAnonymousStructField(structField reflect.StructField, sType *structType) error {
	t := structField.Type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	childSType, err := parseStructTypeRev(t, sType, true)
	if err != nil {
		return err
	}
	for kk, vv := range childSType.Fields {
		if _, ok := sType.Fields[kk]; !ok {
			sType.Fields[kk] = vv
			sType.Names = append(sType.Names, kk)
		}
	}
	return nil
}

func parseNormalStructField(structField reflect.StructField, queryExprString string, updateExprString string, columnName string, tag map[string]string, fieldStructType reflect.Type, sType *structType) error {
	if err := checkQueryExpr(structField, queryExprString); err != nil {
		return err
	}
	if err := checkUpdateExpr(structField, updateExprString); err != nil {
		return err
	}
	column := &fieldType{
		Name:        structField.Name,
		Column:      columnName,
		QueryExpr:   queryExprString,
		UpdateExpr:  updateExprString,
		IsAnonymous: structField.Anonymous,
		Kind:        structField.Type.Kind(),
		OrType:      fieldStructType,
		Tag:         tag,
	}
	// 已经有这个 name 的 field 这说明需要覆盖
	if _, ok := sType.Fields[structField.Name]; ok {
		reOrderNames(sType, structField.Name)
	} else {
		sType.Names = append(sType.Names, structField.Name)
	}
	sType.Fields[structField.Name] = column

	return nil
}
