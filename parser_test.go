// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	var got query
	p := And(
		Lit("select"),
		Do(
			Or(Lit("*"), List(Is("identifier", Ident), Lit(","))),
			func(toks []string) error {
				if toks[0] != "*" {
					for i := 0; i < len(toks); i += 2 {
						got.selects = append(got.selects, toks[i])
					}
				}
				return nil
			}),
		Lit("from"),
		Do(Is("identifier", Ident), func(toks []string) error { got.coll = toks[0]; return nil }),
		Optional(And(
			Lit("limit"),
			Commit,
			Do(Any, func(toks []string) error {
				n, err := strconv.Atoi(toks[0])
				if err != nil {
					return err
				}
				got.limit = n
				return nil
			}))))
	for _, test := range []struct {
		in   string
		want query
		err  string
	}{
		{
			in:   "select * from cities",
			want: query{selects: nil, coll: "cities"},
		},
		{
			in:   "select a , b , c from d",
			want: query{selects: []string{"a", "b", "c"}, coll: "d"},
		},
		{
			in:   "select * from cities limit 5",
			want: query{selects: nil, coll: "cities", limit: 5},
		},
		{
			in:  "select from x",
			err: `expected "from", got "x"`,
		},
		{
			in:  "select a , from x",
			err: `expected "from", got "x"`,
		},
		{
			in:  "select * from cities and more",
			err: `unconsumed input starting at "and"`,
		},
		{
			in:  "query",
			err: `expected "select", got "query"`,
		},
		{
			in:  "select * from x limit b",
			err: `strconv.Atoi: parsing "b": invalid syntax`,
		},
	} {
		got = query{}
		err := Parse(strings.Fields(test.in), p)
		if err == nil {
			if test.err != "" {
				t.Errorf("%q: got success, want error", test.in)
			} else if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(query{})); diff != "" {
				t.Errorf("%q: mismatch (-want, +got)\n%s", test.in, diff)
			}
		} else {
			if test.err == "" {
				t.Errorf("%q: got %v, want success", test.in, err)
			} else if g := err.Error(); g != test.err {
				t.Errorf("%q, error:\ngot:  %s\nwant: %s", test.in, g, test.err)
			}
		}
	}
}

func TestRepeat(t *testing.T) {
	p := Repeat(Lit("x"))
	var xs []string
	for i := 0; i < 3; i++ {
		if err := Parse(xs, p); err != nil {
			t.Fatal(err)
		}
		xs = append(xs, "x")
	}
}
