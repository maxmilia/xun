package query

import (
	"reflect"

	"github.com/yaoapp/xun/dbal"
)

//Join  Add a join clause to the query.
func (builder *Builder) Join(table string, first interface{}, args ...interface{}) Query {
	operator, second, _, _ := builder.joinPrepare(args...)
	return builder.join(dbal.NewName(table), "", first, operator, second, "inner", "on", 0)
}

//JoinSub Add a subquery join clause to the query.
func (builder *Builder) JoinSub(qb interface{}, alias string, first interface{}, args ...interface{}) Query {
	operator, second, _, _ := builder.joinPrepare(args...)
	return builder.joinSub(qb, alias, first, operator, second, "inner", "on", 0)
}

// LeftJoin Add a left join to the query.
func (builder *Builder) LeftJoin() {
}

// RightJoin Add a right join to the query.
func (builder *Builder) RightJoin() {
}

// CrossJoin Add a "cross join" clause to the query.
func (builder *Builder) CrossJoin() {
}

// forJoinClause Create a new query instance for a join clause.
func (builder *Builder) forJoinClause() *Builder {
	new := builder.new()
	new.Query.IsJoinClause = true
	return new
}

func (builder *Builder) joinPrepare(args ...interface{}) (string, interface{}, string, string) {
	var operator = ""
	var second interface{} = nil
	var typ = "inner"
	var method = "on"

	if len(args) > 0 && reflect.TypeOf(args[0]).Kind() == reflect.String {
		operator = args[0].(string)
	}

	if len(args) > 1 {
		second = args[1]
	}

	if len(args) > 2 && reflect.TypeOf(args[2]).Kind() == reflect.String {
		typ = args[2].(string)
	}

	if len(args) > 3 && reflect.TypeOf(args[3]).Kind() == reflect.String {
		method = args[3].(string)
	}

	return operator, second, typ, method
}

//JoinWhere Add a "join where" clause to the query.
func (builder *Builder) JoinWhere() {
}

// LeftJoinWhere Add a "join where" clause to the query.
func (builder *Builder) LeftJoinWhere() {
}

// RightJoinWhere Add a "right join where" clause to the query.
func (builder *Builder) RightJoinWhere() {
}

//Join Add a join clause to the query.
func (builder *Builder) join(table interface{}, alias string, first interface{}, operator string, second interface{}, typ string, method string, offset int) Query {
	if method == "on" {
		qb := builder.forJoinClause()
		join := qb.joinOn(typ, first, operator, second, "and", offset)
		join.Name = table
		if alias != "" {
			join.Alias = alias
			join.SQL = dbal.Raw(table).GetValue()
		}
		builder.Query.Joins = append(builder.Query.Joins, join)
		builder.Query.AddBinding("join", qb.GetBindings())
	}
	return builder
}

// joinSub Add a subquery join clause to the query.
func (builder *Builder) joinSub(qb interface{}, alias string, first interface{}, operator string, second interface{}, typ string, method string, offset int) Query {
	sub, bindings, joinOffset := builder.createSub(qb)
	builder.Query.AddBinding("join", bindings)
	return builder.join(sub, alias, first, operator, second, typ, method, joinOffset+offset)
}

func (builder *Builder) joinOn(typ string, first interface{}, operator string, second interface{}, boolean string, offset int) dbal.Join {

	if builder.isClosure(first) {
		builder.whereNested(first.(func(qb Query)), boolean)
	} else {
		builder.WhereColumn(first, operator, second, boolean)
	}

	return dbal.Join{
		Type:   typ, // inner left join
		Query:  builder.Query,
		Offset: offset,
	}
}
