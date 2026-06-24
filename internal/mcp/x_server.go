package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type XServer struct {
	bearerToken string
}

type tweetResponse struct {
	Data struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"data"`
}

type createTweetRequest struct {
	Text string `json:"text"`
}

type replyTweetRequest struct {
	Text  string `json:"text"`
	Reply struct {
		InReplyToTweetID string `json:"in_reply_to_tweet_id"`
	} `json:"reply"`
}

func NewXServer() *XServer {
	return &XServer{
		bearerToken: os.Getenv("X_BEARER_TOKEN"),
	}
}

func (s *XServer) postTweet(ctx context.Context, text string, replyToID string) (string, error) {
	if s.bearerToken == "" {
		return "", fmt.Errorf("X_BEARER_TOKEN belum dikonfigurasi di .env")
	}

	var payload interface{}
	if replyToID != "" {
		req := replyTweetRequest{}
		req.Text = text
		req.Reply.InReplyToTweetID = replyToID
		payload = req
	} else {
		payload = createTweetRequest{Text: text}
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gagal encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.x.com/2/tweets", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("gagal buat request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.bearerToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal kirim request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 201 {
		return "", fmt.Errorf("Twitter API error (%d): %s", resp.StatusCode, string(body))
	}

	var tweetResp tweetResponse
	if err := json.Unmarshal(body, &tweetResp); err != nil {
		return "", fmt.Errorf("gagal parse response: %w", err)
	}

	return tweetResp.Data.ID, nil
}

func (s *XServer) PostTweet(ctx context.Context, content string) (string, error) {
	return s.postTweet(ctx, content, "")
}

func (s *XServer) PostThread(ctx context.Context, tweets []string) (string, error) {
	if s.bearerToken == "" {
		return "", fmt.Errorf("X_BEARER_TOKEN belum dikonfigurasi di .env")
	}

	if len(tweets) == 0 {
		return "", fmt.Errorf("tidak ada tweet untuk diposting")
	}

	var postedIDs []string
	var lastTweetID string

	for i, tweet := range tweets {
		tweetText := tweet
		if tweetText == "" {
			continue
		}

		fmt.Printf("[X MCP] Posting tweet %d/%d...\n", i+1, len(tweets))

		tweetID, err := s.postTweet(ctx, tweetText, lastTweetID)
		if err != nil {
			return "", fmt.Errorf("gagal posting tweet %d: %w", i+1, err)
		}

		postedIDs = append(postedIDs, tweetID)
		lastTweetID = tweetID

		fmt.Printf("[X MCP] Tweet %d posted: %s\n", i+1, tweetID)

		if i < len(tweets)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	return fmt.Sprintf("Thread berhasil! %d tweets posted. IDs: %v", len(postedIDs), postedIDs), nil
}

func (s *XServer) SearchPosts(ctx context.Context, query string) ([]string, error) {
	return nil, fmt.Errorf("search belum diimplementasi")
}

func (s *XServer) GetTrends(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("trends belum diimplementasi")
}