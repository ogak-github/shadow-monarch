package agents

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/genai"
)

type XDraftAgent struct {
	client *genai.Client
}

func NewXDraftAgent(client *genai.Client) *XDraftAgent {
	return &XDraftAgent{client: client}
}

func (a *XDraftAgent) generateWithRetry(ctx context.Context, prompt string) (string, error) {
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
		fmt.Printf("[XDraft] Attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}
	}
	return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (a *XDraftAgent) DraftThread(ctx context.Context, research string) (string, error) {
	prompt := fmt.Sprintf(`Based on this tech research: %s

Buat thread X/Twitter (3-5 tweets) dengan Bahasa Indonesia.

PENTING - Persona:
- Bahasa Indonesia informal (gw/gue, lo/kamu, kayak, sih, dong, ya)
- Humble, bukan sok tahu
- Chatty, kayak lagi ngobrol sama temen
- Penjelasan simpel, jangan pake jargon berlebihan
- Boleh campur istilah teknis Inggris tapi tetap jelas

Struktur thread:
1. Tweet pertama: HOOK - opini kontroversial, fakta mengejutkan, atau pain point yang relate
2. Tweet tengah: VALUE - tips praktis, code snippet, atau insight
3. Tweet terakhir: CTA - ajak diskusi, tanya pendapat, atau kasih challenge

Tips writing:
- Kalimat pendek-pendek, gampang dibaca
- Boleh pake code snippet relevan
- Emoji secukupnya (jangan berlebihan)
- Sertakan 2-3 hashtag relevan
- Sound like senior dev yang lagi share pengalaman, bukan korporat

Return format:
Tweet 1: ...
Tweet 2: ...
etc.

Hashtags: #Golang #Flutter #AndroidDev #SoftwareArchitecture #TeknologiIndonesia`, research)

	return a.generateWithRetry(ctx, prompt)
}

func (a *XDraftAgent) DraftDevToArticle(ctx context.Context, research string) (string, string, error) {
	prompt := fmt.Sprintf(`Based on this tech research: %s

Buat article untuk Dev.to dengan Bahasa Indonesia.

PENTING - Persona:
- Bahasa Indonesia informal (gw/gue, lo/kamu, kayak, sih, dong, ya)
- Humble, bukan sok tahu
- Penjelasan simpel dan jelas
- Sertakan code snippet yang relevan

Struktur article:
1. Title yang menarik (max 128 char)
2. Opening paragraph - hook readers
3. Isi article dengan heading yang jelas (pakai ##)
4. Code snippets dengan syntax highlighting
5. Kesimpulan dan CTA

Format: Markdown

Return format:
TITLE: [judul article]
TAGS: [tag1, tag2, tag3]
BODY:
[isi article dalam markdown]`, research)

	result, err := a.generateWithRetry(ctx, prompt)
	if err != nil {
		return "", "", err
	}

	title := extractBetween(result, "TITLE:", "TAGS:")
	tags := extractBetween(result, "TAGS:", "BODY:")
	body := extractAfter(result, "BODY:")

	title = cleanString(title)
	tags = cleanString(tags)
	body = cleanString(body)

	return title, body, nil
}

func (a *XDraftAgent) GenerateImagePrompt(ctx context.Context, topic string) (string, error) {
	prompt := fmt.Sprintf(`Generate an image prompt for a tech tweet about: %s

Create a vivid, engaging visual description for social media.
Style: Modern tech aesthetic, clean, professional but approachable

Ideas:
- Code snippets floating in space
- Developer workspace setups
- Terminal/command line aesthetics
- Tech stack logos in creative arrangement
- Before/after comparisons (slow vs fast, messy vs clean)

Return ONLY the image prompt description.`, topic)

	return a.generateWithRetry(ctx, prompt)
}

func extractBetween(s, start, end string) string {
	i := indexOf(s, start)
	if i == -1 {
		return ""
	}
	s = s[i+len(start):]
	j := indexOf(s, end)
	if j == -1 {
		return s
	}
	return s[:j]
}

func extractAfter(s, marker string) string {
	i := indexOf(s, marker)
	if i == -1 {
		return s
	}
	return s[i+len(marker):]
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func cleanString(s string) string {
	result := []byte{}
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' || (i > 0 && s[i-1] != '\n') {
			result = append(result, s[i])
		}
	}
	start := 0
	for start < len(result) && result[start] == '\n' {
		start++
	}
	end := len(result)
	for end > start && result[end-1] == '\n' {
		end--
	}
	return string(result[start:end])
}