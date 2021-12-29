// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"unicode"

	"cloud.google.com/go/firestore"
	"github.com/jba/parco"
)

type query struct {
	selects []string
	coll    string
	wheres  []where
	orders  []order
	limit   int64
}

type where struct {
	path, op, value string
}

type order struct {
	path string
	dir  firestore.Direction
}

func parseQuery(s string) (*query, error) {
	val, err := queryParser.Parse(s)
	if err != nil {
		return nil, err
	}
	return val.(*query), nil
}

// Query syntax:
//     select LIST from ID
//     [where EXPR [(and|or) EXPR]...]
//     [order by {ID [(asc|desc)]}]
//     [limit N]

var queryParser parco.Parser

func init() {
	type Value = parco.Value

	var (
		And    = parco.And
		Or     = parco.Or
		Word   = parco.Word
		Eq     = parco.Equal
		Opt    = parco.Optional
		Regexp = parco.Regexp

		ident = Regexp("identifier", `[_\pL][_\pL\p{Nd}]*`)

		path = parco.While("path", func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '/'
		})
	)

	// selectFrom parses "select ... from coll" and returns a *query.
	selectFrom := And(
		Word("select"),
		Or(Eq("*"), Word("all"), parco.List(ident, Eq(","))).Do(
			func(v Value) (Value, error) {
				// v is either a single string (from "*" or "all") or a slice of strings.
				q := &query{}
				if _, ok := v.(string); !ok {
					for _, id := range v.([]Value) {
						q.selects = append(q.selects, id.(string))
					}
				}
				return q, nil
			}),
		Word("from"),
		path).Do(func(v Value) (Value, error) {
		vs := v.([]Value)
		q := vs[1].(*query)
		q.coll = vs[3].(string)
		return q, nil
	})

	// whereClause parses "where expr {AND expr}" and returns a []where.
	whereClause := And(
		Word("where"), parco.Cut,
		parco.List(And(ident, Regexp("op", `[=<>]=?`), Regexp("arg", `[^\s]+`)).Do(
			func(vs []Value) Value {
				return where{vs[0].(string), vs[1].(string), vs[2].(string)}
			}), Word("and"))).Do(
		func(vs []Value) Value {
			// vs is ["where", slice of where clauses]
			var ws []where
			for _, w := range vs[1].([]Value) {
				ws = append(ws, w.(where))
			}
			return ws
		})

	// orderByClause parses "order by {id [asc|desc],}" and returns a []order.
	orderByClause := And(
		Word("order"), parco.Cut, Word("by"),
		parco.List(
			And(ident, Or(Word("asc"), Word("desc"), parco.Empty)).Do(
				func(vs []Value) Value {
					dir := firestore.Asc
					if len(vs) > 1 && vs[1] == "desc" {
						dir = firestore.Desc
					}
					return order{vs[0].(string), dir}
				}),
			Eq(","))).Do(
		func(vs []Value) Value {
			// vs is ["order", "by", slice of ords]
			var ords []order
			for _, v := range vs[2].([]Value) {
				ords = append(ords, v.(order))
			}
			return ords
		})

	// limitClause parses "limit N" and returns N.
	limitClause := And(Word("limit"), parco.Cut, parco.Int).Do(func(vs []Value) Value { return vs[1] })

	queryParser = And(
		selectFrom,
		Opt(whereClause),
		Opt(orderByClause),
		Opt(limitClause),
	).Do(func(vs []Value) Value {
		q := vs[0].(*query)
		for _, v := range vs[1:] {
			switch v := v.(type) {
			case []where:
				q.wheres = v
			case []order:
				q.orders = v
			case int64:
				q.limit = v
			default:
				panic("unknown")
			}
		}
		return q
	})
}
