package gormx

import (
	"encoding/json"
	"fmt"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func buildSQLUpdate(opt interface{}) (result map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = packPanicError(r)
		}
	}()

	rv, rt, err := getValueAndType(opt)
	if err != nil {
		return nil, err
	}

	// 针对类型的检查 解析的时候有做
	sqlType, err := parseStructType(rt)
	if err != nil {
		return nil, err
	}

	// 遍历 field，将非 nil 的值拼到 map 中
	return buildUpdateMap(rv, sqlType)
}

// 遍历 field，将非 nil 的值拼到 map 中
func buildUpdateMap(rv reflect.Value, structType *structType) (result map[string]interface{}, err error) {
	result = make(map[string]interface{})
	for _, name := range structType.Names {
		column := structType.Fields[name] // 前置函数已经检查过，一定存在
		data := rv.FieldByName(column.Name)
		// 字段的值是空值, 直接忽略, 不做处理
		if isEmptyValue(data) {
			continue
		}
		if data.Kind() == reflect.Ptr {
			data = data.Elem()
		}
		if column.UpdateExpr != "" {
			updateExprBuilder := updaterMap[column.UpdateExpr] // 前置函数已经检查过，一定存在
			if updaterResult := updateExprBuilder(column.Column, data.Interface()); updaterResult.SQL != "" {
				result[column.Column] = updaterResult
			}
		} else {
			result[column.Column] = data.Interface()
		}
	}

	return result, nil
}

const (
	updateExprAdd       = "+"
	updateExprSub       = "-"
	updateExprMergeJSON = "merge_json"
)

type buildUpdateExpr func(field string, data interface{}) clause.Expr

var updaterMap = map[string]buildUpdateExpr{
	updateExprAdd: func(field string, data interface{}) clause.Expr {
		return gorm.Expr(field+" + ?", data)
	},
	updateExprSub: func(field string, data interface{}) clause.Expr {
		return gorm.Expr(field+" - ?", data)
	},
	updateExprMergeJSON: func(field string, data interface{}) clause.Expr {
		var bs []byte
		if isMergeJSONStruct(data) {
			dataMap, _ := mergeJSONStructToJSONMap(data)
			bs, _ = json.Marshal(dataMap)
		} else {
			bs, _ = json.Marshal(data)
		}
		s := string(bs)
		if s == "" {
			return clause.Expr{}
		}

		return gorm.Expr("CASE WHEN (`"+field+"` IS NULL OR `"+field+"` = '') THEN CAST(? AS JSON) ELSE JSON_MERGE_PATCH(`"+field+"`, CAST(? AS JSON)) END", s, s)
	},
}

func isMergeJSONStruct(v interface{}) bool {
	vt := reflect.TypeOf(v)
	if vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}
	return vt.Kind() == reflect.Struct
}

func mergeJSONStructToJSONMap(v interface{}) (map[string]interface{}, error) {
	vt := reflect.TypeOf(v)
	vv := reflect.ValueOf(v)

	if vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
		vv = vv.Elem()
	}
	if vt.Kind() != reflect.Struct {
		return nil, fmt.Errorf("update(JSON_MERGE_PATCH) need struct type")
	}

	m := map[string]interface{}{}
	for i := 0; i < vt.NumField(); i++ {
		vtField := vt.Field(i)
		vvField := vv.Field(i)

		if !vvField.IsValid() {
			continue
		}

		jsonField := vtField.Tag.Get("json")
		if jsonField == "" || jsonField == "-" {
			continue
		}

		// ptr
		if vtField.Type.Kind() == reflect.Ptr {
			if vvField.IsNil() {
				continue
			}
			m[jsonField] = vvField.Elem().Interface()
		} else {
			m[jsonField] = vvField.Interface()
		}
	}

	return m, nil
}
