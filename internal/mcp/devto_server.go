package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

func (s *DevToServer) PostArticle(ctx context.Context, title, bodyMarkdown string, tags []string) (string, error) {
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

func (s *DevToServer) PostTutorial(ctx context.Context, title, content string, tags []string) (string, error) {
	return s.PostArticle(ctx, title, content, tags)
}

func (s *DevToServer) UploadImage(ctx context.Context, imagePath string) (string, error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("DEVTO_API_KEY belum dikonfigurasi")
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("gagal buka file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return "", fmt.Errorf("gagal buat form: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("gagal copy file: %w", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://dev.to/api/images", &buf)
	if err != nil {
		return "", fmt.Errorf("gagal buat request: %w", err)
	}

	req.Header.Set("api-key", s.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal upload image: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Dev.to image upload error (%d): %s", resp.StatusCode, string(body))
	}

	var imgResp struct {
		Link string `json:"link"`
	}
	if err := json.Unmarshal(body, &imgResp); err != nil {
		return "", fmt.Errorf("gagal parse response: %w", err)
	}

	return imgResp.Link, nil
}
