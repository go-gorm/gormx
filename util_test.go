package gormx

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_packPanicError(t *testing.T) {
	as := assert.New(t)

	t.Run("panic-string", func(t *testing.T) {
		defer func() {
			err := packPanicError(recover())
			as.NotNil(err)
			as.Equal("gorm querier panic: test", err.Error())
		}()

		panic("test")
	})

	t.Run("panic-error", func(t *testing.T) {
		defer func() {
			err := packPanicError(recover())
			as.NotNil(err)
			as.Equal("error", err.Error())
		}()

		panic(fmt.Errorf("error"))
	})
}

func Test_interfaceToSlice(t *testing.T) {
	tests := []struct {
		name string
		args any
		want []any
		err  error
	}{
		{"1", []int{1, 2, 3}, []any{1, 2, 3}, nil},
		{"2", [...]int{1, 2, 3}, []any{1, 2, 3}, nil},
		{"3", "string", nil, fmt.Errorf("gorm querier panic: interfaceToSlice: v must be slice")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				assert.Equalf(t, tt.want, interfaceToSlice(tt.args), "interfaceToSlice(%v)", tt.args)
			} else {
				defer func() {
					err := packPanicError(recover())
					assert.NotNil(t, err)
					assert.Equal(t, tt.err.Error(), err.Error())
				}()
				interfaceToSlice(tt.args)
			}
		})
	}
}

func Test_isEmptyValue(t *testing.T) {
	tests := []struct {
		name string
		args any
		want bool
	}{
		{"1", nil, true},

		{"int-0", 0, true},
		{"int-1", 1, false},

		{"uint-0", uint(0), true},
		{"uint-1", uint(1), false},

		{"float-0.0", 0.0, true},
		{"float-1.0", 1.0, false},

		{"complex-0+0i", 0 + 0i, true},
		{"complex-1+0i", 1 + 0i, false},

		{"bool-false", false, true},
		{"bool-true", true, false},

		{"string-''", "", true},
		{"string-str", "str", false},

		{"slice-[]", []int{}, true},
		{"slice-[1]", []int{1}, false},

		{"map-{}", map[string]int{}, true},
		{"map-{1}", map[string]int{"1": 1}, false},

		{"struct-{}", struct{}{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.args)
			assert.Equalf(t, tt.want, isEmptyValue(v), "isEmptyValue(%v)", tt.args)
		})
	}
}

func Test_getValueAndType(t *testing.T) {
	as := assert.New(t)

	t.Run("invalid", func(t *testing.T) {
		_, _, err := getValueAndType(nil)
		as.NotNil(err)
		as.Equal("querier's data is invalid", err.Error())
	})

	t.Run("pointer", func(t *testing.T) {
		_, _, _ = getValueAndType(&struct{}{})
	})

	t.Run("not struct", func(t *testing.T) {
		_, _, err := getValueAndType(1)
		as.NotNil(err)
		as.Equal("data's kind must be struct, but got 'int'", err.Error())
	})
}
