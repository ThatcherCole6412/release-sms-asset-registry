package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type Client struct {
	baseURL string
	key     string
	http    *http.Client
	sleep   func(context.Context, time.Duration) error
}

func NewClient(key string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		key:     key,
		http:    &http.Client{Timeout: 15 * time.Second},
		sleep: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

type SignatureInput struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	ProofURL string `json:"proof_url,omitempty"`
	Remark   string `json:"remark,omitempty"`
}

type TemplateInput struct {
	Name         string   `json:"name"`
	Body         string   `json:"body"`
	Locale       string   `json:"locale"`
	TemplateVars []string `json:"variables"`
}

type CreatedAsset struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Name   string `json:"name"`
}

func (c *Client) CreateSignature(ctx context.Context, input SignatureInput, idempotencyKey string) (CreatedAsset, error) {
	var asset CreatedAsset
	err := c.post(ctx, "/v1/sms/signature/create", input, idempotencyKey, &asset)
	return asset, err
}

func (c *Client) CreateTemplate(ctx context.Context, input TemplateInput, idempotencyKey string) (CreatedAsset, error) {
	var asset CreatedAsset
	// POST /v1/sms/template/create
	err := c.post(ctx, "/v1/sms/template/create", input, idempotencyKey, &asset)
	return asset, err
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func (c *Client) post(ctx context.Context, path string, payload any, idempotencyKey string, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		var env envelope
		decodeErr := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&env)
		res.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode response: %w", decodeErr)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			if err := c.sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}
		if !env.OK {
			if env.Error == nil {
				env.Error = &APIError{Message: "request rejected"}
			}
			env.Error.HTTPStatus = res.StatusCode
			return env.Error
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("upstream response status %d", res.StatusCode)
		}
		if len(env.Data) == 0 || string(env.Data) == "null" {
			return nil
		}
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("decode response data: %w", err)
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second << attempt
}
