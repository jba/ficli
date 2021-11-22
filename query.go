// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/jba/lexer"
)

type query struct {
	selects []string
	coll    string
	orders  []string
	desc    bool
	limit   int
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

func init() {
	lc.Install(unicode.IsSpace, lexer.SkipWhile(unicode.IsSpace))
	lc.Install(unicode.IsLetter, lexer.ReadWhile(isIdent))
	lc.Install(unicode.IsDigit, lexer.ReadWhile(unicode.IsDigit))
	for _, r := range "+-().,*" {
		lc.Install(lexer.IsRune(r), lexer.ReadRune(r))
	}
	lc.Install(lexer.IsRune('>'), lexer.ReadOneOrTwo('='))
	lc.Install(lexer.IsRune('<'), lexer.ReadOneOrTwo('='))

	ident := Is("identifier", Ident)
	identList := List(ident, Lit(","))

	queryParser = And(
		Lit("select"),
		Do(
			Or(Lit("*"), identList),
			func(toks []string) error {
				if toks[0] != "*" {
					parsedQuery.selects = identsFromList(toks)
				}
				return nil
			}),
		Lit("from"),
		Do(ident, func(toks []string) error { parsedQuery.coll = toks[0]; return nil }),
		Optional(And(
			Lit("order"), Commit, Lit("by"),
			Do(identList, func(toks []string) error {
				parsedQuery.orders = identsFromList(toks)
				return nil
			}),
			Do(
				Or(Lit("asc"), Lit("desc"), Empty),
				func(toks []string) error {
					parsedQuery.desc = (len(toks) > 0 && toks[0] == "desc")
					return nil
				}),
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

func isIdent(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
