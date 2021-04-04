package sql

import (
	"fmt"
	"strings"

	"github.com/yaoapp/xun/dbal"
)

// CompileSelect Compile a select query into SQL.
func (grammarSQL SQL) CompileSelect(query *dbal.Query) string {
	bindingOffset := 0
	return grammarSQL.CompileSelectOffset(query, &bindingOffset)
}

// CompileSelectOffset Compile a select query into SQL.
func (grammarSQL SQL) CompileSelectOffset(query *dbal.Query, offset *int) string {

	sqls := map[string]string{}

	// If the query does not have any columns set, we'll set the columns to the
	// * character to just get all of the columns from the database. Then we
	// can build the query and concatenate all the pieces together as one.
	columns := query.Columns
	if len(columns) == 0 {
		query.AddColumn(grammarSQL.Raw("*"))
	}

	// To compile the query, we'll spin through each component of the query and
	// see if that component exists. If it does we'll just call the compiler
	// function for the component which is responsible for making the SQL.

	sqls["aggregate"] = grammarSQL.compileAggregate(query, query.Aggregate)
	sqls["columns"] = grammarSQL.compileColumns(query, query.Columns)
	sqls["from"] = grammarSQL.compileFrom(query, query.From)
	sqls["joins"] = grammarSQL.compileJoins(query, query.Joins, offset)
	sqls["wheres"] = grammarSQL.compileWheres(query, query.Wheres, offset)
	// sqls["groups"] = grammarSQL.compileGroups()
	// sqls["havings"] = grammarSQL.compileHavings()
	// sqls["orders"] = grammarSQL.compileOrders()
	// sqls["limit"] = grammarSQL.compileLimit()
	// sqls["offset"] = grammarSQL.compileOffset()
	// sqls["lock"] = grammarSQL.compileLock()

	sql := ""
	for _, name := range []string{"aggregate", "columns", "from", "joins", "wheres", "groups", "havings", "orders", "limit", "offset", "lock"} {
		segment, has := sqls[name]
		if has && segment != "" {
			sql = sql + segment + " "
		}
	}

	// Compile unions
	if len(query.Unions) > 0 {
		sql = fmt.Sprintf("%s %s", grammarSQL.WrapUnion(sql), grammarSQL.compileUnions(query, query.Unions, offset))
	}

	// reset columns
	query.Columns = columns
	return strings.Trim(sql, " ")
}

// compileAggregate Compile an aggregated select clause.
func (grammarSQL SQL) compileAggregate(query *dbal.Query, aggregate dbal.Aggregate) string {

	if aggregate.Func == "" {
		return ""
	}

	column := grammarSQL.Columnize(aggregate.Columns)

	// If the query has a "distinct" constraint and we're not asking for all columns
	// we need to prepend "distinct" onto the column name so that the query takes
	// it into account when it performs the aggregating operations on the data.
	if len(query.DistinctColumns) > 0 {
		column = fmt.Sprintf("distinct %s", grammarSQL.Columnize(query.DistinctColumns))
	} else if query.Distinct && column != "*" {
		column = fmt.Sprintf("distinct %s", column)
	}
	return fmt.Sprintf("select %s(%s) as aggregate", aggregate.Func, column)
}

// compileUnions  Compile the "union" queries attached to the main query.
func (grammarSQL SQL) compileUnions(query *dbal.Query, unions []dbal.Union, offset *int) string {
	sql := ""
	for _, union := range unions {
		sql = sql + grammarSQL.compileUnion(query, union, offset)
	}

	// unionOrders

	// unionLimit

	// unionOffset

	return strings.TrimPrefix(sql, " ")
}

// compileUnion Compile a single union statement.
func (grammarSQL SQL) compileUnion(query *dbal.Query, union dbal.Union, offset *int) string {
	conjunction := "union "
	if union.All {
		conjunction = "union all "
	}
	return fmt.Sprintf("%s%s", conjunction, grammarSQL.WrapUnion(grammarSQL.CompileSelectOffset(union.Query, offset)))
}

// compileJoins Compile the "join" portions of the query.
func (grammarSQL SQL) compileJoins(query *dbal.Query, joins []dbal.Join, offset *int) string {
	sql := ""
	for _, join := range joins {
		table := grammarSQL.WrapTable(join.Table)
		nestedJoins := " "
		if len(join.Query.Joins) > 0 {
			nestedJoins = grammarSQL.compileJoins(query, join.Query.Joins, offset)
		}
		tableAndNestedJoins := table
		if len(join.Query.Joins) > 0 {
			tableAndNestedJoins = fmt.Sprintf("(%s%s)", table, nestedJoins)
		}

		return strings.Trim(
			fmt.Sprintf("%s join %s %s", join.Type, tableAndNestedJoins, grammarSQL.compileWheres(join.Query, join.Query.Wheres, offset)),
			" ",
		)
	}
	return sql
}

// compileColumns Compile the "select *" portion of the query.
func (grammarSQL SQL) compileColumns(query *dbal.Query, columns []interface{}) string {

	// If the query is actually performing an aggregating select, we will let that
	// compiler handle the building of the select clauses, as it will need some
	// more syntax that is best handled by that function to keep things neat.
	if query.Aggregate.Func != "" {
		return ""
	}

	sql := "select"
	if query.Distinct {
		sql = "select distinct"
	}
	return fmt.Sprintf("%s %s", sql, grammarSQL.Columnize(columns))
}

