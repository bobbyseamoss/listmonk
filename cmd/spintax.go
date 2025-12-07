package main

import (
	"context"
	"net/http"
	"time"

	"github.com/knadh/listmonk/internal/spintax"
	"github.com/labstack/echo/v4"
)

// ProcessSpintaxRequest is the request body for processing spintax.
type ProcessSpintaxRequest struct {
	Text string `json:"text"`
}

// GenerateSpintaxRequest is the request body for AI-generated spintax.
type GenerateSpintaxRequest struct {
	Text           string `json:"text"`
	VariationLevel int    `json:"variation_level"` // 1-5, default from settings
	PreserveHTML   bool   `json:"preserve_html"`
	PreserveLinks  bool   `json:"preserve_links"`
	OnlyPhrases    bool   `json:"only_phrases"`
}

// SpintaxResponse is the response body for spintax operations.
type SpintaxResponse struct {
	Original  string `json:"original"`
	Processed string `json:"processed"`
}

// GetSpintaxSettings returns the current spintax settings.
func (a *App) GetSpintaxSettings(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{Data: struct {
		Enabled        bool `json:"enabled"`
		AIEnabled      bool `json:"ai_enabled"`
		VariationLevel int  `json:"variation_level"`
	}{
		Enabled:        a.cfg.SpintaxEnabled,
		AIEnabled:      a.cfg.SpintaxAIEnabled,
		VariationLevel: a.cfg.SpintaxVariationLevel,
	}})
}

// ProcessSpintax processes spintax in the given text and returns a random variation.
func (a *App) ProcessSpintax(c echo.Context) error {
	var req ProcessSpintaxRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Text == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "text is required")
	}

	// Check if spintax is enabled
	if !a.cfg.SpintaxEnabled {
		return echo.NewHTTPError(http.StatusBadRequest, "spintax processing is disabled in settings")
	}

	// Process the spintax
	processed := spintax.Process(req.Text)

	return c.JSON(http.StatusOK, okResp{Data: SpintaxResponse{
		Original:  req.Text,
		Processed: processed,
	}})
}

// PreviewSpintax generates multiple preview variations of spintax text.
func (a *App) PreviewSpintax(c echo.Context) error {
	var req struct {
		Text  string `json:"text"`
		Count int    `json:"count"` // Number of variations to generate (default 5)
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Text == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "text is required")
	}

	// Check if spintax is enabled
	if !a.cfg.SpintaxEnabled {
		return echo.NewHTTPError(http.StatusBadRequest, "spintax processing is disabled in settings")
	}

	// Default to 5 variations if not specified
	count := req.Count
	if count <= 0 {
		count = 5
	}
	if count > 20 {
		count = 20 // Cap at 20 variations
	}

	// Generate variations
	variations := make([]string, count)
	for i := 0; i < count; i++ {
		variations[i] = spintax.Process(req.Text)
	}

	return c.JSON(http.StatusOK, okResp{Data: struct {
		Original   string   `json:"original"`
		Variations []string `json:"variations"`
		HasSpintax bool     `json:"has_spintax"`
	}{
		Original:   req.Text,
		Variations: variations,
		HasSpintax: spintax.HasSpintax(req.Text),
	}})
}

// GenerateSpintax uses Claude AI to generate spintax variations for text.
func (a *App) GenerateSpintax(c echo.Context) error {
	var req GenerateSpintaxRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Text == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "text is required")
	}

	// Check if AI spintax is enabled
	if !a.cfg.SpintaxEnabled {
		return echo.NewHTTPError(http.StatusBadRequest, "spintax is disabled in settings")
	}
	if !a.cfg.SpintaxAIEnabled {
		return echo.NewHTTPError(http.StatusBadRequest, "AI spintax generation is disabled in settings")
	}
	if a.cfg.SpintaxAIAPIKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Claude API key is not configured")
	}

	// Create AI client
	aiClient := spintax.NewAIClient(a.cfg.SpintaxAIAPIKey)

	// Use variation level from request or fall back to settings
	variationLevel := req.VariationLevel
	if variationLevel <= 0 {
		variationLevel = a.cfg.SpintaxVariationLevel
	}
	if variationLevel <= 0 {
		variationLevel = 3 // Default
	}

	// Set up options
	opts := &spintax.GenerateSpintaxOptions{
		VariationLevel: variationLevel,
		PreserveHTML:   req.PreserveHTML,
		PreserveLinks:  req.PreserveLinks,
		OnlyPhrases:    req.OnlyPhrases,
	}

	// Generate spintax with a timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()

	result, err := aiClient.GenerateSpintax(ctx, req.Text, opts)
	if err != nil {
		a.log.Printf("error generating spintax with AI: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate spintax: "+err.Error())
	}

	return c.JSON(http.StatusOK, okResp{Data: struct {
		Original  string `json:"original"`
		Spintax   string `json:"spintax"`
		HasChange bool   `json:"has_change"`
	}{
		Original:  req.Text,
		Spintax:   result,
		HasChange: result != req.Text,
	}})
}

// ValidateSpintax validates spintax syntax and returns information about it.
func (a *App) ValidateSpintax(c echo.Context) error {
	var req ProcessSpintaxRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Text == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "text is required")
	}

	hasSpintax := spintax.HasSpintax(req.Text)
	isValid, err := spintax.Validate(req.Text)

	response := struct {
		HasSpintax bool   `json:"has_spintax"`
		IsValid    bool   `json:"is_valid"`
		Error      string `json:"error,omitempty"`
	}{
		HasSpintax: hasSpintax,
		IsValid:    isValid,
	}

	if err != nil {
		response.Error = err.Error()
	}

	return c.JSON(http.StatusOK, okResp{Data: response})
}
