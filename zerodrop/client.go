// Copyright 2026 tyranitar-mega
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zerodrop

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultBaseURL      = "https://zerodrop.dev"
	FreeDomain          = "zerodrop-sandbox.online"
	DefaultTimeout      = 10 * time.Second
	DefaultPollInterval = 2 * time.Second
)

var adjectives = []string{
	"swift", "dark", "cold", "null", "void",
	"zero", "dead", "raw", "base", "core",
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL: DefaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) GenerateInbox() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	id := randomString(7)
	return fmt.Sprintf("%s-%s@%s", adj, id, FreeDomain)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type apiEmail struct {
	ID         string  `json:"id"`
	From       string  `json:"from"`
	To         string  `json:"to"`
	Subject    string  `json:"subject"`
	Raw        string  `json:"raw"`
	ReceivedAt string  `json:"receivedAt"`
	OTP        *string `json:"otp"`
	MagicLink  *string `json:"magicLink"`
}

type apiResponse struct {
	Emails []apiEmail `json:"emails"`
	Count  int        `json:"count"`
}

func (c *Client) FetchLatest(ctx context.Context, inbox string) (*Email, error) {
	inboxName := strings.Split(inbox, "@")[0]
	inboxName = strings.ToLower(inboxName)

	url := fmt.Sprintf("%s/api/inbox/%s?source=sdk", c.baseURL, inboxName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &NetworkError{Message: fmt.Sprintf("create request: %v", err)}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &NetworkError{Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &AuthError{}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &NetworkError{Message: fmt.Sprintf("API returned %d", resp.StatusCode)}
	}

	var data apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, &NetworkError{Message: fmt.Sprintf("parse response: %v", err)}
	}

	if len(data.Emails) == 0 {
		return nil, nil
	}

	latest := data.Emails[0]
	return parseEmail(&latest), nil
}

func (c *Client) WaitForLatest(ctx context.Context, inbox string, opts *WaitOptions) (*Email, error) {
	if opts == nil {
		opts = &WaitOptions{}
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	pollInterval := opts.PollInterval
	if pollInterval == 0 {
		pollInterval = DefaultPollInterval
	}
	useSSE := opts.SSE == nil || *opts.SSE

	if useSSE {
		email, err := c.waitForLatestSSE(ctx, inbox, timeout)
		if err == nil && email != nil {
			return email, nil
		}
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		email, err := c.FetchLatest(ctx, inbox)
		if err != nil {
			if _, ok := err.(*AuthError); ok {
				return nil, err
			}
		}
		if email != nil {
			return email, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return nil, &TimeoutError{Inbox: inbox, Timeout: timeout}
}

func (c *Client) waitForLatestSSE(ctx context.Context, inbox string, timeout time.Duration) (*Email, error) {
	inboxName := strings.Split(inbox, "@")[0]
	inboxName = strings.ToLower(inboxName)

	url := fmt.Sprintf("%s/api/inbox/%s/stream", c.baseURL, inboxName)

	ctx, cancel := context.WithTimeout(ctx, timeout+time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SSE unavailable, falling back to polling")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SSE unavailable, falling back to polling")
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, fmt.Errorf("SSE unavailable, falling back to polling")
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "__timeout__" {
				return nil, nil
			}
			if data != "" {
				email := parseRawEmail(data)
				if email != nil {
					return email, nil
				}
				return nil, nil
			}
		}
	}
}

func parseEmail(e *apiEmail) *Email {
	receivedAt, _ := time.Parse(time.RFC3339, e.ReceivedAt)
	return &Email{
		ID:         e.ID,
		From:       e.From,
		To:         e.To,
		Subject:    e.Subject,
		Body:       extractBody(e.Raw),
		RawBody:    e.Raw,
		ReceivedAt: receivedAt,
		OTP:        e.OTP,
		MagicLink:  e.MagicLink,
	}
}

func parseRawEmail(raw string) *Email {
	var parsed interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}

	if arr, ok := parsed.([]interface{}); ok && len(arr) > 0 {
		if s, ok := arr[0].(string); ok {
			return parseRawEmail(s)
		}
		return nil
	}

	m, ok := parsed.(map[string]interface{})
	if !ok {
		return nil
	}

	email := &apiEmail{}
	if v, ok := m["id"].(string); ok {
		email.ID = v
	}
	if v, ok := m["from"].(string); ok {
		email.From = v
	}
	if v, ok := m["to"].(string); ok {
		email.To = v
	}
	if v, ok := m["subject"].(string); ok {
		email.Subject = v
	}
	if v, ok := m["raw"].(string); ok {
		email.Raw = v
	}
	if v, ok := m["receivedAt"].(string); ok {
		email.ReceivedAt = v
	}
	if v, ok := m["otp"]; ok && v != nil {
		s := fmt.Sprintf("%v", v)
		email.OTP = &s
	}
	if v, ok := m["magicLink"]; ok && v != nil {
		s := fmt.Sprintf("%v", v)
		email.MagicLink = &s
	}

	return parseEmail(email)
}

func extractBody(raw string) string {
	if raw == "" {
		return ""
	}

	if idx := strings.Index(raw, "Content-Type: text/plain"); idx != -1 {
		rest := raw[idx:]
		if headerEnd := strings.Index(rest, "\r\n\r\n"); headerEnd != -1 {
			body := rest[headerEnd+4:]
			if boundaryIdx := strings.Index(body, "\r\n--"); boundaryIdx != -1 {
				return strings.TrimSpace(body[:boundaryIdx])
			}
			if boundaryIdx := strings.Index(body, "\r\n\r\n--"); boundaryIdx != -1 {
				return strings.TrimSpace(body[:boundaryIdx])
			}
		}
	}

	lines := strings.Split(raw, "\r\n")
	var bodyStart int
	for i, l := range lines {
		if l == "" {
			bodyStart = i + 1
			break
		}
	}
	body := strings.Join(lines[bodyStart:], "\n")
	body = strings.TrimSpace(body)
	if len(body) > 5000 {
		body = body[:5000]
	}
	return body
}
