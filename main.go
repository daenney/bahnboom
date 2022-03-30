package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
)

func main() {
	showVersion := flag.Bool("version", false, "show version and build info")
	asJSON := flag.Bool("json", false, "output JSON instead")

	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, "{\"version\": \"%s\", \"commit\": \"%s\", \"date\": \"%s\"}\n", version, commit, date)
		os.Exit(0)
	}

	err, cookie, csrf := tokens(context.TODO())
	if err != nil {
		log.Fatalln(err)
	}

	err, issues := issues(context.TODO(), cookie, csrf)
	if err != nil {
		log.Fatalln(err)
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Date.Before(issues[j].Date)
	})

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(issues); err != nil {
			log.Fatalln(err)
		}
		os.Exit(0)
	}

	for _, entry := range issues {
		if entry.Planned {
			fmt.Println(formatMaintenance(&entry))
		} else {
			fmt.Println(formatDisruption(&entry))
		}
	}
}
