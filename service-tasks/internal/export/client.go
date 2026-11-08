package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const responseHeaderTimeout = 90 * time.Second

// NewHTTPClient has no Client.Timeout: that deadline would also cover the reading of the body,
// which lasts as long as the stream.
func NewHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = responseHeaderTimeout
	return &http.Client{Transport: transport}
}

// OpenStream returns the response body once the producer has answered 200. The caller closes it.
func OpenStream(ctx context.Context, client *http.Client, producerURL, warehouseID string) (io.ReadCloser, error) {
	payload, err := json.Marshal(map[string]string{"warehouseId": warehouseID})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	url := strings.TrimRight(producerURL, "/") + "/movements/export"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call producer: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("producer answered %s: %s", resp.Status, bytes.TrimSpace(excerpt))
	}
	return resp.Body, nil
}
