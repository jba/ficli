// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"errors"
	"fmt"
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
		Action(
			Or(Lit("*"), identList),
			func(toks []string) error {
				if toks[0] != "*" {
					for i := 0; i < len(toks); i += 2 {
						parsedQuery.selects = append(parsedQuery.selects, toks[i])
					}
				}
				return nil
			}),
		Lit("from"),
		Action(ident, func(toks []string) error { parsedQuery.coll = toks[0]; return nil }),
		// TODO: If this Optional fails because the limit arg is not a number (e.g. "limit b"),
		// then the resulting error is "unconsumed input" rather than the error from the function.
		// Should we add a Cut() parser that prevents backtracking past a certain point?
		Optional(And(
			Lit("limit"),
			Commit,
			Action(Any, func(toks []string) error {
				n, err := strconv.Atoi(toks[0])
				if err != nil {
					return err
				}
				parsedQuery.limit = n
				return nil
			}))))
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

func expect(lex *lexer.Lexer, want string) error {
	tok, err := lex.Next()
	if err != nil {
		return err
	}
	if strings.ToLower(tok) != want {
		return fmt.Errorf("expected %q, saw %q", want, tok)
	}
	return nil
}

func parseSelect(lex *lexer.Lexer, q *query) error {
	if err := expect(lex, "select"); err != nil {
		return err
	}
	toks, next, err := parseList(lex)
	fmt.Println("####", toks, next, err)
	if err != nil {
		return err
	}
	q.selects = toks
	if strings.ToLower(next) != "from" {
		return fmt.Errorf(`expected "from", saw %q`, next)
	}
	coll, err := lex.Next()
	fmt.Println("####", coll, err)
	if err != nil {
		return err
	}
	q.coll = coll
	for {
		tok, err := lex.Next()
		if err == io.EOF {
			return nil
		}
		switch strings.ToLower(tok) {
		case "where":
			return errors.New("unimp")
		case "order":
			return errors.New("unimp")
		case "limit":
			n, err := parseLimit(lex)
			if err != nil {
				return err
			}
			q.limit = n
		default:
			return fmt.Errorf("unknown clause start: %q", tok)
		}
	}
}

func parseList(lex *lexer.Lexer) ([]string, string, error) {
	var toks []string
	for {
		tok, err := lex.Next()
		if err == io.EOF {
			return toks, "", nil
		}
		if err != nil {
			return nil, "", err
		}
		toks = append(toks, tok)
		tok2, err := lex.Next()
		if err != nil && err != io.EOF {
			return nil, "", err
		}
		if tok2 != "," {
			return toks, tok2, nil
		}
	}
}

func parseLimit(lex *lexer.Lexer) (int, error) {
	tok, err := lex.Next()
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(tok)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func isIdent(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
