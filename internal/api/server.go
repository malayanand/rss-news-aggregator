package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/malayanand/newsx/internal/classifier"
	"github.com/malayanand/newsx/internal/fetcher"
	"github.com/malayanand/newsx/internal/store"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	db  *store.Config
	clf *classifier.Client
	mux *mux.Router
}

type ArticleOut struct {
	ID             string    `json:"id"`
	Source         string    `json:"source"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	PublishedAt    time.Time `json:"published_at"`
	Content        string    `json:"content"`
	ScrapedSummary string    `json:"scrapped_summary"`
	Rating         string    `json:"rating"`
}

type scrapeRequest struct {
	URL string `json:"url"`
}

func NewServer(dbConfig *store.Config, clf *classifier.Client) *Server {
	s := &Server{
		db:  dbConfig,
		clf: clf,
		mux: mux.NewRouter(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/manual/fetch", s.HandleManualFetch).Methods("POST")
	s.mux.HandleFunc("/articles/all", s.HandleAllArticles).Methods("GET")
	s.mux.HandleFunc("/manual/scrape/v1", s.HandleSingleScrape).Methods("POST")
}

func (s *Server) HandleManualFetch(w http.ResponseWriter, r *http.Request) {
	// for testing setting the context timeout to 2 min
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	articles, err := fetcher.FetchFromRSS()
	if err != nil {
		http.Error(w, "Fetch error: "+err.Error(), http.StatusBadGateway)
		return
	}

	//TODO: Change the from errgroups to waitgroups since use of errgroups here serves little purpose
	upsertCh := store.StartArticleIngestor(ctx, s.db)
	classifierCh := classifier.StartArticleClassifier(ctx, s.db, s.clf)

	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, 10)
	var newCount int64

	for i := range articles {
		if err := gctx.Err(); err != nil {
			log.Printf("manual-fetch: context done (%v), aborting remaining scrapes\n", err)
			break
		}

		article := &articles[i]
		cached, _ := s.db.CheckCache(gctx, article.URL)
		log.Printf("scrapping: %s - %v", article.Title, cached)
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
				log.Println("scrape error: ", article.URL, err)
				return nil // so as to not fail the whole group
			}
			article.ScrappedSummary = summary

			if gctx.Err() == nil {
				upsertCh <- *article
				classifierCh <- *article
				atomic.AddInt64(&newCount, 1)
			}
			return nil
		})
	}

	_ = g.Wait()
	close(upsertCh)
	close(classifierCh)

	resp := map[string]int{"fetched": len(articles), "new": int(atomic.LoadInt64(&newCount))}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) HandleAllArticles(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	rows, err := s.db.DB.QueryContext(ctx, `
                SELECT id, source, title, url, published_at, content, scrapped_summary, rating FROM articles ORDER BY published_at DESC LIMIT 20
                `)
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []ArticleOut
	for rows.Next() {
		var a ArticleOut
		var summary sql.NullString
		var rating sql.NullString

		err := rows.Scan(&a.ID, &a.Source, &a.Title, &a.URL, &a.PublishedAt, &a.Content, &summary, &rating)
		if err != nil {
			http.Error(w, "scan error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if summary.Valid {
			a.ScrapedSummary = summary.String
		} else {
			a.ScrapedSummary = ""
		}
		if rating.Valid {
			a.Rating = rating.String
		}
		list = append(list, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) HandleSingleScrape(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req scrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "missing url field", http.StatusBadRequest)
		return
	}

	// Now you can use req.URL
	summary, err := fetcher.ScrapeSummary(r.Context(), req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"summary": summary,
	})
}
