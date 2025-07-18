package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/malayanand/newsx/internal/classifier"
	"github.com/malayanand/newsx/internal/fetcher"
	"github.com/malayanand/newsx/internal/store"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
)

func Start(ctx context.Context, dbConfig *store.Config, clf *classifier.Client) {
	c := cron.New(cron.WithLocation(time.UTC))

	_, err := c.AddFunc("@every 1h", func() {
		// 30-minute max per run
		// for testing set to 2 min
		runCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		log.Println("[scheduler] fetching top headlines for India...")
		articles, err := fetcher.FetchFromRSS()
		if err != nil {
			log.Println("Fetch error: ", err)
			return
		}

		// start ingest and classifier gorouting upsert and classify articles into db
		upsertCh := store.StartArticleIngestor(runCtx, dbConfig)
		classifierCh := classifier.StartArticleClassifier(runCtx, dbConfig, clf)

		//TODO: Change the from errgroups to waitgroups since use of errgroups here serves little purpose
		sem := make(chan struct{}, 10)
		g, gctx := errgroup.WithContext(runCtx)

		for i := range articles {
			if err := gctx.Err(); err != nil {
				log.Printf("scheduler: context done (%v), aborting remaining scrapes\n", err)
				break
			}

			article := &articles[i]
			cached, _ := dbConfig.CheckCache(gctx, article.URL)
			if cached || len(article.Content) >= 150 {
				continue
			}

			sem <- struct{}{}
			g.Go(func() error {
				defer func() { <-sem }()
				subctx, cancel := context.WithTimeout(gctx, 30*time.Second)
				defer cancel()

				summary, err := fetcher.ScrapeSummary(subctx, article.URL)
				if err != nil {
					log.Println("scrape error:", article.URL, err)
					return nil
				}
				article.ScrappedSummary = summary

				if gctx.Err() == nil {
					upsertCh <- *article
					classifierCh <- *article
				}
				return nil
			})
		}

		_ = g.Wait()
		close(upsertCh)
		close(classifierCh)
		log.Println("[scheduler] scrape+upsert completed")
	})

	if err != nil {
		log.Fatalf("Failed to schedule fetch job: %v", err)
	}

	c.Start()
	<-ctx.Done()
	log.Println("[scheduler] shutting down")
	c.Stop()
}
