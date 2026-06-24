package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ImgBBClient struct {
	apiKey string
}

type imgBBResponse struct {
	Data struct {
		URL      string `json:"url"`
		Display  string `json:"display_url"`
		Delete   string `json:"delete_url"`
	} `json:"data"`
	Success bool `json:"success"`
}

func NewImgBBClient() *ImgBBClient {
	return &ImgBBClient{
		apiKey: os.Getenv("IMGBB_API_KEY"),
	}
}

func (c *ImgBBClient) IsConfigured() bool {
	return c.apiKey != ""
}

func (c *ImgBBClient) UploadImage(ctx context.Context, imagePath string) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("IMGBB_API_KEY belum dikonfigurasi")
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("gagal buka file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("gagal baca file: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("key", c.apiKey)
	_ = writer.WriteField("image", encoded)

	ext := filepath.Ext(imagePath)
	if ext == ".jpg" || ext == ".jpeg" {
		_ = writer.WriteField("type", "jpg")
	} else if ext == ".png" {
		_ = writer.WriteField("type", "png")
	} else if ext == ".gif" {
		_ = writer.WriteField("type", "gif")
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.imgbb.com/1/upload", &buf)
	if err != nil {
		return "", fmt.Errorf("gagal buat request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal upload ke imgbb: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("imgbb error (%d): %s", resp.StatusCode, string(body))
	}

	var imgResp imgBBResponse
	if err := json.Unmarshal(body, &imgResp); err != nil {
		return "", fmt.Errorf("gagal parse response: %w", err)
	}

	if !imgResp.Success {
		return "", fmt.Errorf("imgbb upload gagal")
	}

	return imgResp.Data.URL, nil
}
