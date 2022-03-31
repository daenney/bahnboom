package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"

	"golang.org/x/net/html"
)

const base = "https://bahnhof.se/kundservice/driftinfo"
const api = "https://bahnhof.se/ajax/kundservice/driftinfo"
const userAgent = "bahnboom (+https://github.com/daenney/bahnboom)"
const timeFormat = "15:04"
const dateFormat = "2006-01-02"
const dateTimeFormat = "2006-01-02 15:04"

var (
	titleMatcher = regexp.MustCompile(`(?i)^\p{L}+ - (?P<date>\d{4}-\d{2}-\d{2}) - (?P<planned>planerat servicearbete - )?(?P<rest>.*)$`)
	startMatcher = regexp.MustCompile(`Start:\s+(?P<time>\d{4}-\d{2}-\d{2} \d{2}:\d{2})`)
	stopMatcher  = regexp.MustCompile(`Stop:\s+(?P<time>\d{4}-\d{2}-\d{2} \d{2}:\d{2})`)
)

type transport struct{}

func (*transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", userAgent)
	return http.DefaultTransport.RoundTrip(req)
}

var client = &http.Client{Transport: &transport{}}

type response struct {
	Status string `json:"status"`
	Data   data   `json:"data"`
}

type data struct {
	Open []entry `json:"open,omitempty"`
}

type entry struct {
	Location string     `json:"location"`
	Operator string     `json:"operator"`
	Planned  bool       `json:"planned"`
	Date     time.Time  `json:"date"`
	Start    *time.Time `json:"start,omitempty"`
	Stop     *time.Time `json:"stop,omitempty"`
}

type message struct {
	Message string `json:"message"`
}

func (e *entry) UnmarshalJSON(b []byte) error {
	var objmap map[string]*json.RawMessage
	err := json.Unmarshal(b, &objmap)
	if err != nil {
		return err
	}

	var title string
	err = json.Unmarshal(*objmap["title"], &title)
	if err != nil {
		return err
	}

	var msgs []message
	err = json.Unmarshal(*objmap["messages"], &msgs)
	if err != nil {
		return err
	}

	date, location, operator, planned := parseTitle(title)
	e.Location = location
	e.Operator = operator
	e.Date = date
	e.Planned = planned
	start, stop := parseStartStop(msgs)
	if !start.Equal(time.Time{}) {
		e.Start = &start
	}
	if !stop.Equal(time.Time{}) {
		e.Stop = &stop
	}
	return nil
}

func formatMaintenance(e *entry) string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("• 👷 %s: Scheduled maintenance on %s", e.Date.Format(dateFormat), e.Operator))
	if !e.Start.Equal(time.Time{}) {
		format := dateTimeFormat
		if sameDay(e.Date, *e.Start) {
			format = timeFormat
		}
		b.WriteString(fmt.Sprintf(" starting at: %s", e.Start.Format(format)))
	}
	if !e.Stop.Equal(time.Time{}) {
		format := dateTimeFormat
		if sameDay(e.Date, *e.Stop) {
			format = timeFormat
		}
		b.WriteString(fmt.Sprintf(" lasting until: %s", e.Stop.Format(format)))
	}
	if e.Location != "" {
		b.WriteString(fmt.Sprintf(" in %s", e.Location))
	}
	return b.String()
}

func sameDay(d1, d2 time.Time) bool {
	if d1.Year() == d2.Year() && d1.Month() == d2.Month() && d1.Day() == d2.Day() {
		return true
	}
	return false
}

func formatDisruption(e *entry) string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("• 🔥 %s: Ongoing service disruption on %s", e.Date.Format(dateFormat), e.Operator))
	if e.Location != "" {
		b.WriteString(fmt.Sprintf(" in %s", e.Location))
	}
	return b.String()
}

func parseTitle(title string) (date time.Time, location, operator string, planned bool) {
	match := titleMatcher.FindStringSubmatch(title)
	params := map[string]string{}
	for i, name := range titleMatcher.SubexpNames() {
		if i > 0 && i <= len(match) {
			params[name] = match[i]
		}
	}

	if params["planned"] != "" {
		planned = true
	}

	location, operator = extractLocationAndOperator(params["rest"])

	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		loc = time.UTC
	}

	date, err = time.ParseInLocation(dateFormat, params["date"], loc)
	if err != nil {
		date = time.Time{}
	}

	return date, location, operator, planned
}

func extractLocationAndOperator(s string) (location, operator string) {
	if strings.HasPrefix(s, "(") {
		return "", strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "("), ")"))
	}
	elems := strings.Split(s, "(")
	switch len(elems) {
	case 1:
		return "", strings.TrimSpace(elems[0])
	case 2:
		return strings.TrimSpace(elems[0]), strings.TrimSpace(strings.TrimSuffix(elems[1], ")"))
	default:
		return "", ""
	}
}

func parseStartStop(msgs []message) (start, stop time.Time) {
	for _, msg := range msgs {
		matchStart := startMatcher.FindStringSubmatch(msg.Message)
		matchStop := stopMatcher.FindStringSubmatch(msg.Message)
		if len(matchStart) != 2 || len(matchStop) != 2 {
			continue
		}

		loc, err := time.LoadLocation("Europe/Stockholm")
		if err != nil {
			loc = time.UTC
		}

		start, err = time.ParseInLocation(dateTimeFormat, matchStart[1], loc)
		if err != nil {
			continue
		}
		stop, err = time.ParseInLocation(dateTimeFormat, matchStop[1], loc)
		if err != nil {
			continue
		}
		break
	}
	return start, stop
}

func tokens(ctx context.Context) (e error, cookie *http.Cookie, csrf string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base, nil)
	if err != nil {
		return err, cookie, csrf
	}

	resp, err := client.Do(req)
	if err != nil {
		return err, cookie, csrf
	}

	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return fmt.Errorf("request to Bahnhof failed"), cookie, csrf
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body"), cookie, csrf
	}

	for _, item := range resp.Cookies() {
		if item.Name == "PHPSESSID" {
			cookie = item
		}
	}

	if cookie == nil {
		return fmt.Errorf("failed to retrieve cookie"), cookie, csrf
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to parse body as HTML"), cookie, csrf
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			for _, attr := range n.Attr {
				if attr.Key == "name" && attr.Val == "csrf-token" {
					for _, attr := range n.Attr {
						if attr.Key == "content" {
							csrf = attr.Val
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if csrf == "" {
		return fmt.Errorf("failed to extract CSRF token"), cookie, csrf
	}

	return nil, cookie, csrf
}

func issues(ctx context.Context, cookie *http.Cookie, csrf string) (error, []entry) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return err, nil
	}

	req.AddCookie(cookie)
	req.Header.Set("X-CSRF-TOKEN", csrf)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return err, nil
	}

	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return fmt.Errorf("request to Bahnhof failed"), nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body"), nil
	}

	issues := &response{}
	err = json.Unmarshal(body, issues)
	if err != nil {
		return fmt.Errorf("failed to decode body: %w", err), nil
	}
	if issues.Status != "ok" {
		return fmt.Errorf("API returned an error"), nil
	}

	return nil, issues.Data.Open
}
