package orchestrator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"google.golang.org/genai"

	"yga.my.id/shadow-monarch/internal/agents"
	"yga.my.id/shadow-monarch/internal/mcp"
)

type OrchestratorEngine struct {
	client    *genai.Client
	xScanner  *agents.XScannerAgent
	gcpExpert *agents.GCPExpertAgent
	xDrafter  *agents.XDraftAgent
	xServer   *mcp.XServer
	imageGen  *mcp.ImageGenServer
	devTo     *mcp.DevToServer
}

func NewOrchestratorEngine(client *genai.Client) *OrchestratorEngine {
	return &OrchestratorEngine{
		client:    client,
		xScanner:  agents.NewXScannerAgent(client),
		gcpExpert: agents.NewGCPExpertAgent(client),
		xDrafter:  agents.NewXDraftAgent(client),
		xServer:   mcp.NewXServer(),
		imageGen:  mcp.NewImageGenServer(client),
		devTo:     mcp.NewDevToServer(),
	}
}

type agentResult struct {
	data string
	err  error
}

func (e *OrchestratorEngine) ProcessRequest(ctx context.Context, userInput string, generateImage bool) (string, error) {
	fmt.Printf("[Orchestrator] Processing request: %s\n", userInput)

	scanResultCh := make(chan agentResult, 1)
	researchCh := make(chan agentResult, 1)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		fmt.Printf("[Orchestrator] Starting X Scanner...\n")
		result, err := e.xScanner.ScanTrends(ctx, userInput)
		scanResultCh <- agentResult{data: result, err: err}
	}()

	go func() {
		defer wg.Done()
		fmt.Printf("[Orchestrator] Starting GCP Expert...\n")
		result, err := e.gcpExpert.Research(ctx, userInput)
		researchCh <- agentResult{data: result, err: err}
	}()

	wg.Wait()
	close(scanResultCh)
	close(researchCh)

	scanResult := <-scanResultCh
	if scanResult.err != nil {
		return "", fmt.Errorf("scanner error: %w", scanResult.err)
	}
	fmt.Printf("[Orchestrator] Scanner done\n")

	researchResult := <-researchCh
	if researchResult.err != nil {
		return "", fmt.Errorf("research error: %w", researchResult.err)
	}
	fmt.Printf("[Orchestrator] Research done\n")

	combinedContext := fmt.Sprintf("Trends: %s\n\nResearch: %s", scanResult.data, researchResult.data)

	threadContent, err := e.xDrafter.DraftThread(ctx, combinedContext)
	if err != nil {
		return "", fmt.Errorf("draft error: %w", err)
	}
	fmt.Printf("[Orchestrator] Draft result: %s\n", truncate(threadContent, 100))

	if generateImage {
		imagePath, err := e.imageGen.GenerateImage(ctx, userInput)
		if err != nil {
			fmt.Printf("[Orchestrator] Image generation failed: %v\n", err)
		} else {
			threadContent += "\n\nImage: " + imagePath
		}
	}

	return threadContent, nil
}

func (e *OrchestratorEngine) AutoPost(ctx context.Context, userInput string, generateImage bool) (string, error) {
	return e.AutoPostWithPlatform(ctx, userInput, "x", generateImage)
}

func (e *OrchestratorEngine) AutoPostWithPlatform(ctx context.Context, userInput string, platform string, generateImage bool) (string, error) {
	switch platform {
	case "devto":
		return e.publishToDevTo(ctx, userInput, generateImage)
	case "x":
		return e.postToX(ctx, userInput, generateImage)
	default:
		return "", fmt.Errorf("platform tidak valid: %s (gunakan 'devto' atau 'x')", platform)
	}
}

func (e *OrchestratorEngine) postToX(ctx context.Context, userInput string, generateImage bool) (string, error) {
	result, err := e.ProcessRequest(ctx, userInput, generateImage)
	if err != nil {
		return "", err
	}

	tweets := strings.Split(result, "\n")
	for i := range tweets {
		tweets[i] = strings.TrimSpace(tweets[i])
	}

	postResult, err := e.xServer.PostThread(ctx, tweets)
	if err != nil {
		return "", fmt.Errorf("post error: %w", err)
	}

	return fmt.Sprintf("Posted successfully: %s\n%s", postResult, result), nil
}

func (e *OrchestratorEngine) PublishToDevTo(ctx context.Context, userInput string, generateImage bool) (string, error) {
	return e.publishToDevTo(ctx, userInput, generateImage)
}

func (e *OrchestratorEngine) publishToDevTo(ctx context.Context, userInput string, generateImage bool) (string, error) {
	fmt.Printf("[Orchestrator] Publishing to Dev.to: %s\n", userInput)

	scanResultCh := make(chan agentResult, 1)
	researchCh := make(chan agentResult, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		result, err := e.xScanner.ScanTrends(ctx, userInput)
		scanResultCh <- agentResult{data: result, err: err}
	}()

	go func() {
		defer wg.Done()
		result, err := e.gcpExpert.Research(ctx, userInput)
		researchCh <- agentResult{data: result, err: err}
	}()

	wg.Wait()
	close(scanResultCh)
	close(researchCh)

	scanResult := <-scanResultCh
	if scanResult.err != nil {
		return "", fmt.Errorf("scanner error: %w", scanResult.err)
	}

	researchResult := <-researchCh
	if researchResult.err != nil {
		return "", fmt.Errorf("research error: %w", researchResult.err)
	}

	combinedContext := fmt.Sprintf("Trends: %s\n\nResearch: %s", scanResult.data, researchResult.data)

	fmt.Printf("[Orchestrator] Generating title...\n")
	generatedTitle, err := e.gcpExpert.GenerateTitle(ctx, userInput)
	if err != nil {
		fmt.Printf("[Orchestrator] Title generation failed, falling back to draft title: %v\n", err)
	}

	_, body, err := e.xDrafter.DraftDevToArticle(ctx, combinedContext)
	if err != nil {
		return "", fmt.Errorf("draft error: %w", err)
	}

	title := cleanGeneratedTitle(generatedTitle)
	if title == "" {
		title = extractTitleFromBody(body)
	}

	if generateImage {
		imagePath, err := e.imageGen.GenerateImage(ctx, userInput)
		if err != nil {
			fmt.Printf("[Orchestrator] Image generation failed: %v\n", err)
		} else {
			imageURL, uploadErr := e.devTo.UploadImage(ctx, imagePath)
			os.Remove(imagePath)
			if uploadErr != nil {
				fmt.Printf("[Orchestrator] Image upload failed: %v\n", uploadErr)
			} else {
				body += "\n\n![Cover Image](" + imageURL + ")"
			}
		}
	}

	tags := []string{"programming", "tutorial", "tech"}

	publishResult, err := e.devTo.PostArticle(ctx, title, body, tags)
	if err != nil {
		fmt.Printf("[Orchestrator] Dev.to publish error: %v\n", err)
		return "", fmt.Errorf("publish error: %w", err)
	}

	return publishResult, nil
}

func (e *OrchestratorEngine) GenerateImage(ctx context.Context, prompt string) (string, error) {
	return e.imageGen.GenerateImage(ctx, prompt)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func cleanGeneratedTitle(raw string) string {
	raw = strings.TrimSpace(raw)
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "Judul:")
		line = strings.TrimPrefix(line, "Judul :")
		line = strings.TrimSpace(line)
		if len(line) > 5 && len(line) <= 128 {
			return line
		}
	}
	return ""
}

func extractTitleFromBody(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimPrefix(line, "# ")
			if len(title) > 0 && len(title) <= 128 {
				return title
			}
		}
	}
	return ""
}