package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/agentregistry-dev/agentregistry/pkg/models"
	"github.com/agentregistry-dev/agentregistry/pkg/registry/database"
	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

// BuildServerEmbeddingPayload converts a server document into the canonical text payload
// used for semantic embeddings. The payload deliberately combines all metadata that
// describes the resource so checksum comparisons stay stable across systems.
func BuildServerEmbeddingPayload(server *apiv0.ServerJSON) string {
	if server == nil {
		return ""
	}

	var parts []string
	appendIf(&parts, server.Name, server.Title, server.Description, server.Version, server.WebsiteURL)
	appendJSON(&parts, server.Repository)
	appendJSONArray(&parts, server.Packages)
	appendJSONArray(&parts, server.Remotes)

	if server.Meta != nil && server.Meta.PublisherProvided != nil {
		appendJSON(&parts, server.Meta.PublisherProvided)
	}

	return strings.Join(parts, "\n")
}

// BuildAgentEmbeddingPayload mirrors BuildServerEmbeddingPayload but for AgentJSON entries.
// When a2aCard is non-nil, skill descriptions from the A2A Agent Card are appended
// to the embedding payload, enabling semantic discovery by agent capability.
func BuildAgentEmbeddingPayload(agent *models.AgentJSON, a2aCard json.RawMessage) string {
	if agent == nil {
		return ""
	}

	var parts []string
	appendIf(&parts,
		agent.Name,
		agent.Title,
		agent.Description,
		agent.Version,
		agent.WebsiteURL,
		agent.Language,
		agent.Framework,
		agent.ModelProvider,
		agent.ModelName,
		agent.Image,
	)
	appendJSONArray(&parts, agent.McpServers)
	appendJSON(&parts, agent.Repository)
	appendJSONArray(&parts, agent.Packages)
	appendJSONArray(&parts, agent.Remotes)
	appendA2ACardPayload(&parts, a2aCard)

	return strings.Join(parts, "\n")
}

// appendA2ACardPayload extracts semantic content from an A2A Agent Card for embedding.
// Pulls skill names, descriptions, and tags to enable capability-based discovery.
func appendA2ACardPayload(parts *[]string, cardRaw json.RawMessage) {
	if len(cardRaw) == 0 {
		return
	}
	var card map[string]any
	if err := json.Unmarshal(cardRaw, &card); err != nil {
		slog.Warn("failed to unmarshal a2a card for embedding payload", "error", err)
		return
	}
	appendIf(parts, getString(card, "name"), getString(card, "description"))
	skills, ok := card["skills"].([]any)
	if !ok {
		return
	}
	for _, s := range skills {
		skill, ok := s.(map[string]any)
		if !ok {
			continue
		}
		appendIf(parts, getString(skill, "name"), getString(skill, "description"))
		if tags, ok := skill["tags"].([]any); ok {
			for _, tag := range tags {
				if t, ok := tag.(string); ok {
					appendIf(parts, t)
				}
			}
		}
	}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// PayloadChecksum returns the deterministic checksum for an embedding payload.
func PayloadChecksum(payload string) string {
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// GenerateSemanticEmbedding transforms the provided payload into a SemanticEmbedding
// by invoking the configured provider. The payload must be non-empty.
// When expectedDimensions > 0, the provider output is validated against it.
func GenerateSemanticEmbedding(ctx context.Context, provider Provider, payload string, expectedDimensions int) (*database.SemanticEmbedding, error) {
	if provider == nil {
		return nil, errors.New("embedding provider is not configured")
	}
	if strings.TrimSpace(payload) == "" {
		return nil, errors.New("embedding payload is empty")
	}

	result, err := provider.Generate(ctx, Payload{Text: payload})
	if err != nil {
		return nil, err
	}

	dims := result.Dimensions
	if dims == 0 {
		dims = len(result.Vector)
	}
	if expectedDimensions > 0 && dims != expectedDimensions {
		return nil, fmt.Errorf("embedding dimensions mismatch: expected %d, got %d", expectedDimensions, dims)
	}

	generated := result.GeneratedAt
	if generated.IsZero() {
		generated = time.Now().UTC()
	}

	return &database.SemanticEmbedding{
		Vector:     result.Vector,
		Provider:   result.Provider,
		Model:      result.Model,
		Dimensions: dims,
		Checksum:   PayloadChecksum(payload),
		Generated:  generated,
	}, nil
}

func appendIf(parts *[]string, values ...string) {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			*parts = append(*parts, v)
		}
	}
}

func appendJSON(parts *[]string, value any) {
	if value == nil {
		return
	}
	if data, err := json.Marshal(value); err == nil && len(data) > 0 {
		*parts = append(*parts, string(data))
	}
}

func appendJSONArray(parts *[]string, value any) {
	if value == nil {
		return
	}
	appendJSON(parts, value)
}
