package store

import (
	"context"
	"log"
	"sync/atomic"
)

func StartArticleIngestor(ctx context.Context, db *Config) chan<- Article {
	ch := make(chan Article, 100)

	go func() {
		var upsertCnt int64
		defer func() {
			log.Printf(`[ingestor] upserted %d articles\n`, upsertCnt)
		}()

		for {
			select {
			case article, ok := <-ch:
				if !ok {
					return
				}
				if ctx.Err() != nil {
					log.Println("Context cancelled before upserting: ", article.URL)
					continue
				}
				if err := db.UpsertArticle(ctx, article); err != nil {
					log.Println("Upsert error: ", article.URL, err)
				} else {
					atomic.AddInt64(&upsertCnt, 1)
				}
			case <-ctx.Done():
				log.Println("Context canelled: stopping upsert goroutine")
				return
			}
		}
	}()

	return ch
}
