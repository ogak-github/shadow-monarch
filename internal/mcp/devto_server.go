package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type DevToServer struct {
	apiKey string
}

type devtoArticleRequest struct {
	Title          string   `json:"title"`
	BodyMarkdown   string   `json:"body_markdown"`
	Published      bool     `json:"published"`
	Tags           []string `json:"tags"`
	MainImage      string   `json:"main_image,omitempty"`
	OrganizationID string   `json:"organization_id,omitempty"`
}

type devtoArticleResponse struct {
	ID             int    `json:"id"`
	URL            string `json:"url"`
	Title          string `json:"title"`
	Published      bool   `json:"published"`
	Slug           string `json:"slug"`
}

func NewDevToServer() *DevToServer {
	return &DevToServer{
		apiKey: os.Getenv("DEVTO_API_KEY"),
	}
}

func (s *DevToServer) PostArticle(ctx context.Context, title, bodyMarkdown string, tags []string, imageURL string) (string, error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("DEVTO_API_KEY belum dikonfigurasi di .env")
	}

	if len(tags) > 4 {
		tags = tags[:4]
	}

	article := devtoArticleRequest{
		Title:        title,
		BodyMarkdown: bodyMarkdown,
		Published:    true,
		Tags:         tags,
		MainImage:    imageURL,
	}

	fmt.Printf("[DevTo] Posting article: %s (tags: %v)\n", title, tags)

	payload := map[string]interface{}{
		"article": article,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gagal encode article: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://dev.to/api/articles", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("gagal buat request: %w", err)
	}

	req.Header.Set("api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal kirim request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 201 {
		fmt.Printf("[DevTo] API error (status %d): %s\n", resp.StatusCode, string(body))
		return "", fmt.Errorf("Dev.to API error (%d): %s", resp.StatusCode, string(body))
	}

	var articleResp devtoArticleResponse
	if err := json.Unmarshal(body, &articleResp); err != nil {
		return "", fmt.Errorf("gagal parse response: %w", err)
	}

	return fmt.Sprintf("Article posted! URL: %s", articleResp.URL), nil
}

func (s *DevToServer) PostTutorial(ctx context.Context, title, content string, tags []string, imageURL string) (string, error) {
	return s.PostArticle(ctx, title, content, tags, imageURL)
}

func (s *DevToServer) ImageToBase64(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("gagal baca file: %w", err)
	}

	ext := filepath.Ext(imagePath)
	mimeType := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}
