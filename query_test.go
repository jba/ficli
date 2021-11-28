// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"testing"

	"cloud.google.com/go/firestore"
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
			query{coll: "c", orders: []order{{"a", firestore.Asc}, {"b", firestore.Asc}}},
		},
		{
			"select * from c order by a, b limit 10",
			query{coll: "c", orders: []order{{"a", firestore.Asc}, {"b", firestore.Asc}}, limit: 10},
		},
		{
			"select * from c order by a desc, b asc limit 10",
			query{coll: "c", orders: []order{{"a", firestore.Desc}, {"b", firestore.Asc}}, limit: 10},
		},
		{
			"select * from c order by a, b desc limit 10",
			query{coll: "c", orders: []order{{"a", firestore.Asc}, {"b", firestore.Desc}}, limit: 10},
		},
		{
			"select * from c where a > 0",
			query{coll: "c", wheres: []where{{"a", ">", "0"}}},
		},
		{
			"select * from c where a > 0 and b = d",
			query{coll: "c", wheres: []where{
				{"a", ">", "0"},
				{"b", "=", "d"},
			}},
		},
		{
			"select * from c where a > 0 and b = d and s3<=2 limit 5",
			query{coll: "c", limit: 5, wheres: []where{
				{"a", ">", "0"},
				{"b", "=", "d"},
				{"s3", "<=", "2"},
			}},
		},
	} {
		got, err := parseQuery(test.in)
		if err != nil {
			t.Fatalf("%s: %v", test.in, err)
		}
		if diff := cmp.Diff(&test.want, got, cmp.AllowUnexported(query{}, order{}, where{})); diff != "" {
			t.Errorf("%s: -want, +got:\n%s", test.in, diff)
		}
	}
}
