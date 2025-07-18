package fetcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/malayanand/newsx/internal/store"
	"github.com/mmcdole/gofeed"
	"github.com/temoto/robotstxt"
)

var RSSFeeds = []string{
	// Indian national dailies
	"https://timesofindia.indiatimes.com/rssfeedstopstories.cms", // Times of India :contentReference[oaicite:0]{index=0}
	"https://www.thehindu.com/feeder/default.rss",                // The Hindu :contentReference[oaicite:2]{index=2}
	"https://indianexpress.com/section/india/feed/",              // Indian Express :contentReference[oaicite:3]{index=3}

	"https://feeds.feedburner.com/ndtvnews-top-stories",
	"https://www.livemint.com/rss/politics",
	"http://www.business-standard.com/rss/home_page_top_stories.rss",
	"view-source:https://theprint.in/category/politics/",
	"https://feeds.feedburner.com/ScrollinArticles.rss",
	"https://www.indiatoday.in/rss/1206514",
	"https://feeds.feedburner.com/ndtvnews-top-stories",

	"https://b2b.economictimes.indiatimes.com/rss/topstories", // Economic Times :contentReference[oaicite:6]{index=6}
	"http://www.dnaindia.com/rss.xml",                         // DNA India :contentReference[oaicite:8]{index=8}
	"http://www.deccanchronicle.com/rss.xml",                  // Deccan Chronicle :contentReference[oaicite:9]{index=9}

	// International with India focus
	"https://feeds.bbci.co.uk/news/world/asia/india/rss.xml", // BBC News Asia/India :contentReference[oaicite:11]{index=11}
	"https://www.reuters.com/subjects/indiaNews/rss.xml",     // Reuters India RSS :contentReference[oaicite:12]{index=12}

	// Other prominent outlets
	"https://economictimes.indiatimes.com/markets/rssfeeds/1977021501.cms", // ET Markets
}

var (
	client = &http.Client{Timeout: 10 * time.Second}
	robots = make(map[string]*robotstxt.RobotsData)
	rLock  sync.Mutex

	// in-memory rate limiter
	lastFetch = make(map[string]time.Time)
	lfLock    sync.Mutex
)

func ScrapeSummary(ctx context.Context, pageURL string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}
	host := u.Scheme + "://" + u.Host

	// robots.txt check
	rLock.Lock()
	rData, ok := robots[host]
	rLock.Unlock()
	if !ok {
		resp, err := client.Get(host + "/robots.txt")
		if err == nil && resp.StatusCode == 200 {
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			rData, _ = robotstxt.FromBytes(data)
		}
		rLock.Lock()
		robots[host] = rData
		rLock.Unlock()
	}
	if rData != nil && !rData.TestAgent(u.Path, "MyNewsMVPBot") {
		return "", errors.New("scrapping dissallowed by robots.txt")
	}

	// rate limiting
	lfLock.Lock()
	since := time.Since(lastFetch[host])
	if since < time.Second {
		time.Sleep(time.Second - since)
	}
	lastFetch[host] = time.Now()
	lfLock.Unlock()

	// fetch article
	req, _ := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	req.Header.Set("User-Agent", "MyNewsMVPBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// use readabillity to extract
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return "", err
	}
	doc, err := readability.FromReader(buf, u)
	if err != nil {
		return "", err
	}
	text := doc.TextContent
	if len(text) > 200 {
		text = text[:200] + "…"
	}
	return text, nil
}

func FetchFromRSS() ([]store.Article, error) {
	parser := gofeed.NewParser()
	var out []store.Article

	for _, url := range RSSFeeds {
		feed, err := parser.ParseURL(url)
		if err != nil {
			fmt.Println("[WARN]: Unable to fetch feed from: ", url)
			continue
		}
		for _, item := range feed.Items {
			published := time.Now()
			if item.PublishedParsed != nil {
				published = *item.PublishedParsed
			}
			out = append(out, store.Article{
				Source:      feed.Title,
				Title:       item.Title,
				URL:         item.Link,
				PublishedAt: published,
				Content:     item.Description,
				Rating:      "",
			})
		}
	}
	return out, nil
}
