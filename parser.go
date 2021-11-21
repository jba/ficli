// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"errors"
	"fmt"
	"unicode"
)

func parse() {
	Seq(
		Lit("select"),
		Action(
			Or(Lit("*"), List(Ident, Lit(","))),
			func(toks []string) { q.selects = toks }),
		Lit("from"),
		Action(Ident, func(toks []string) { q.coll = toks[0] }),
		Optional(Seq(Lit("where"), Action(Expr, func(toks []string) {
			q.wheres = append(q.wheres, toks)
		}))),
	)
}

type state struct {
	toks    []string
	pos     int
	failure string
}

func (s *state) atEOF() bool {
	return s.pos >= len(s.toks)
}

func (s *state) current() string {
	if s.atEOF() {
		return "end of input"
	}
	return s.toks[s.pos]
}

func Parse(toks []string, p parser) error {
	s := &state{toks: toks, pos: 0}
	if !p(s) {
		if s.failure != "" {
			return errors.New(s.failure)
		}
		return fmt.Errorf("parse failed at %q", s.current())
	}
	if s.pos != len(s.toks) {
		return fmt.Errorf("unconsumed input starting at %q", s.current())
	}
	return nil
}

type parser func(*state) bool

func And(parsers ...parser) parser {
	return func(s *state) bool {
		for _, p := range parsers {
			if !p(s) {
				return false
			}
		}
		return true
	}
}

func Lit(lit string) parser {
	return func(s *state) bool {
		if s.atEOF() || s.toks[s.pos] != lit {
			s.failure = fmt.Sprintf("expected %q, got %q", lit, s.current())
			return false
		}
		s.pos++
		return true
	}
}

func Is(name string, pred func(s string) bool) parser {
	return func(s *state) bool {
		if s.atEOF() || !pred(s.toks[s.pos]) {
			s.failure = fmt.Sprintf("expected %s, got %q", name, s.current())
			return false
		}
		s.pos++
		return true
	}
}

func Or(parsers ...parser) parser {
	return func(s *state) bool {
		start := s.pos
		for _, p := range parsers {
			if p(s) {
				return true
			}
			s.pos = start
		}
		s.failure = fmt.Sprintf("parse failed at %q", s.current())
		return false
	}
}

func Optional(p parser) parser {
	return func(s *state) bool {
		start := s.pos
		if p(s) {
			return true
		}
		s.pos = start
		return true
	}
}

func Action(p parser, f func([]string)) {
	return func(s *state) bool {
		start := s.pos
		if p(s) {
			f(s.toks[start:s.pos])
		}
		return false
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
