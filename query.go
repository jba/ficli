// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"io"
	"strconv"
	"strings"
	"unicode"

	"cloud.google.com/go/firestore"
	"github.com/jba/lexer"
)

type query struct {
	selects []string
	coll    string
	wheres  []where
	orders  []order
	limit   int
}

type where struct {
	path, op, value string
}

type order struct {
	path string
	dir  firestore.Direction
}

func parseQuery(s string) (*query, error) {
	lex := lexer.New(strings.NewReader(s), &lc)
	var toks []string
	for {
		tok, err := lex.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		toks = append(toks, tok)
	}
	parsedQuery = query{}
	if err := Parse(toks, queryParser); err != nil {
		return nil, err
	}
	q := parsedQuery // make a copy
	return &q, nil
}

// Query syntax:
//     select LIST from ID
//     [where EXPR [(and|or) EXPR]...]
//     order by ID [(asc|desc)]
//     limit N

var (
	lc          lexer.Config
	queryParser parser
	parsedQuery query
)

func noerr(f func([]string)) func([]string) error {
	return func(toks []string) error {
		f(toks)
		return nil
	}
}

func init() {
	lc.Install(unicode.IsSpace, lexer.SkipWhile(unicode.IsSpace))
	lc.Install(unicode.IsLetter, lexer.ReadWhile(isPathRune))
	lc.Install(unicode.IsDigit, lexer.ReadWhile(unicode.IsDigit))
	for _, r := range "+().,*=" {
		lc.Install(lexer.IsRune(r), lexer.ReadRune(r))
	}
	lc.Install(lexer.IsRune('>'), lexer.ReadOneOrTwo('='))
	lc.Install(lexer.IsRune('<'), lexer.ReadOneOrTwo('='))

	ident := Is("identifier", Ident)

	path := Is("path", func(s string) bool {
		for _, r := range s {
			if !isPathRune(r) {
				return false
			}
		}
		return true
	})

	expr := Do(And(ident, Any, Any), noerr(func(toks []string) {
		parsedQuery.wheres = append(parsedQuery.wheres, where{toks[0], toks[1], toks[2]})
	}))

	queryParser = And(
		Lit("select"),
		Or(Lit("*"), Lit("all"), List(
			Do(ident, noerr(func(toks []string) {
				parsedQuery.selects = append(parsedQuery.selects, toks[0])
			})),
			Lit(","))),
		Lit("from"),
		Do(path, noerr(func(toks []string) { parsedQuery.coll = toks[0] })),
		Optional(And(Lit("where"), Commit, List(expr, Lit("and")))),
		Optional(And(
			Lit("order"), Commit, Lit("by"),
			List(
				Do(
					And(ident, Or(Lit("asc"), Lit("desc"), Empty)),
					noerr(func(toks []string) {
						dir := firestore.Asc
						if len(toks) > 1 && toks[1] == "desc" {
							dir = firestore.Desc
						}
						parsedQuery.orders = append(parsedQuery.orders, order{toks[0], dir})
					})),
				Lit(",")),
		)),
		Optional(And(
			Lit("limit"),
			Commit,
			Do(Any, func(toks []string) error {
				n, err := strconv.Atoi(toks[0])
				if err != nil {
					return err
				}
				parsedQuery.limit = n
				return nil
			}))))
}

// skip over commas
func identsFromList(toks []string) []string {
	var ids []string
	for i := 0; i < len(toks); i += 2 {
		ids = append(ids, toks[i])
	}
	return ids
}

func isPathRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '/'
}
