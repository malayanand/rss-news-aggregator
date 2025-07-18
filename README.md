# RSS News Aggregator

This service periodically fetches Indian news headlines from several RSS feeds, scrapes each article for a short summary and stores the results in PostgreSQL. A small NLP service rates articles using a zero-shot classifier.

## Setup

1. Install Docker and Docker Compose.
2. Clone this repository and open the project directory.
3. Run `docker-compose up --build` to build images, run migrations and start all services.
4. Once the containers are running, visit `http://localhost:8080` to access the API.

### Endpoints

- `POST /manual/fetch` – fetch and ingest all RSS feeds immediately.
- `GET  /articles/all` – retrieve the latest stored articles.
- `POST /manual/scrape/v1` with `{ "url": "https://example.com" }` – scrape a single article.

The ingestor also runs hourly via the scheduler.

