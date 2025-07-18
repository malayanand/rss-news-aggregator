# RSS News Aggregator

This service periodically fetches Indian news headlines from several RSS feeds, scrapes each article for a short summary and stores the results in PostgreSQL. A small NLP service rates articles using a zero-shot classifier.

## Quick start

1. `docker-compose up --build` – starts Postgres, runs migrations, the NLP service and the API.
2. Access the API at `http://localhost:8080`.

### Endpoints

- `POST /manual/fetch` – fetch and ingest all RSS feeds immediately.
- `GET  /articles/all` – retrieve the latest stored articles.
- `POST /manual/scrape/v1` with `{ "url": "https://example.com" }` – scrape a single article.

The ingestor also runs hourly via the scheduler.

## Environment

- `DATABASE_URL` – connection string for Postgres.
- `CLASSIFIER_URL` – URL of the NLP classification service (default `http://localhost:8000`).
