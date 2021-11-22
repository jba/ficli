// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseQuery(t *testing.T) {
	for _, test := range []struct {
		in   string
		want query
	}{
		{"select * from cities", query{coll: "cities"}},
		{"select a, b from c", query{coll: "c", selects: []string{"a", "b"}}},
		{
			"select  * from c limit 3",
			query{coll: "c", limit: 3},
		},
		{
			"select * from c order by a, b",
			query{coll: "c", orders: []string{"a", "b"}},
		},
		{
			"select * from c order by a, b limit 10",
			query{coll: "c", orders: []string{"a", "b"}, limit: 10},
		},
		{
			"select * from c order by a, b asc limit 10",
			query{coll: "c", orders: []string{"a", "b"}, limit: 10},
		},
		{
			"select * from c order by a, b desc limit 10",
			query{coll: "c", orders: []string{"a", "b"}, desc: true, limit: 10},
		},
	} {
		got, err := parseQuery(test.in)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(&test.want, got, cmp.AllowUnexported(query{})); diff != "" {
			t.Errorf("%s: -want, +got:\n%s", test.in, diff)
		}
	}
}
