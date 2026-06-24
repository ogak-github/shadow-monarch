package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/genai"

	"yga.my.id/shadow-monarch/internal/orchestrator"
)

func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func humanizeError(err error) string {
	msg := err.Error()

	if strings.Contains(msg, "503") || strings.Contains(msg, "UNAVAILABLE") {
		return "Layanan AI sedang sibuk, coba lagi dalam beberapa saat."
	}
	if strings.Contains(msg, "429") || strings.Contains(msg, "RESOURCE_EXHAUSTED") {
		return "Terlalu banyak request, tunggu sebentar lalu coba lagi."
	}
	if strings.Contains(msg, "401") || strings.Contains(msg, "UNAUTHENTICATED") {
		return "API key tidak valid atau belum dikonfigurasi."
	}
	if strings.Contains(msg, "403") || strings.Contains(msg, "PERMISSION_DENIED") {
		return "Akses ditolak. Periksa API key kamu."
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "DEADLINE_EXCEEDED") {
		return "Request timeout. Coba lagi dengan topik yang lebih simpel."
	}
	if strings.Contains(msg, "scanner error") {
		return "Gagal memindai tren. " + humanizeErrorInner(msg)
	}
	if strings.Contains(msg, "research error") {
		return "Gagal melakukan riset. " + humanizeErrorInner(msg)
	}
	if strings.Contains(msg, "draft error") {
		return "Gagal membuat konten. " + humanizeErrorInner(msg)
	}
	if strings.Contains(msg, "post error") {
		return "Gagal memposting ke X/Twitter. " + humanizeErrorInner(msg)
	}
	if strings.Contains(msg, "X_BEARER_TOKEN") || strings.Contains(msg, "belum dikonfigurasi") {
		return "X/Twitter API belum dikonfigurasi. Set env X_BEARER_TOKEN di .env"
	}
	if strings.Contains(msg, "DEVTO_API_KEY") {
		return "Dev.to API belum dikonfigurasi. Set env DEVTO_API_KEY di .env"
	}
	if strings.Contains(msg, "publish error") {
		return "Gagal publish ke Dev.to. " + humanizeErrorInner(msg)
	}
	if strings.Contains(msg, "failed after") {
		return "Gagal setelah beberapa percobaan. Layanan mungkin sedang gangguan."
	}

	return "Terjadi kesalahan. Coba lagi nanti."
}

func humanizeErrorInner(msg string) string {
	if strings.Contains(msg, "503") || strings.Contains(msg, "UNAVAILABLE") {
		return "Layanan AI sedang sibuk."
	}
	if strings.Contains(msg, "429") {
		return "Rate limit terlampaui."
	}
	return ""
}

func main() {
	godotenv.Load()

	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: GEMINI_API_KEY not set")
		os.Exit(1)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		fmt.Printf("ERROR: Failed to create Gemini client: %v\n", err)
		os.Exit(1)
	}

	engine := orchestrator.NewOrchestratorEngine(client)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	http.HandleFunc("/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "Method tidak diizinkan. Gunakan POST.")
			return
		}

		var req struct {
			Topic        string `json:"topic"`
			AutoPost     bool   `json:"auto_post,omitempty"`
			Platform     string `json:"platform,omitempty"`
			GenerateImage *bool  `json:"generate_image,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Format request tidak valid.")
			return
		}

		if req.Topic == "" {
			respondError(w, http.StatusBadRequest, "Field 'topic' harus diisi.")
			return
		}

		if req.Platform == "" {
			req.Platform = "devto"
		}

		if req.Platform != "devto" && req.Platform != "x" {
			respondError(w, http.StatusBadRequest, "Field 'platform' harus 'devto' atau 'x'.")
			return
		}

		doGenerateImage := true
		if req.GenerateImage != nil {
			doGenerateImage = *req.GenerateImage
		}

		var result string
		var err error
		if req.AutoPost {
			result, err = engine.AutoPostWithPlatform(r.Context(), req.Topic, req.Platform, doGenerateImage)
		} else {
			result, err = engine.ProcessRequest(r.Context(), req.Topic, doGenerateImage)
		}

		if err != nil {
			fmt.Printf("[/generate] Error: %v\n", err)
			respondError(w, http.StatusInternalServerError, humanizeError(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"result": result})
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "Method tidak diizinkan. Gunakan POST.")
			return
		}

		var req struct {
			Message string `json:"message"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Format request tidak valid.")
			return
		}

		if req.Message == "" {
			respondError(w, http.StatusBadRequest, "Field 'message' harus diisi.")
			return
		}

		result, err := engine.ProcessRequest(r.Context(), req.Message, false)
		if err != nil {
			respondError(w, http.StatusInternalServerError, humanizeError(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"response": result})
	})

	http.HandleFunc("/generate-image", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "Method tidak diizinkan. Gunakan POST.")
			return
		}

		var req struct {
			Prompt string `json:"prompt"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Format request tidak valid.")
			return
		}

		if req.Prompt == "" {
			respondError(w, http.StatusBadRequest, "Field 'prompt' harus diisi.")
			return
		}

		imagePath, err := engine.GenerateImage(r.Context(), req.Prompt)
		if err != nil {
			fmt.Printf("[/generate-image] Error: %v\n", err)
			respondError(w, http.StatusInternalServerError, humanizeError(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"image_path": imagePath})
	})

	server := &http.Server{
		Addr:    ":9099",
		Handler: nil,
	}

	go func() {
		fmt.Println("🚀 Server starting on :9099")
		fmt.Println("   POST /generate - Generate & post content (platform: devto/x)")
		fmt.Println("   POST /generate-image - Generate image from prompt")
		fmt.Println("   POST /chat - Chat with the agent")
		fmt.Println("   GET  /health - Health check")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("ERROR: Server failed: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("ERROR: Server forced to shutdown: %v\n", err)
	}

	fmt.Println("✅ Server stopped")
}