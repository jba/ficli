// Copyright 2021 Jonathan Amsterdam.

/*
ficli is a simple command-line client for Google Cloud Firestore.
It lets you get, set and delete documents from the command line.

You must provide a -project flag, or set the environment variable
FICLI_PROJECT.

Run with -h for documentation on the get, set, delete and docs commands.
The select command is documented below.


Select

ficli has a parser for a simplified SQL-like query language. The sub-command
"select" and the arguments following it are parsed into a Firestore query and
executed.

The smallest query is

  select FIELDS from COLLECTION

where FIELDS is a comma-separated list of fields or "*" (which can also be written
"all" to avoid shell glob expansion).

To the basic query can be appended one or more "where" clauses joined by "and",
an "order by" clause, and a "limit" clause. Each "where" expression is of the
form

  FIELD OP VALUE

where FIELD is a field name, OP is one of ==, >, >=, < or <=, and VALUE can be a
number or string.

Example:
  ficli select StartedAt, NumProcessed from Namespaces/dev/Updates order by StartedAt desc
*/
package main
