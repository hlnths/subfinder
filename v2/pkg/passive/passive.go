package passive

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hlnths/subfinder/v2/pkg/subscraping"
	"github.com/projectdiscovery/gologger"
)

// EnumerateSubdomains enumerates all the subdomains for a given domain
func (a *Agent) EnumerateSubdomains(domain string, proxy string, rateLimit, timeout int, maxEnumTime time.Duration) chan subscraping.Result {
	results := make(chan subscraping.Result)
	go func() {
		defer close(results)

		session, err := subscraping.NewSession(domain, proxy, rateLimit, timeout)
		if err != nil {
			results <- subscraping.Result{Type: subscraping.Error, Error: fmt.Errorf("could not init passive session for %s: %s", domain, err)}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), maxEnumTime)

		timeTaken := make(map[string]string)
		timeTakenMutex := &sync.Mutex{}

		wg := &sync.WaitGroup{}
		// Run each source in parallel on the target domain
		for _, runner := range a.sources {
			wg.Add(1)

			now := time.Now()
			go func(source subscraping.Source) {
				for resp := range source.Run(ctx, domain, session) {
					results <- resp
				}

				duration := time.Since(now)
				timeTakenMutex.Lock()
				timeTaken[source.Name()] = fmt.Sprintf("Source took %s for enumeration\n", duration)
				timeTakenMutex.Unlock()

				wg.Done()
			}(runner)
		}
		wg.Wait()

		for source, data := range timeTaken {
			gologger.Verbose().Label(source).Msg(data)
		}

		cancel()
	}()
	return results
}
