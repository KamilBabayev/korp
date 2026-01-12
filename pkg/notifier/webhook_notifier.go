/*
Copyright 2026 The Korp Authors.

Licensed under the MIT License.
*/

package notifier

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/kamilbabayev/korp/api/v1alpha1"
)

const (
	defaultMethod              = "POST"
	defaultTimeoutSeconds      = 30
	defaultMaxRetries          = 3
	defaultInitialDelaySeconds = 1
)

// WebhookNotifier handles sending webhook notifications
type WebhookNotifier struct {
	config v1alpha1.WebhookConfig
	client *http.Client
	logger logr.Logger
}

// NewWebhookNotifier creates a new webhook notifier with the given configuration
func NewWebhookNotifier(config v1alpha1.WebhookConfig, logger logr.Logger) *WebhookNotifier {
	timeout := defaultTimeoutSeconds
	if config.TimeoutSeconds > 0 {
		timeout = config.TimeoutSeconds
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
	}

	return &WebhookNotifier{
		config: config,
		client: &http.Client{
			Timeout:   time.Duration(timeout) * time.Second,
			Transport: transport,
		},
		logger: logger,
	}
}

// Send sends a webhook notification with the given payload
// Returns error if all retry attempts fail
func (w *WebhookNotifier) Send(ctx context.Context, payload WebhookPayload) error {
	maxRetries := defaultMaxRetries
	if w.config.RetryPolicy != nil && w.config.RetryPolicy.MaxRetries >= 0 {
		maxRetries = w.config.RetryPolicy.MaxRetries
	}

	initialDelay := defaultInitialDelaySeconds
	if w.config.RetryPolicy != nil && w.config.RetryPolicy.InitialDelaySeconds > 0 {
		initialDelay = w.config.RetryPolicy.InitialDelaySeconds
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: initialDelay * 2^(attempt-1)
			delay := time.Duration(initialDelay*(1<<(attempt-1))) * time.Second
			w.logger.Info("Retrying webhook after delay",
				"attempt", attempt,
				"delay", delay.String(),
				"url", w.config.URL)

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		err := w.sendOnce(ctx, payload)
		if err == nil {
			if attempt > 0 {
				w.logger.Info("Webhook succeeded after retry",
					"attempt", attempt,
					"url", w.config.URL)
			}
			return nil
		}

		lastErr = err
		w.logger.Error(err, "Webhook attempt failed",
			"attempt", attempt,
			"url", w.config.URL,
			"maxRetries", maxRetries)
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", maxRetries+1, lastErr)
}

// sendOnce performs a single webhook send attempt
func (w *WebhookNotifier) sendOnce(ctx context.Context, payload WebhookPayload) error {
	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Determine HTTP method
	method := defaultMethod
	if w.config.Method != "" {
		method = w.config.Method
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, w.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default Content-Type header
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers (will override Content-Type if specified)
	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error details
	body, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d, body: %s",
			resp.StatusCode, string(body))
	}

	w.logger.V(1).Info("Webhook sent successfully",
		"url", w.config.URL,
		"status", resp.StatusCode)

	return nil
}
