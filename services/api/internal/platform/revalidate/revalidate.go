// Package revalidate notifies the web application when catalog content changes, so Next.js can
// drop cached pages that depend on it.
package revalidate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const secretHeader = "X-Revalidate-Secret"

// CatalogTag is the Next.js cache tag that covers every public catalog read.
const CatalogTag = "catalog"

// Notifier posts tag revalidation requests to the web application. A nil or unconfigured notifier
// does nothing, which keeps local development free of web dependencies.
type Notifier struct {
	url    string
	secret string
	client *http.Client
}

// NewNotifier creates a notifier. An empty URL disables notifications.
func NewNotifier(url, secret string) *Notifier {
	return &Notifier{
		url:    url,
		secret: secret,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Revalidate asks the web application to revalidate the given tags.
func (n *Notifier) Revalidate(ctx context.Context, tags ...string) error {
	if n == nil || n.url == "" {
		return nil
	}
	body, err := json.Marshal(map[string][]string{"tags": tags})
	if err != nil {
		return fmt.Errorf("encode revalidation request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build revalidation request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(secretHeader, n.secret)

	response, err := n.client.Do(request)
	if err != nil {
		return fmt.Errorf("send revalidation request: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("revalidation rejected with status %d", response.StatusCode)
	}
	return nil
}
