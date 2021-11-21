// Copyright 2021 Jonathan Amsterdam.

package main

type query struct {
	selects []string
	orders []string
	limit int
}

// Query syntax:
//     select LIST from ID
//     [where EXPR [(and|or) EXPR]...]
//     order by ID [(asc|desc)]
//     limit N



func parseQuery(s string) (*query, error) {
	lex := lexer.New(

func init() {
