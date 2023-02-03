package gormx

import "gorm.io/gorm/clause"

type notIn struct {
	in clause.IN
}

func (in notIn) Build(builder clause.Builder) {
	in.in.NegationBuild(builder)
}

type errExpression struct {
	err error
}

func (e errExpression) Build(builder clause.Builder) {
	_ = builder.AddError(e.err)
}
