package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/genai"
)

type ImageGenServer struct {
	client *genai.Client
}

func NewImageGenServer(client *genai.Client) *ImageGenServer {
	return &ImageGenServer{client: client}
}

func (s *ImageGenServer) GenerateImage(ctx context.Context, prompt string) (string, error) {
	fmt.Printf("[ImageGen] Generating image with prompt: %s\n", prompt)

	config := &genai.GenerateImagesConfig{
		NumberOfImages: 1,
		AspectRatio:    "16:9",
		OutputMIMEType: "image/png",
	}

	resp, err := s.client.Models.GenerateImages(ctx, "imagen-4.0-generate-001", prompt, config)
	if err != nil {
		return "", fmt.Errorf("gagal generate image: %w", err)
	}

	if len(resp.GeneratedImages) == 0 {
		return "", fmt.Errorf("tidak ada image yang di-generate")
	}

	genImage := resp.GeneratedImages[0]
	if genImage.Image == nil || len(genImage.Image.ImageBytes) == 0 {
		return "", fmt.Errorf("image data kosong")
	}

	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("generated_%d.png", timestamp)
	outputPath := filepath.Join("uploads", filename)

	if err := os.MkdirAll("uploads", 0755); err != nil {
		return "", fmt.Errorf("gagal buat directory: %w", err)
	}

	if err := os.WriteFile(outputPath, genImage.Image.ImageBytes, 0644); err != nil {
		return "", fmt.Errorf("gagal simpan image: %w", err)
	}

	fmt.Printf("[ImageGen] Image saved to: %s\n", outputPath)

	return outputPath, nil
}

func (s *ImageGenServer) GenerateImageBase64(ctx context.Context, prompt string) (string, string, error) {
	fmt.Printf("[ImageGen] Generating image (base64) with prompt: %s\n", prompt)

	config := &genai.GenerateImagesConfig{
		NumberOfImages: 1,
		AspectRatio:    "16:9",
		OutputMIMEType: "image/png",
	}

	resp, err := s.client.Models.GenerateImages(ctx, "imagen-4.0-generate-001", prompt, config)
	if err != nil {
		return "", "", fmt.Errorf("gagal generate image: %w", err)
	}

	if len(resp.GeneratedImages) == 0 {
		return "", "", fmt.Errorf("tidak ada image yang di-generate")
	}

	genImage := resp.GeneratedImages[0]
	if genImage.Image == nil || len(genImage.Image.ImageBytes) == 0 {
		return "", "", fmt.Errorf("image data kosong")
	}

	b64 := base64.StdEncoding.EncodeToString(genImage.Image.ImageBytes)
	mimeType := genImage.Image.MIMEType
	if mimeType == "" {
		mimeType = "image/png"
	}

	return b64, mimeType, nil
}
