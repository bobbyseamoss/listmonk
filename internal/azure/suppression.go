package azure

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// SuppressionConfig holds Azure suppression list configuration
type SuppressionConfig struct {
	Enabled          bool   `json:"enabled"`
	SubscriptionID   string `json:"subscription_id"`
	ResourceGroup    string `json:"resource_group"`
	EmailServiceName string `json:"email_service_name"`
	DomainName       string `json:"domain_name"`
	ListName         string `json:"list_name"` // e.g., "donotreply" matches MailFrom username
}

// SuppressionClient manages Azure Communication Services suppression lists
type SuppressionClient struct {
	config     SuppressionConfig
	httpClient *http.Client
	log        *log.Logger
}

// NewSuppressionClient creates a new Azure suppression list client
func NewSuppressionClient(config SuppressionConfig, l *log.Logger) *SuppressionClient {
	return &SuppressionClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: l,
	}
}

// AddToSuppressionList adds an email address to the Azure domain suppression list
func (c *SuppressionClient) AddToSuppressionList(email string) error {
	if !c.config.Enabled {
		return nil
	}

	// Validate required config
	if c.config.SubscriptionID == "" || c.config.ResourceGroup == "" ||
		c.config.EmailServiceName == "" || c.config.DomainName == "" || c.config.ListName == "" {
		c.log.Printf("Azure suppression list not fully configured, skipping")
		return nil
	}

	// Get Azure access token using managed identity or environment credentials
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Azure access token: %w", err)
	}

	// Generate unique address ID from email (MD5 hash for idempotency)
	hash := md5.Sum([]byte(strings.ToLower(email)))
	addressID := hex.EncodeToString(hash[:])

	// Build the API URL
	apiVersion := "2023-06-01-preview"
	url := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Communication/emailServices/%s/domains/%s/suppressionLists/%s/suppressionListAddresses/%s?api-version=%s",
		c.config.SubscriptionID,
		c.config.ResourceGroup,
		c.config.EmailServiceName,
		c.config.DomainName,
		c.config.ListName,
		addressID,
		apiVersion,
	)

	// Build request body
	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"email": email,
			"notes": "Unsubscribed via listmonk",
		},
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create PUT request
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.log.Printf("Added %s to Azure suppression list %s/%s", email, c.config.DomainName, c.config.ListName)
		return nil
	}

	// Read error response
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("Azure API error (status %d): %s", resp.StatusCode, string(respBody))
}

// getAccessToken retrieves an Azure access token using DefaultAzureCredential
// This supports managed identity, environment variables, Azure CLI, etc.
func (c *SuppressionClient) getAccessToken(ctx context.Context) (string, error) {
	// Try DefaultAzureCredential which supports multiple auth methods:
	// 1. Environment variables (AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_CLIENT_SECRET)
	// 2. Managed Identity (when running in Azure)
	// 3. Azure CLI
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Get token for Azure Resource Manager
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	return token.Token, nil
}

// IsEnabled returns whether the suppression list integration is enabled
func (c *SuppressionClient) IsEnabled() bool {
	return c.config.Enabled
}
