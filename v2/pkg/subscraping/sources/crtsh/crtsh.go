// Package crtsh logic
package crtsh

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"

	// postgres driver
	_ "github.com/lib/pq"

	"github.com/hlnths/subfinder/v2/pkg/subscraping"
)

type subdomain struct {
	ID        int    `json:"id"`
	NameValue string `json:"name_value"`
}

// Source is the passive scraping agent
type Source struct{}

// Run function returns all subdomains found with the service
func (s *Source) Run(ctx context.Context, domain string, session *subscraping.Session) <-chan subscraping.Result {
	results := make(chan subscraping.Result)

	go func() {
		defer close(results)

		count := s.getSubdomainsFromSQL(domain, session, results)
		if count > 0 {
			return
		}
		_ = s.getSubdomainsFromHTTP(ctx, domain, session, results)
	}()

	return results
}

func (s *Source) getSubdomainsFromSQL(domain string, session *subscraping.Session, results chan subscraping.Result) int {
	db, err := sql.Open("postgres", "host=crt.sh user=guest dbname=certwatch sslmode=disable binary_parameters=yes")
	if err != nil {
		results <- subscraping.Result{Source: s.Name(), Type: subscraping.Error, Error: err}
		return 0
	}

	defer db.Close()

	pattern := "%." + domain
	query := `SELECT DISTINCT ci.NAME_VALUE as domain FROM certificate_identity ci
					  WHERE reverse(lower(ci.NAME_VALUE)) LIKE reverse(lower($1))
					  ORDER BY ci.NAME_VALUE`
	rows, err := db.Query(query, pattern)
	if err != nil {
		results <- subscraping.Result{Source: s.Name(), Type: subscraping.Error, Error: err}
		return 0
	}
	if err := rows.Err(); err != nil {
		results <- subscraping.Result{Source: s.Name(), Type: subscraping.Error, Error: err}
		return 0
	}

	var count int
	var data string
	// Parse all the rows getting subdomains
	for rows.Next() {
		err := rows.Scan(&data)
		if err != nil {
			results <- subscraping.Result{Source: s.Name(), Type: subscraping.Error, Error: err}
			return count
		}
		count++
		results <- subscraping.Result{Source: s.Name(), Type: subscraping.Subdomain, Value: session.Extractor.FindString(data)}
	}
	return count
}

func (s *Source) getSubdomainsFromHTTP(ctx context.Context, domain string, session *subscraping.Session, results chan subscraping.Result) bool {
	resp, err := session.SimpleGet(ctx, fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json", domain))
	if err != nil {
		results <- subscraping.Result{Source: s.Name(), Type: subscraping.Error, Error: err}
		session.DiscardHTTPResponse(resp)
		return false
	}

	var subdomains []subdomain
	err = jsoniter.NewDecoder(resp.Body).Decode(&subdomains)
	if err != nil {
		results <- subscraping.Result{Source: s.Name(), Type: subscraping.Error, Error: err}
		resp.Body.Close()
		return false
	}

	resp.Body.Close()

	for _, subdomain := range subdomains {
		for _, sub := range strings.Split(subdomain.NameValue, "\n") {
			results <- subscraping.Result{Source: s.Name(), Type: subscraping.Subdomain, Value: session.Extractor.FindString(sub)}
		}
	}

	return true
}

// Name returns the name of the source
func (s *Source) Name() string {
	return "crtsh"
}

func (s *Source) IsDefault() bool {
	return true
}

func (s *Source) HasRecursiveSupport() bool {
	return true
}

func (s *Source) NeedsKey() bool {
	return false
}

func (s *Source) AddApiKeys(_ []string) {
	// no key needed
}
