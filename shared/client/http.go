package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aegis/shared/utils"
	"go.uber.org/zap"
)

// HTTPClient provides HTTP functionality for the service client
type HTTPClient struct {
	client *http.Client
	logger *zap.Logger
}

// NewHTTPClient creates a new HTTP client with timeout
func NewHTTPClient(timeout time.Duration, logger *zap.Logger) (*HTTPClient, error) {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}, nil
}

// Call makes an HTTP request and unmarshals the response
func (c *HTTPClient) Call(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var requestBody []byte
	var err error

	if body != nil {
		requestBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Create request
	var req *http.Request
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(requestBody))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Log request
	c.logger.Debug("making HTTP request",
		zap.String("method", method),
		zap.String("url", url),
		zap.Int("body_size", len(requestBody)))

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			return utils.TimeoutError(fmt.Sprintf("%s %s", method, url), c.client.Timeout)
		}
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response
	c.logger.Debug("HTTP response received",
		zap.String("method", method),
		zap.String("url", url),
		zap.Int("status_code", resp.StatusCode),
		zap.Int("body_size", len(responseBody)))

	// Check status code
	if resp.StatusCode >= 500 {
		return utils.NewServiceError(utils.ErrCodeServiceUnavail, 
			fmt.Sprintf("service unavailable: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
			string(responseBody))
	}

	if resp.StatusCode >= 400 {
		return utils.NewServiceError(utils.ErrCodeInvalidRequest, 
			fmt.Sprintf("request failed: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
			string(responseBody))
	}

	// Unmarshal response if result is provided
	if result != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Close closes the HTTP client
func (c *HTTPClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}