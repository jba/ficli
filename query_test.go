// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseQuery(t *testing.T) {
	got, err := parseQuery("select * from cities")
	if err != nil {
		t.Fatal(err)
	}
	want := &query{
		coll:    "cities",
		selects: []string{"*"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("-want, +got:\n%s", diff)
	}
}