//  Compile the "from" portion of the query.
func (grammarSQL SQL) compileFrom(query *dbal.Query, table dbal.Name) string {
	if table.As() != "" {
		return fmt.Sprintf("from %s as %s", grammarSQL.ID(table.Fullname()), grammarSQL.ID(table.As()))
	}
	return fmt.Sprintf("from %s", grammarSQL.ID(table.Fullname()))
}

func (grammarSQL SQL) compileWheres(query *dbal.Query, wheres []dbal.Where, bindingOffset *int) string {

	// Each type of where clauses has its own compiler function which is responsible
	// for actually creating the where clauses SQL. This helps keep the code nice
	// and maintainable since each clause has a very small method that it uses.
	if wheres == nil {
		return ""
	}

	clauses := []string{}
	// If we actually have some where clauses, we will strip off the first boolean
	// operator, which is added by the query builders for convenience so we can
	// avoid checking for the first clauses in each of the compilers methods.
	for _, where := range wheres {
		// fmt.Printf("CompileSelect: %s %s %s %v\n", where.Boolean, where.Type, where.Operator, where.Value)
		boolen := strings.ToLower(where.Boolean)
		switch where.Type {
		case "basic":
			clauses = append(clauses, fmt.Sprintf("%s %s", boolen, grammarSQL.whereBasic(query, where, bindingOffset)))
			break
		case "null":
			clauses = append(clauses, fmt.Sprintf("%s %s", boolen, grammarSQL.whereNull(query, where)))
		case "notnull":
			clauses = append(clauses, fmt.Sprintf("%s %s", boolen, grammarSQL.whereNotNull(query, where)))
		case "column":
			clauses = append(clauses, fmt.Sprintf("%s %s", boolen, grammarSQL.whereColumn(query, where)))
		case "sub":
			clauses = append(clauses, fmt.Sprintf("%s %s", boolen, grammarSQL.whereSub(query, where, bindingOffset)))
			break
		case "nested":
			clauses = append(clauses, fmt.Sprintf("%s %s", boolen, grammarSQL.whereNested(query, where, bindingOffset)))
			break
		}
	}

	conjunction := "where"
	if query.IsJoinClause {
		conjunction = "on"
	}

	return fmt.Sprintf("%s %s", conjunction, grammarSQL.RemoveLeadingBoolean(strings.Join(clauses, " ")))
}

// whereBasic Compile a date based where clause.
func (grammarSQL SQL) whereBasic(query *dbal.Query, where dbal.Where, bindingOffset *int) string {
	*bindingOffset = *bindingOffset + where.Offset
	value := grammarSQL.Parameter(where.Value, *bindingOffset)
	operator := strings.ReplaceAll(where.Operator, "?", "??")
	return fmt.Sprintf("%s %s %s", grammarSQL.Wrap(where.Column), operator, value)
}

// whereColumn Compile a where clause comparing two columns.
func (grammarSQL SQL) whereColumn(query *dbal.Query, where dbal.Where) string {
	return fmt.Sprintf("%s %s %s", grammarSQL.Wrap(where.First), where.Operator, grammarSQL.Wrap(where.Second))
}

// whereNested Compile a nested where clause.
func (grammarSQL SQL) whereNested(query *dbal.Query, where dbal.Where, bindingOffset *int) string {

	offset := 6 // - where
	if query.IsJoinClause {
		offset = 3 // - on
	}

	sql := grammarSQL.compileWheres(where.Query, where.Query.Wheres, bindingOffset)
	end := len(sql)
	if end > offset {
		sql = sql[offset:end]
	}
	return fmt.Sprintf("(%s)", sql)
}

// whereSub Compile a where condition with a sub-select.
func (grammarSQL SQL) whereSub(query *dbal.Query, where dbal.Where, bindingOffset *int) string {
	selectSQL := grammarSQL.CompileSelectOffset(where.Query, bindingOffset)
	return fmt.Sprintf("%s %s (%s)", grammarSQL.Wrap(where.Column), where.Operator, selectSQL)
}

// whereNull Compile a "where null" clause.
func (grammarSQL SQL) whereNull(query *dbal.Query, where dbal.Where) string {
	return fmt.Sprintf("%s is null", grammarSQL.Wrap(where.Column))
}

// whereNotNull Compile a "where not null" clause.
func (grammarSQL SQL) whereNotNull(query *dbal.Query, where dbal.Where) string {
	return fmt.Sprintf("%s is not null", grammarSQL.Wrap(where.Column))
}

// Utils for compiling

// RemoveLeadingBoolean Remove the leading boolean from a statement.
func (grammarSQL SQL) RemoveLeadingBoolean(value string) string {
	value = strings.TrimPrefix(value, "and ")
	value = strings.TrimPrefix(value, "or ")
	return value
}

// Raw make a new expression
func (grammarSQL SQL) Raw(value string) dbal.Expression {
	return dbal.NewExpression(value)
}
