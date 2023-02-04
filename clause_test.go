package gormx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/clause"
)

func Test_Clause(t *testing.T) {
	as := assert.New(t)
	db := newDB()

	t.Run("no in", func(t *testing.T) {
		res := notIn{clause.IN{Column: "name", Values: []any{1, 2}}}
		res.Build(db.Statement)
		as.Equal("`name` NOT IN (?,?)", db.Statement.SQL.String())
	})
}
