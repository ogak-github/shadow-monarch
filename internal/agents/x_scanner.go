package agents

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/genai"
)

type XScannerAgent struct {
	client *genai.Client
}

func NewXScannerAgent(client *genai.Client) *XScannerAgent {
	return &XScannerAgent{client: client}
}

func (a *XScannerAgent) generateWithRetry(ctx context.Context, prompt string) (string, error) {
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := a.client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*genai.Content{
			{Role: "user", Parts: []*genai.Part{{Text: prompt}}},
		}, nil)
		if err == nil {
			return resp.Text(), nil
		}
		lastErr = err
		fmt.Printf("[XScanner] Attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}
	}
	return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (a *XScannerAgent) ScanTrends(ctx context.Context, topic string) (string, error) {
	prompt := fmt.Sprintf(`Analisis tren X/Twitter untuk topik tech: %s

Fokus: Golang, Flutter/Dart, Android/Kotlin Compose, System Architecture

Cari:
- Hashtag yang lagi trending
- Diskusi dan debat yang lagi rame
- Pain point yang developer rasain
- Hook untuk engagement (opini kontroversial, pertanyaan menarik)

Response dengan Bahasa Indonesia, ringkas dan jelas.`, topic)

	return a.generateWithRetry(ctx, prompt)
}

func (a *XScannerAgent) SearchPosts(ctx context.Context, query string) (string, error) {
	prompt := fmt.Sprintf(`Cari dan analisis postingan X/Twitter tentang: %s

Fokus tech: Golang, Flutter, Android/Kotlin, System Architecture

Return dengan Bahasa Indonesia:
1. Diskusi dan debat yang happening
2. Sentimen developer (positif/negatif/pain points)
3. Hot takes dan opini kontroversial
4. Pola engagement (apa yang dapet likes, retweets)
5. Gap konten (topik yang belum dibahas tapi harusnya ada)`, query)

	return a.generateWithRetry(ctx, prompt)
}