package classifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client wraps http calls to the NLP service
type Client struct {
	baseURL string
	httpc   *http.Client
}

type classifyRequest struct {
	Text string `json:"text"`
}

type classifyResponse struct {
	Rating string `json:"rating"`
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	return &Client{baseURL: baseURL, httpc: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) Classify(ctx context.Context, text string) (string, error) {
	payload, _ := json.Marshal(classifyRequest{Text: text})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/classify", bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("classifier bad status: %s", resp.Status)
	}
	var out classifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Rating, nil
}
