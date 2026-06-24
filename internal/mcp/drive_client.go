package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type DriveClient struct {
	service    *drive.Service
	folderID   string
}

func NewDriveClient(credentialsPath, folderID string) (*DriveClient, error) {
	ctx := context.Background()

	credData, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("gagal baca credentials: %w", err)
	}

	config, err := google.JWTConfigFromJSON(credData, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("gagal parse credentials: %w", err)
	}

	client := config.Client(ctx)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("gagal buat drive service: %w", err)
	}

	return &DriveClient{
		service:  srv,
		folderID: folderID,
	}, nil
}

func (d *DriveClient) UploadImage(ctx context.Context, imagePath string) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("gagal buka file: %w", err)
	}
	defer file.Close()

	fileName := filepath.Base(imagePath)
	mimeType := "image/png"
	switch ext := filepath.Ext(imagePath); ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	}

	folderID := d.folderID
	if folderID == "" {
		folderID = "root"
	}

	fileMeta := &drive.File{
		Name:     fileName,
		MimeType: mimeType,
		Parents:  []string{folderID},
	}

	result, err := d.service.Files.Create(fileMeta).Media(file).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("gagal upload ke drive: %w", err)
	}

	perm := &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}
	_, err = d.service.Permissions.Create(result.Id, perm).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("gagal set permission: %w", err)
	}

	imageURL := fmt.Sprintf("https://drive.google.com/uc?export=view&id=%s", result.Id)
	return imageURL, nil
}

func (d *DriveClient) CleanupOldImages(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	query := fmt.Sprintf("mimeType contains 'image/' and modifiedTime < '%s'", cutoff.Format(time.RFC3339))
	if d.folderID != "" {
		query += fmt.Sprintf(" and '%s' in parents", d.folderID)
	}

	result, err := d.service.Files.List().Q(query).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("gagal list files: %w", err)
	}

	for _, file := range result.Files {
		err := d.service.Files.Delete(file.Id).Context(ctx).Do()
		if err != nil {
			fmt.Printf("[Drive] Gagal hapus %s: %v\n", file.Name, err)
		}
	}

	return nil
}


