package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Analyzer struct {
	client *genai.Client
}

func NewAnalyzer(ctx context.Context, apiKey string) (*Analyzer, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	return &Analyzer{
		client: client,
	}, nil
}

func (a *Analyzer) Close() {
	if a.client != nil {
		a.client.Close()
	}
}

type ComplexityAnalysis struct {
	TimeComplexity  string `json:"time_complexity"`
	SpaceComplexity string `json:"space_complexity"`
	Feedback        string `json:"feedback"`
}

func (a *Analyzer) AnalyzeComplexity(ctx context.Context, sourceCode string, language string) (*ComplexityAnalysis, error) {
	model := a.client.GenerativeModel("gemini-3.5-flash")
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text("You are an expert computer science tutor. Analyze the provided code and return a JSON object with 'time_complexity', 'space_complexity', and 'feedback'. 'time_complexity' and 'space_complexity' should be standard Big O notation like 'O(n)'. 'feedback' should be a short 1-2 sentence explanation."),
		},
	}

	prompt := fmt.Sprintf("Language: %s\nCode:\n%s", language, sourceCode)
	
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response type")
	}

	// The response should be JSON due to ResponseMIMEType setting
	var analysis ComplexityAnalysis
	if err := json.Unmarshal([]byte(string(text)), &analysis); err != nil {
		// Try to clean up markdown if present
		cleanStr := strings.TrimPrefix(string(text), "```json\n")
		cleanStr = strings.TrimSuffix(cleanStr, "\n```")
		cleanStr = strings.TrimSuffix(cleanStr, "```")
		
		if err2 := json.Unmarshal([]byte(cleanStr), &analysis); err2 != nil {
			return nil, fmt.Errorf("failed to parse JSON from AI: %w, text: %s", err2, string(text))
		}
	}

	return &analysis, nil
}
