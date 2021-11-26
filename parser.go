// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"fmt"
	"strconv"
	"unicode"
)

type parser func(*state)

type state struct {
	toks       []string
	start, pos int
	committed  bool
}

func (s *state) atEOF() bool {
	return s.pos >= len(s.toks)
}

func (s *state) current() string {
	if s.atEOF() {
		return "end of input"
	}
	return strconv.Quote(s.toks[s.pos])
}

func (s *state) Tokens() []string {
	return s.toks[s.start:s.pos]
}

func (s *state) Token() string {
	if s.pos-s.start > 1 {
		panic("more than one token")
	}
	if s.pos == s.start {
		return ""
	}
	return s.toks[s.start]
}

type failure struct {
	err error
}

func (s *state) fail(err error) {
	panic(failure{err})
}

func (s *state) Failf(format string, args ...interface{}) {
	s.fail(fmt.Errorf(format, args...))
}

func Parse(p parser, toks []string) error {
	s := &state{toks: toks, pos: 0}
	if err := parse(p, s); err != nil {
		return err
	}
	if s.pos != len(s.toks) {
		return fmt.Errorf("unconsumed input starting at %s", s.current())
	}
	return nil
}

func parse(p parser, s *state) (err error) {
	defer func() {
		if x := recover(); x != nil {
			if f, ok := x.(failure); ok {
				err = f.err
			} else {
				panic(x)
			}
		}
	}()
	p(s)
	return nil
}

func Lit(lit string) parser {
	return func(s *state) {
		if s.atEOF() || s.toks[s.pos] != lit {
			s.Failf("expected %q, got %s", lit, s.current())
		}
		s.pos++
	}
}

func Is(name string, pred func(s string) bool) parser {
	return func(s *state) {
		if s.atEOF() || !pred(s.toks[s.pos]) {
			s.Failf("expected %s, got %s", name, s.current())
		}
		s.pos++
	}
}

func And(parsers ...parser) parser {
	return func(s *state) {
		for _, p := range parsers {
			if err := parse(p, s); err != nil {
				s.fail(err)
			}
		}
	}
}

func Or(parsers ...parser) parser {
	return func(s *state) {
		start := s.pos
		defer func(c bool) { s.committed = c }(s.committed)
		s.committed = false

		for _, p := range parsers {
			err := parse(p, s)
			if err == nil {
				return
			}
			if s.committed {
				s.fail(err)
			}
			s.pos = start
		}
		s.Failf("parse failed at %q", s.current())
	}
}

var (
	Empty parser = func(*state) {}

	Commit parser = func(s *state) { s.committed = true }

	Any parser = func(s *state) {
		if s.atEOF() {
			s.Failf("unexpected end of unput")
		}
		s.pos++
	}
)

func Optional(p parser) parser {
	return Or(p, Empty)
}

// Zero or more p's.
func Repeat(p parser) parser {
	// We can't write
	//  Or(And(p, Repeat(p)), Empty)
	// as we would like. Go is applicative-order, so the recursive call to Repeat happens
	// immediately and we have infinite recursion. We must delay the recursion.
	return Or(
		And(p, func(s *state) { Repeat(p)(s) }),
		Empty)
}

// non-empty list:
//   item
// or
//   item sep item
// or
//   item sep item sep item
// etc.
func List(item, sep parser) parser {
	return And(item, Repeat(And(sep, item)))
}

func Do(p parser, f func(*state)) parser {
	return func(s *state) {
		start := s.pos
		p(s)
		s.start = start
		f(s)
	}
}

func Ident(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !(r == '_' || unicode.IsLetter(r)) {
			return false
		}
		if i > 0 && !(r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}
