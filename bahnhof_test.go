package main

import (
	"testing"
	"time"
)

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

func TestParseTitle(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		loc = time.UTC
	}

	options := []struct {
		input    string
		date     time.Time
		location string
		operator string
		planned  bool
	}{
		{input: "Driftstörning - 2022-03-29 - Ludvika (IP-Only)", date: time.Date(2022, 03, 29, 0, 0, 0, 0, loc), location: "Ludvika", operator: "IP-Only", planned: false},
		{input: "Driftstörning - 2022-03-30 - Planerat Servicearbete - Bodekullsvägen, Karlshamn (Open Universe)", date: time.Date(2022, 03, 30, 0, 0, 0, 0, loc), location: "Bodekullsvägen, Karlshamn", operator: "Open Universe", planned: true},
		{input: "Driftstörning - 2022-03-31 - Planerat Servicearbete - (Open Universe)", date: time.Date(2022, 03, 31, 0, 0, 0, 0, loc), location: "", operator: "Open Universe", planned: true},
		{input: "Driftstörning - 2022-03-31 - Planerat Servicearbete - Open Universe", date: time.Date(2022, 03, 31, 0, 0, 0, 0, loc), location: "", operator: "Open Universe", planned: true},
	}

	for _, opt := range options {
		opt := opt
		t.Run(opt.input, func(t *testing.T) {
			t.Parallel()
			date, location, operator, planned := parseTitle(opt.input)
			if !opt.date.Equal(date) {
				t.Errorf("expected %+v, got %+v", opt.date, date)
			}
			if opt.location != location {
				t.Errorf("expected %s, got %s", opt.location, location)
			}
			if opt.operator != operator {
				t.Errorf("expected %s, got %s", opt.operator, operator)
			}
			if opt.planned != planned {
				t.Errorf("expected %t, got %t", opt.planned, planned)
			}
		})
	}
}
