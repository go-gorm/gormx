package gormx

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Query(where any) clause.Expression {
	expression, err := buildSQLWhere(where)
	if err != nil {
		return errExpression{err}
	}
	return expression
}

func Update(update any) gorm.StatementModifier {
	return &updateModifyStatement{update}
}

type updateModifyStatement struct {
	update any
}

var _ gorm.StatementModifier = (*updateModifyStatement)(nil)

func (u updateModifyStatement) ModifyStatement(stmt *gorm.Statement) {
	m, err := buildSQLUpdate(u.update)
	if err != nil {
		_ = stmt.AddError(err)
		return
	}
	stmt.Dest = m
}
