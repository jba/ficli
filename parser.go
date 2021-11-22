// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"errors"
	"fmt"
	"unicode"
)

type parser func(*state) error

type state struct {
	toks      []string
	pos       int
	committed bool
}

func (s *state) atEOF() bool {
	return s.pos >= len(s.toks)
}

func (s *state) current() string {
	if s.atEOF() {
		return "end of input"
	}
	return fmt.Sprintf("%q", s.toks[s.pos])
}

func Parse(toks []string, p parser) error {
	s := &state{toks: toks, pos: 0}
	if err := p(s); err != nil {
		return err
	}
	if s.pos != len(s.toks) {
		return fmt.Errorf("unconsumed input starting at %s", s.current())
	}
	return nil
}

func And(parsers ...parser) parser {
	return func(s *state) error {
		for _, p := range parsers {
			if err := p(s); err != nil {
				return err
			}
		}
		return nil
	}
}

func Lit(lit string) parser {
	return func(s *state) error {
		if s.atEOF() || s.toks[s.pos] != lit {
			return fmt.Errorf("expected %q, got %s", lit, s.current())
		}
		s.pos++
		return nil
	}
}

func Is(name string, pred func(s string) bool) parser {
	return func(s *state) error {
		if s.atEOF() || !pred(s.toks[s.pos]) {
			return fmt.Errorf("expected %s, got %s", name, s.current())
		}
		s.pos++
		return nil
	}
}

func Or(parsers ...parser) parser {
	return func(s *state) error {
		start := s.pos
		c := s.committed
		s.committed = false
		for _, p := range parsers {
			err := p(s)
			if err == nil || s.committed {
				s.committed = c
				return err
			}
			s.pos = start
		}
		return fmt.Errorf("parse failed at %q", s.current())
	}
}

var (
	Empty parser = func(*state) error {
		return nil
	}

	Any parser = func(s *state) error {
		if s.atEOF() {
			return errors.New("unexpected end of unput")
		}
		s.pos++
		return nil
	}

	Commit parser = func(s *state) error {
		s.committed = true
		return nil
	}
)

func Optional(p parser) parser {
	return Or(p, Empty)
}

func opt(p parser, s *state) bool {
	start := s.pos
	if p(s) != nil {
		s.pos = start
		return false
	}
	return true
}

func List(item, sep parser) parser {
	return func(s *state) error {
		for {
			if err := item(s); err != nil {
				return err
			}
			if !opt(sep, s) {
				return nil
			}
		}
	}
}

func Do(p parser, f func([]string) error) parser {
	return func(s *state) error {
		start := s.pos
		if err := p(s); err != nil {
			return err
		}
		return f(s.toks[start:s.pos])
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
