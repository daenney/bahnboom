package main

import "testing"

func TestExtractLocationAndOperator(t *testing.T) {
	options := []struct {
		input, location, operator string
	}{
		{input: "Gärds Köpinge (iTUX)", location: "Gärds Köpinge", operator: "iTUX"},
		{input: "Kurbit Stadsnät", location: "", operator: "Kurbit Stadsnät"},
		{input: "(IP-Only)", location: "", operator: "IP-Only"},
	}

	for _, opt := range options {
		opt := opt
		t.Run(opt.input, func(t *testing.T) {
			t.Parallel()
			loc, op := extractLocationAndOperator(opt.input)
			if opt.location != loc {
				t.Errorf("expected %s, got %s", opt.location, loc)
			}
			if opt.operator != op {
				t.Errorf("expected %s, got %s", opt.operator, op)
			}
		})
	}
}
