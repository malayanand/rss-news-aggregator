package fetcher

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/temoto/robotstxt"
)

func TestScrapeSummary_HttpBinHTML(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := "https://httpbin.org/html"
	scrape, err := ScrapeSummary(ctx, url)
	if err != nil {
		t.Fatalf("scrapeSummary(%q) returned err: %v", url, err)
	}
	if len(scrape) == 0 {
		t.Fatalf("expected non-empty snippet but got empty string")
	}
	if !strings.Contains(scrape, "Herman Melville") {
		t.Errorf("snippet did not contain expected string, got %q", scrape)
	}
}

func TestScrapeSummary_InvalidURL(t *testing.T) {
	_, err := ScrapeSummary(context.Background(), "://not-a-valid-url")
	if err == nil {
		t.Fatalf("expected an invalid URL, got nil")
	}
}

func TestScrapeSummary_DisallowedByRobots(t *testing.T) {
	// example.com robots.txt disallows/deny any path
	url := "https://example.com/deny"
	// create dummy robots.txt entry to disallow the bot
	const raw = `
		User-agent: MyNewsMVPBot
		Disallow: /deny
		`
	robotsData, err := robotstxt.FromString(raw)
	if err != nil {
		t.Fatalf("parsing dummy robots.txt: %v", err)
	}
	robots = make(map[string]*robotstxt.RobotsData)
	robots["https://example.com"] = robotsData

	_, err = ScrapeSummary(context.Background(), url)
	if err == nil {
		t.Fatal("expected dissallowed by robots, got nil")
	}
}
