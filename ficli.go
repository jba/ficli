// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
)

var (
	project = flag.String("project", "", "Google Cloud project ID")
)

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
	switch flag.Arg(0) {
	case "select":
		err = doSelect(ctx, client, flag.Args())
	default:
		die("unknown command %q", flag.Arg(0))
	}
	if err != nil {
		die("%v", err)
	}
}

func doSelect(ctx context.Context, c *firestore.Client, args []string) error {
	q := strings.Join(args, " ")
	fmt.Println(q)
	return nil
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}
