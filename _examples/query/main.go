package main

import (
	"fmt"

	"gorm.io/gorm"

	"gorm.io/gormx"
)

type User struct {
	Name string `gorm:"column:name"`
	Age  uint   `gorm:"column:age"`
}

type Where struct {
	Name  *string  `gorm:"column:name;"`
	AgeGT *uint    `gorm:"column:age; query_expr:>"`
	Or    *WhereOr `gorm:"query_expr:or"`
}

type WhereOr struct {
	Name  *string `gorm:"column:name;"`
	AgeGT *uint   `gorm:"column:age; query_expr:>"`
}

type Update struct {
	Age   uint       `gorm:"column:age; update_expr:+"`
	Extra *ExtraInfo `gorm:"column:extra; update_expr:merge_json"`
}

type ExtraInfo struct {
	City string `json:"city"`
}

func queryExample(db *gorm.DB) {
	fmt.Println(db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		res := tx.Table("users").Where(gormx.Query(Where{
			Name:  ptr("a"),
			AgeGT: ptr(uint(10)),
			Or: &WhereOr{
				Name:  ptr("or-name"),
				AgeGT: ptr(uint(20)),
			},
		})).Find(&[]User{})
		if res.Error != nil {
			panic(res.Error)
		}
		return res
	}))
}

func ptr[T any](v T) *T {
	return &v
}
