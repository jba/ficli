// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"context"
	"strings"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/google/go-cmp/cmp"
)

func TestFirestore(t *testing.T) {
	type data = map[string]interface{}

	if *project == "" {
		t.Skip("no -project")
	}
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, *project)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	run := func(s string) {
		t.Helper()
		if err := runCommand(ctx, client, strings.Fields(s)); err != nil {
			t.Fatal(err)
		}
	}

	run("set cities/detroit nick:motor-city  pop:3400")
	detroit := data{
		"nick": "motor-city",
		"pop":  int64(3400),
	}

	run("set cities/miami   nick:the-hotness pop:1234")
	miami := data{
		"nick": "the-hotness",
		"pop":  int64(1234),
	}

	ds, err := client.Doc("cities/detroit").Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := ds.Data()
	want := detroit
	if !cmp.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	for _, test := range []struct {
		q    string
		want []data
	}{
		{"select * from cities", []data{detroit, miami}},
		{"select pop from cities", []data{{"pop": int64(3400)}, {"pop": int64(1234)}}},
		{"select * from cities limit 1", []data{detroit}},
		{"select * from cities order by pop", []data{miami, detroit}},
		{"select * from cities order by pop asc", []data{miami, detroit}},
		{"select * from cities order by pop desc", []data{detroit, miami}},
		{"select pop, nick from cities where pop > 3000", []data{detroit}},
		{"select pop, nick from cities where pop > 1 and pop < 3000", []data{miami}},
	} {
		q, err := parseQuery(test.q)
		if err != nil {
			t.Fatal(err)
		}
		dss, err := runQuery(ctx, client, q)
		if err != nil {
			t.Fatalf("%q: %v", test.q, err)
		}
		var got []data
		for _, ds := range dss {
			got = append(got, ds.Data())
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("%q: (-want, +got):\n%s", test.q, diff)
		}
	}
}

func TestArgsToMap(t *testing.T) {
	for _, test := range []struct {
		in   string
		want map[string]interface{} // nil means error
	}{
		{":", nil},
		{":x", nil},
		{"abc", nil},
		{
			"a:1 b:2 c: d:11a",
			map[string]interface{}{"a": int64(1), "b": int64(2), "c": "", "d": "11a"},
		},
	} {
		got, err := pairsToMap(strings.Fields(test.in))
		if err != nil && test.want != nil {
			t.Errorf("%q: got %v, wanted no error", test.in, err)
		} else if err == nil {
			if test.want == nil {
				t.Errorf("%q: got no error, wanted one", test.in)
			} else if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("%q: -want, +got:\n%s", test.in, diff)
			}
		}
	}
}
