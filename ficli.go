// Copyright 2021 Jonathan Amsterdam.

// A simple command-line client for Google Cloud Firestore.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"cloud.google.com/go/firestore"
	"github.com/jba/cli"
)

var flags = struct {
	Project string `cli:"flag=, Google Cloud project ID"`
	Format  string `cli:"flag=, oneof=table|json, output format"`
}{
	Format: "table",
}

var top = cli.Top(&cli.Command{
	Struct: &flags,
	Usage:  "firestore command-line tool",
})

var client *firestore.Client

func main() {
	flag.Parse()
	if flags.Project == "" {
		fmt.Fprintln(os.Stderr, "need -project")
		flag.Usage()
		os.Exit(2)
	}
	ctx := context.Background()
	var err error
	client, err = firestore.NewClient(ctx, flags.Project)
	if err != nil {
		die("creating client: %v", err)
	}
	os.Exit(top.Main(context.Background()))
}

type set struct {
	Path  string   `cli:"path to document"`
	Pairs []string `cli:"name=KEY:VALUE, key-value pairs"`
}

func init() {
	top.Command("set", &set{}, "set document fields")
}

func (c *set) Run(ctx context.Context) error {
	dr := client.Doc(c.Path)
	if dr == nil {
		return fmt.Errorf("invalid path %q", c.Path)
	}
	mapval, err := pairsToMap(c.Pairs)
	if err != nil {
		return err
	}
	_, err = dr.Set(ctx, mapval)
	return err
}

type get struct {
	Paths []string `cli:"name=PATH, min=1, paths to documents"`
}

func init() {
	top.Command("get", &get{}, "get documents")
}

func (c *get) Run(ctx context.Context) error {
	for _, a := range c.Paths {
		dr := client.Doc(a)
		if dr == nil {
			return fmt.Errorf("invalid path %q", a)
		}
		ds, err := dr.Get(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %v\n", a, ds.Data())
	}
	return nil
}

type delete struct {
	Paths []string `cli:"name=PATH, min=1, paths to documents"`
}

func init() {
	top.Command("delete", &delete{}, "delete documents")
}

func (c *delete) Run(ctx context.Context) error {
	for _, a := range c.Paths {
		dr := client.Doc(a)
		if dr == nil {
			return fmt.Errorf("invalid path %q", a)
		}
		if _, err := dr.Delete(ctx); err != nil {
			return fmt.Errorf("%s: %w", a, err)
		}
	}
	return nil
}

type sel struct {
	Args []string `cli:"name=FIELDS from COLLECTION..., min=1, select expression"`
}

func init() {
	top.Command("select", &sel{}, "run a query")
}

func (c *sel) Run(ctx context.Context) error {
	q, err := parseQuery("select " + strings.Join(c.Args, " "))
	if err != nil {
		return err
	}
	docsnaps, err := runQuery(ctx, client, q)
	if err != nil {
		return err
	}

	if len(docsnaps) == 0 {
		fmt.Println("No results.")
		return nil
	}
	displayDocs(os.Stdout, docsnaps, q.selects)
	return nil
}

type docs struct {
	Collection string
}

func init() {
	top.Command("docs", &docs{}, "list the IDs of all documents in a collection")
}

func (c *docs) Run(ctx context.Context) error {
	coll := client.Collection(c.Collection)
	docsnaps, err := coll.DocumentRefs(ctx).GetAll()
	if err != nil {
		return err
	}
	for _, ds := range docsnaps {
		fmt.Println(ds.ID)
	}
	return nil
}

func runQuery(ctx context.Context, c *firestore.Client, q *query) ([]*firestore.DocumentSnapshot, error) {
	coll := c.Collection(q.coll)
	if coll == nil {
		return nil, fmt.Errorf("invalid collection %q", q.coll)
	}
	fq := coll.Query
	if len(q.selects) > 0 {
		fq = fq.Select(q.selects...)
	}
	for _, w := range q.wheres {
		fq = fq.Where(w.path, w.op, convertString(w.value))
	}
	for _, ord := range q.orders {
		fq = fq.OrderBy(ord.path, ord.dir)
	}
	if q.limit > 0 {
		fq = fq.Limit(int(q.limit))
	}
	return fq.Documents(ctx).GetAll()
}

func pairsToMap(args []string) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	for _, a := range args {
		i := strings.IndexRune(a, ':')
		if i < 0 {
			return nil, fmt.Errorf("missing colon: %q", a)
		}
		if i == 0 {
			return nil, fmt.Errorf("empty key: %q", a)
		}
		m[a[:i]] = convertString(a[i+1:])
	}
	return m, nil
}

func mapToPairs(m map[string]interface{}) string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s:%v", k, m[k]))
	}
	return strings.Join(pairs, " ")
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func convertString(s string) interface{} {
	i, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return i
	}
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f
	}
	return s
}

func displayDocs(w io.Writer, docsnaps []*firestore.DocumentSnapshot, cols []string) {
	switch flags.Format {
	case "table":
		if len(cols) == 0 {
			for _, ds := range docsnaps {
				fmt.Fprintf(w, "%s: %s\n", ds.Ref.ID, mapToPairs(ds.Data()))
			}
		} else {
			tw := tabwriter.NewWriter(w, 0, 8, 1, ' ', 0)
			io.WriteString(tw, cols[0])
			for _, c := range cols[1:] {
				fmt.Fprintf(tw, "\t%s", c)
			}
			io.WriteString(tw, "\n")
			for _, ds := range docsnaps {
				d := ds.Data()

				field := func(s string) interface{} {
					if s == "ID" {
						return ds.Ref.ID
					}
					return d[s]
				}

				fmt.Fprint(tw, field(cols[0]))
				for _, c := range cols[1:] {
					fmt.Fprintf(tw, "\t%v", field(c))
				}
				io.WriteString(tw, "\n")
			}
			tw.Flush()
		}
	case "json":
		enc := json.NewEncoder(w)
		for _, ds := range docsnaps {
			if err := enc.Encode(ds.Data()); err != nil {
				fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
			}
		}

	default:
		die("unknown output format %q", flags.Format)
	}
}
