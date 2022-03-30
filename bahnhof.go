package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"

	"golang.org/x/net/html"
)

const base = "https://bahnhof.se/kundservice/driftinfo"
const api = "https://bahnhof.se/ajax/kundservice/driftinfo"
const userAgent = "bahnboom (+https://github.com/daenney/bahnboom)"

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
	Location string    `json:"location"`
	Operator string    `json:"operator"`
	Planned  bool      `json:"planned"`
	Date     time.Time `json:"date"`
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

	date, location, operator, planned := parseTitle(title)
	e.Location = location
	e.Operator = operator
	e.Date = date
	e.Planned = planned
	return nil
}

func formatMaintenance(e *entry) string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("• 👷 %s: Scheduled maintenance on %s", e.Date.Format("2006-01-02"), e.Operator))
	if e.Location != "" {
		b.WriteString(fmt.Sprintf(" in %s", e.Location))
	}
	return b.String()
}

func formatDisruption(e *entry) string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("• 🔥 %s: Ongoing service disruption on %s", e.Date.Format("2006-01-02"), e.Operator))
	if e.Location != "" {
		b.WriteString(fmt.Sprintf(" in %s", e.Location))
	}
	return b.String()
}

func parseTitle(title string) (date time.Time, location, operator string, planned bool) {
	elems := strings.Split(title, "-")
	switch len(elems) {
	case 6:
		location, operator = extractLocationAndOperator(strings.TrimSpace(elems[5]))
		if strings.ToLower(strings.TrimSpace(elems[4])) == "planerat servicearbete" {
			planned = true
		}
	case 5:
		location, operator = extractLocationAndOperator(strings.TrimSpace(elems[4]))
	default:
		return time.Unix(0, 0), "", "", false
	}

	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		loc = time.UTC
	}

	sdate := strings.Join([]string{strings.TrimSpace(elems[1]), strings.TrimSpace(elems[2]), strings.TrimSpace(elems[3])}, "-")
	date, err = time.ParseInLocation("2006-01-02", sdate, loc)
	if err != nil {
		date = time.Unix(0, 0)
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
