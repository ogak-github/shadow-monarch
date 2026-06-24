package agents

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/genai"
)

type GCPExpertAgent struct {
	client *genai.Client
}

func NewGCPExpertAgent(client *genai.Client) *GCPExpertAgent {
	return &GCPExpertAgent{client: client}
}

func (a *GCPExpertAgent) generateWithRetry(ctx context.Context, prompt string) (string, error) {
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
		fmt.Printf("[GCPExpert] Attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}
	}
	return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (a *GCPExpertAgent) Research(ctx context.Context, topic string) (string, error) {
	prompt := fmt.Sprintf(`Riset mendalam tentang topik tech: %s

Target pembaca:
Junior-Senior Software Engineer

Jangan hanya menjelaskan konsep.

Analisis seperti engineer yang sudah menjalankan sistem production.

Bahas:
1. Konsep dasar singkat
2. Masalah nyata yang biasanya memunculkan kebutuhan akan solusi ini
3. Trade-off dan konsekuensi teknis
4. Kapan solusi ini tepat digunakan
5. Kapan solusi ini sebaiknya dihindari
6. Kesalahan yang sering dilakukan developer
7. Evolusi arsitektur dari skala kecil hingga besar
8. Pengalaman lapangan dan kasus produksi yang umum terjadi
9. Pendapat dan rekomendasi yang kuat beserta alasannya
10. Insight yang jarang dibahas di tutorial atau dokumentasi resmi
11. Jika butuh contoh code gunakan bahasa pemrograman yang relevan (Golang, Flutter/Dart, Android Kotlin/Compose, TypeScript)

Sertakan:
- contoh dunia nyata
- diagram ASCII jika relevan
- data atau statistik jika tersedia
- referensi sumber terpercaya

Output harus membuat pembaca berkata:
"Saya belum pernah memikirkan hal itu sebelumnya."`, topic)

	return a.generateWithRetry(ctx, prompt)
}

func (a *GCPExpertAgent) GenerateTitle(ctx context.Context, topic string) (string, error) {
	prompt := fmt.Sprintf(`Buat judul artikel tech yang ringkas, menarik, dan engaging.

Topik: %s

Aturan judul:
1. Maksimal 50 kata
2. Gunakan angka atau listicle jika relevan
3. Buat penasaran tapi tidak clickbait
4. Tunjukkan value atau insight yang didapat pembaca
5. Gunakan bahasa Indonesia yang natural dan percakapan
6. Hindari jargon berlebihan, tetapi tetap teknis
7. Bisa menggunakan pattern seperti: "Kenapa...", "Cara...", "X yang Perlu...", "Mengapa X Lebih Baik dari Y"`, topic)

	return a.generateWithRetry(ctx, prompt)
}

func (a *GCPExpertAgent) SearchGoogle(ctx context.Context, query string) (string, error) {
	prompt := fmt.Sprintf(`Cari informasi tech tentang: %s

Fokus: Golang, Flutter, Android/Kotlin, PostgreSQL, Docker, System Architecture

Ringkas dengan Bahasa Indonesia:
1. Informasi paling relevan dan terbaru
2. Insight dari dokumentasi resmi
3. Konsensus dan debat komunitas
4. Detail implementasi praktis
5. Perubahan versi (jika applicable)

Spesifik dan actionable. Sertakan code snippet jika relevan.`, query)

	return a.generateWithTools(ctx, prompt)
}

func (a *GCPExpertAgent) generateWithTools(ctx context.Context, prompt string) (string, error) {
	maxRetries := 3
	var lastErr error

	tools := []*genai.Tool{
		{GoogleSearch: &genai.GoogleSearch{}},
	}

	for i := 0; i < maxRetries; i++ {
		resp, err := a.client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*genai.Content{
			{Role: "user", Parts: []*genai.Part{{Text: prompt}}},
		}, &genai.GenerateContentConfig{
			Tools: tools,
		})
		if err == nil {
			return resp.Text(), nil
		}
		lastErr = err
		fmt.Printf("[GCPExpert] Attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}
	}
	return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}