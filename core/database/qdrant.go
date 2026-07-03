package database

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

// QdrantDriver implements VectorDriver for Qdrant.
type QdrantDriver struct {
	host   string
	port   int
	client *http.Client
	name   string
}

// NewQdrantDriver creates a new Qdrant driver.
func NewQdrantDriver(name, host string, port int) *QdrantDriver {
	return &QdrantDriver{
		name:   name,
		host:   host,
		port:   port,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns the driver name.
func (d *QdrantDriver) Name() string {
	return d.name
}

// Connect verifies the Qdrant connection.
func (d *QdrantDriver) Connect(ctx context.Context) error {
	return d.Ping(ctx)
}

func (d *QdrantDriver) baseURL() string {
	return fmt.Sprintf("http://%s:%d", d.host, d.port)
}

// Search performs a semantic search.
// The query string is used as a payload filter match (text search).
// For true vector similarity, use SearchWithVector.
func (d *QdrantDriver) Search(ctx context.Context, collection string, query string, topK int) ([]SearchResult, error) {
	// Try the search endpoint first (works if collection has a payload index)
	url := fmt.Sprintf("%s/collections/%s/points/search", d.baseURL(), collection)

	// Build a must-match filter on the payload text fields
	filter := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"match": map[string]interface{}{
					"any": []string{query},
				},
			},
		},
	}

	body := map[string]interface{}{
		"limit":        uint32(topK),
		"with_payload": true,
		"filter":       filter,
	}
	jsonData, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		// Fallback to scroll if search endpoint fails
		return d.searchFallback(ctx, collection, query, topK)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		// Fallback to scroll if search returns error
		return d.searchFallback(ctx, collection, query, topK)
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var searchResults []SearchResult
	for _, point := range result.Result {
		searchResults = append(searchResults, SearchResult{
			ID:       point.ID,
			Score:    point.Score,
			Metadata: point.Payload,
		})
	}

	return searchResults, nil
}

// searchFallback uses scroll API when search endpoint is unavailable.
// Returns results without real similarity scoring.
func (d *QdrantDriver) searchFallback(ctx context.Context, collection string, query string, topK int) ([]SearchResult, error) {
	url := fmt.Sprintf("%s/collections/%s/points/scroll", d.baseURL(), collection)

	body := map[string]interface{}{
		"limit":        uint32(topK),
		"with_payload": true,
	}
	jsonData, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scroll search failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("scroll returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Result struct {
			Points []struct {
				ID      string                 `json:"id"`
				Payload map[string]interface{} `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var searchResults []SearchResult
	for _, point := range result.Result.Points {
		// Check if payload contains the query text (simple text matching)
		if d.payloadMatches(point.Payload, query) {
			searchResults = append(searchResults, SearchResult{
				ID:       point.ID,
				Score:    0.5, // Partial score for text match without vector similarity
				Metadata: point.Payload,
			})
		}
	}

	return searchResults, nil
}

// payloadMatches checks if any payload value contains the query text.
func (d *QdrantDriver) payloadMatches(payload map[string]interface{}, query string) bool {
	queryLower := strings.ToLower(query)
	for _, v := range payload {
		if s, ok := v.(string); ok {
			if strings.Contains(strings.ToLower(s), queryLower) {
				return true
			}
		}
	}
	return false
}

// SearchWithVector performs true vector similarity search.
// The caller must provide the embedding vector.
func (d *QdrantDriver) SearchWithVector(ctx context.Context, collection string, vector []float32, topK int) ([]SearchResult, error) {
	url := fmt.Sprintf("%s/collections/%s/points/search", d.baseURL(), collection)

	body := map[string]interface{}{
		"vector":       vector,
		"limit":        uint32(topK),
		"with_payload": true,
	}
	jsonData, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vector search returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var searchResults []SearchResult
	for _, point := range result.Result {
		searchResults = append(searchResults, SearchResult{
			ID:       point.ID,
			Score:    point.Score,
			Metadata: point.Payload,
		})
	}

	return searchResults, nil
}

// Insert adds a vector with metadata.
func (d *QdrantDriver) Insert(ctx context.Context, collection string, id string, metadata map[string]interface{}) error {
	url := fmt.Sprintf("%s/collections/%s/points", d.baseURL(), collection)

	body := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":      id,
				"payload": metadata,
			},
		},
	}
	jsonData, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("insert returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Delete removes a vector by ID.
func (d *QdrantDriver) Delete(ctx context.Context, collection string, id string) error {
	url := fmt.Sprintf("%s/collections/%s/points/delete", d.baseURL(), collection)

	body := map[string]interface{}{
		"points": []string{id},
	}
	jsonData, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Collections lists all collections.
func (d *QdrantDriver) Collections(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/collections", d.baseURL())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing collections: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Result struct {
			Collections []struct {
				Name string `json:"name"`
			} `json:"collections"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var names []string
	for _, c := range result.Result.Collections {
		names = append(names, c.Name)
	}

	return names, nil
}

// GetSchema returns collection info.
func (d *QdrantDriver) GetSchema(ctx context.Context) (string, error) {
	collections, err := d.Collections(ctx)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("Qdrant Vector Database\n\nCollections:\n")

	for _, name := range collections {
		url := fmt.Sprintf("%s/collections/%s", d.baseURL(), name)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- %s (error creating request)\n", name))
			continue
		}

		resp, err := d.client.Do(req)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- %s (error: %v)\n", name, err))
			continue
		}
		defer resp.Body.Close() //nolint:errcheck

		var info struct {
			Result struct {
				Vectors struct {
					Size     int    `json:"size"`
					Distance string `json:"distance"`
				} `json:"vectors"`
				PointsCount int64 `json:"points_count"`
			} `json:"result"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			sb.WriteString(fmt.Sprintf("- %s (error decoding info)\n", name))
			continue
		}

		sb.WriteString(fmt.Sprintf("- %s (vectors: %d dims, distance: %s, points: %d)\n",
			name, info.Result.Vectors.Size, info.Result.Vectors.Distance, info.Result.PointsCount))
	}

	return sb.String(), nil
}

// Ping checks the connection.
func (d *QdrantDriver) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/healthz", d.baseURL())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %d", resp.StatusCode)
	}
	return nil
}

// Close is a no-op for HTTP-based drivers.
func (d *QdrantDriver) Close() error {
	return nil
}
