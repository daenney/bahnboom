package main

import (
	"sync"
	"time"
	_ "time/tzdata"
)

var location = time.UTC
var once sync.Once

func getLocation() *time.Location {
	once.Do(func() {
		loc, err := time.LoadLocation("Europe/Stockholm")
		if err == nil {
			location = loc
		}
	})

	return location
}
