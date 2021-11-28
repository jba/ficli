// Copyright 2021 Jonathan Amsterdam.

// A simple command-line client for Google Cloud Firestore.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"cloud.google.com/go/firestore"
)

var (
	project = flag.String("project", "", "Google Cloud project ID")
	format  = flag.String("format", "table", "output format (table, json)")
)

var commands = map[string]func(context.Context, *firestore.Client, []string) error{
	"set":    doSet,
	"get":    doGet,
	"delete": doDelete,
	"select": doSelect,
}

func main() {
	flag.Parse()
	if *project == "" {
		die("need -project")
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, *project)
	if err != nil {
		die("creating client: %v", err)
	}
	if err := runCommand(ctx, client, flag.Args()); err != nil {
		die("%v", err)
	}
}

func runCommand(ctx context.Context, c *firestore.Client, args []string) error {
	cmd := commands[args[0]]
	if cmd == nil {
		return fmt.Errorf("unknown command %q", args[0])
	}
	return cmd(ctx, c, args[1:])
}

func doSet(ctx context.Context, c *firestore.Client, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: set path key1:value1 key2:value2 ...")
	}
	dr := c.Doc(args[0])
	if dr == nil {
		return fmt.Errorf("invalid path %q", args[0])
	}
	mapval, err := pairsToMap(args[1:])
	if err != nil {
		return err
	}
	_, err = dr.Set(ctx, mapval)
	return err
}

func doGet(ctx context.Context, c *firestore.Client, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: get path1 [path2 ...]")
	}
	for _, a := range args {
		dr := c.Doc(a)
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

func doDelete(ctx context.Context, c *firestore.Client, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: delete path1 [path2 ...]")
	}
	for _, a := range args {
		dr := c.Doc(a)
		if dr == nil {
			return fmt.Errorf("invalid path %q", a)
		}
		if _, err := dr.Delete(ctx); err != nil {
			return fmt.Errorf("%s: %w", a, err)
		}
	}
	return nil
}

func doSelect(ctx context.Context, c *firestore.Client, args []string) error {
	q, err := parseQuery("select " + strings.Join(args, " "))
	if err != nil {
		return err
	}
	docsnaps, err := runQuery(ctx, c, q)
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
	switch *format {
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
				fmt.Fprint(tw, d[cols[0]])
				for _, c := range cols[1:] {
					fmt.Fprintf(tw, "\t%v", d[c])
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
		die("unknown output format %q", *format)
	}
}
