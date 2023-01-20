// Package chaos logic
package chaos

import (
	"context"
	"fmt"

	"github.com/hlnths/subfinder/v2/pkg/subscraping"
	"github.com/projectdiscovery/chaos-client/pkg/chaos"
)

// Source is the passive scraping agent
type Source struct {
	apiKeys []string
}

// Run function returns all subdomains found with the service
func (s *Source) Run(_ context.Context, domain string, _ *subscraping.Session) <-chan subscraping.Result {
	results := make(chan subscraping.Result)

	go func() {
		defer close(results)

		randomApiKey := subscraping.PickRandom(s.apiKeys, s.Name())
		if randomApiKey == "" {
			return
		}

		chaosClient := chaos.New(randomApiKey)
		for result := range chaosClient.GetSubdomains(&chaos.SubdomainsRequest{
			Domain: domain,
		}) {
			if result.Error != nil {
				results <- subscraping.Result{Source: s.Name(), Type: subscraping.Error, Error: result.Error}
				break
			}
			results <- subscraping.Result{Source: s.Name(), Type: subscraping.Subdomain, Value: fmt.Sprintf("%s.%s", result.Subdomain, domain)}
		}
	}()

	return results
}

// Name returns the name of the source
func (s *Source) Name() string {
	return "chaos"
}

func (s *Source) IsDefault() bool {
	return true
}

func (s *Source) HasRecursiveSupport() bool {
	return false
}

func (s *Source) NeedsKey() bool {
	return true
}

func (s *Source) AddApiKeys(keys []string) {
	s.apiKeys = keys
}
