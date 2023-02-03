package gormx

import (
	"fmt"
	"reflect"
)

func getValueAndType(structData interface{}) (reflect.Value, reflect.Type, error) {
	rv := reflect.ValueOf(structData)

	if !rv.IsValid() {
		return reflect.Value{}, nil, fmt.Errorf("querier's data is invalid")
	}

	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, nil, fmt.Errorf("data's kind must be struct, but got '%s'", rv.Kind())
	}

	rt := rv.Type()
	return rv, rt, nil
}

func packPanicError(r interface{}) (err error) {
	switch je := r.(type) {
	case error:
		return je
	default:
		return fmt.Errorf("gorm querier panic: %s", r)
	}
}

func interfaceToSlice(v any) []any {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		panic("interfaceToSlice: v must be slice")
	}

	sliceType := rv.Type().Elem()
	slice := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		x := reflect.New(sliceType).Elem()
		x.Set(rv.Index(i))
		slice[i] = x.Interface()
	}
	return slice
}

func isEmptyValue(rv reflect.Value) bool {
	// data may be string, int, *string, slice, check data is empty
	// example: "", 0, nil, []string{}, []int{}, []*string{}, []*int{}

	switch rv.Kind() {
	case reflect.String:
		return rv.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Interface, reflect.Ptr:
		return rv.IsNil()
	case reflect.Invalid:
		return true
	case reflect.Complex64, reflect.Complex128:
		return rv.Complex() == 0
	case reflect.Slice, reflect.Array, reflect.Map:
		return rv.IsNil() || rv.Len() == 0
	default:
		return false
	}
}
