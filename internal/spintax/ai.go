package spintax

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AIClient handles communication with Claude AI for spintax generation
type AIClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	model      string
}

// NewAIClient creates a new Claude AI client
func NewAIClient(apiKey string) *AIClient {
	return &AIClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: "https://api.anthropic.com/v1/messages",
		model:   "claude-sonnet-4-20250514",
	}
}

// ClaudeRequest represents a request to Claude API
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
}

// ClaudeMessage represents a message in Claude conversation
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents a response from Claude API
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateSpintaxOptions contains options for spintax generation
type GenerateSpintaxOptions struct {
	// VariationLevel controls how many variations to generate (1-5, default 3)
	VariationLevel int
	// PreserveHTML keeps HTML tags intact
	PreserveHTML bool
	// PreserveLinks keeps URLs and links intact
	PreserveLinks bool
	// OnlyPhrases only creates variations for key phrases, not entire sentences
	OnlyPhrases bool
}

// GenerateSpintax uses Claude AI to add spintax variations to text
func (c *AIClient) GenerateSpintax(ctx context.Context, text string, opts *GenerateSpintaxOptions) (string, error) {
	if opts == nil {
		opts = &GenerateSpintaxOptions{
			VariationLevel: 3,
			PreserveHTML:   true,
			PreserveLinks:  true,
			OnlyPhrases:    false,
		}
	}

	if opts.VariationLevel < 1 {
		opts.VariationLevel = 1
	}
	if opts.VariationLevel > 5 {
		opts.VariationLevel = 5
	}

	systemPrompt := buildSystemPrompt(opts)
	userPrompt := buildUserPrompt(text, opts)

	req := ClaudeRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages: []ClaudeMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if claudeResp.Error != nil {
		return "", fmt.Errorf("Claude API error: %s", claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}

	return claudeResp.Content[0].Text, nil
}

func buildSystemPrompt(opts *GenerateSpintaxOptions) string {
	prompt := `You are an expert at creating spintax variations for email marketing content.

Spintax format uses curly braces with pipe-separated options: {option1|option2|option3}

Your task is to add spintax to the provided text to create natural variations while:
1. Maintaining the original meaning and tone
2. Keeping the message professional and clear
3. Creating variations that sound natural when read
4. Not over-spinning - focus on key phrases and words that benefit from variation

Rules:
- Use the format {word1|word2|word3} for variations
- Each variation should be grammatically correct in context
- Synonyms and rephrased expressions should convey the same meaning
- Don't change proper nouns, brand names, or specific product names
- Don't modify email personalization tokens like {{ .Subscriber.Name }}
- Don't modify Go template syntax like {{ if }} or {{ end }}`

	if opts.PreserveHTML {
		prompt += `
- Preserve all HTML tags exactly as they are
- Don't add spintax inside HTML attributes`
	}

	if opts.PreserveLinks {
		prompt += `
- Don't modify URLs or href values
- Keep tracking links intact`
	}

	if opts.OnlyPhrases {
		prompt += `
- Only create variations for key phrases and expressions
- Don't vary common words like "the", "and", "is"
- Focus on action words, greetings, and marketing phrases`
	}

	prompt += `

Output ONLY the modified text with spintax. Do not include explanations, markdown formatting, or anything else - just the text with spintax added.`

	return prompt
}

func buildUserPrompt(text string, opts *GenerateSpintaxOptions) string {
	levelDesc := map[int]string{
		1: "minimal (only 2-3 key phrases)",
		2: "light (a few variations per paragraph)",
		3: "moderate (regular variations throughout)",
		4: "heavy (many variations, but still readable)",
		5: "maximum (variations wherever possible)",
	}

	return fmt.Sprintf(`Add spintax to this text with %s variation level:

%s`, levelDesc[opts.VariationLevel], text)
}

// IsConfigured returns true if the AI client has an API key
func (c *AIClient) IsConfigured() bool {
	return c.apiKey != ""
}
