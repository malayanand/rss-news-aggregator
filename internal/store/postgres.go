package store

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	DB *sql.DB
}

type Article struct {
	Source          string
	Title           string
	URL             string
	PublishedAt     time.Time
	Content         string
	ScrappedSummary string
	Rating          string
}

func NewDbConnection(dsn string) (*Config, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &Config{DB: db}, nil
}

func (c *Config) CheckCache(ctx context.Context, url string) (bool, error) {
	var exists bool
	err := c.DB.QueryRowContext(ctx, `SELECT scrapped_summary IS NOT NULL FROM articles WHERE url = $1`, url).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists, err
}

func (c *Config) UpsertArticle(ctx context.Context, article Article) error {
	var summaryArg interface{}
	if article.ScrappedSummary == "" {
		summaryArg = nil
	} else {
		summaryArg = article.ScrappedSummary
	}

	_, err := c.DB.ExecContext(ctx, `
       INSERT INTO articles(source, title, url, published_at, content, scrapped_summary, rating)
       VALUES($1, $2, $3, $4, $5, $6, $7)
       ON CONFLICT (url) DO UPDATE
         SET content          = EXCLUDED.content,
            scrapped_summary = EXCLUDED.scrapped_summary,
            rating          = EXCLUDED.rating
   `, article.Source, article.Title, article.URL, article.PublishedAt, article.Content, summaryArg, article.Rating)
	return err
}
