package classifier

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/malayanand/newsx/internal/store"
)

func StartArticleClassifier(ctx context.Context, db *store.Config, clf *Client) chan<- store.Article {
	ch := make(chan store.Article, 100)
	var processed int64

	go func() {
		defer func() {
			log.Printf("[classifier] processed %d articles\n", atomic.LoadInt64(&processed))
		}()

		for {
			select {
			case art, ok := <-ch:
				if !ok {
					return
				}
				label, err := clf.Classify(ctx, art.ScrappedSummary)
				if err != nil {
					log.Println("classification error: ", art.URL, err)
				} else {
					art.Rating = label
				}

				if err := db.UpsertArticle(ctx, art); err != nil {
					log.Println("upsert after classification failed", art.URL, err)
				} else {
					atomic.AddInt64(&processed, 1)
				}

			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}
