package v0

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/agentregistry-dev/agentregistry/internal/registry/service"
	agentmodels "github.com/agentregistry-dev/agentregistry/pkg/models"
	"github.com/agentregistry-dev/agentregistry/pkg/registry/auth"
	"github.com/agentregistry-dev/agentregistry/pkg/registry/database"
	"github.com/agentregistry-dev/agentregistry/pkg/types"
	"github.com/danielgtaylor/huma/v2"
)

// UpsertAgentCardInput represents the input for creating or updating an A2A Agent Card.
type UpsertAgentCardInput struct {
	AgentName string `path:"agentName" json:"agentName" doc:"URL-encoded agent name" example:"com.example%2Fmy-agent"`
	Version   string `path:"version" json:"version" doc:"URL-encoded agent version" example:"1.0.0"`
	Body      struct {
		Card json.RawMessage `json:"card" doc:"A2A v0.3 Agent Card JSON"`
	}
}

// AgentCardVersionInput represents the input for retrieving or deleting a card by version.
type AgentCardVersionInput struct {
	AgentName string `path:"agentName" json:"agentName" doc:"URL-encoded agent name" example:"com.example%2Fmy-agent"`
	Version   string `path:"version" json:"version" doc:"URL-encoded agent version" example:"1.0.0"`
}

// AgentCardLatestInput represents the input for retrieving the latest card.
type AgentCardLatestInput struct {
	AgentName string `path:"agentName" json:"agentName" doc:"URL-encoded agent name" example:"com.example%2Fmy-agent"`
}

// RegisterAgentCardEndpoints registers A2A Agent Card CRUD endpoints.
func RegisterAgentCardEndpoints(api huma.API, pathPrefix string, registry service.RegistryService) {
	tags := []string{"agents"}

	// PUT /agents/{agentName}/versions/{version}/card -- upsert
	huma.Register(api, huma.Operation{
		OperationID: "upsert-agent-card" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodPut,
		Path:        pathPrefix + "/agents/{agentName}/versions/{version}/card",
		Summary:     "Publish or update an A2A Agent Card",
		Description: "Store or replace the A2A Agent Card for a specific agent version. The card JSON is validated against A2A v0.3 required fields.",
		Tags:        tags,
	}, func(ctx context.Context, input *UpsertAgentCardInput) (*types.Response[types.EmptyResponse], error) {
		agentName, err := url.PathUnescape(input.AgentName)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid agent name encoding", err)
		}
		version, err := url.PathUnescape(input.Version)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid version encoding", err)
		}

		if len(input.Body.Card) == 0 || string(input.Body.Card) == "null" {
			return nil, huma.Error400BadRequest("card field is required")
		}

		if err := registry.UpsertAgentCard(ctx, agentName, version, input.Body.Card); err != nil {
			if errors.Is(err, database.ErrNotFound) || errors.Is(err, auth.ErrForbidden) || errors.Is(err, auth.ErrUnauthenticated) {
				return nil, huma.Error404NotFound("Agent not found")
			}
			if errors.Is(err, database.ErrInvalidInput) {
				return nil, huma.Error400BadRequest(err.Error(), err)
			}
			return nil, huma.Error400BadRequest("Failed to upsert agent card", err)
		}

		return &types.Response[types.EmptyResponse]{
			Body: types.EmptyResponse{Message: "Agent card saved successfully"},
		}, nil
	})

	// GET /agents/{agentName}/versions/{version}/card -- get by version
	huma.Register(api, huma.Operation{
		OperationID: "get-agent-card-version" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/agents/{agentName}/versions/{version}/card",
		Summary:     "Get A2A Agent Card for a specific version",
		Description: "Retrieve the A2A Agent Card for a specific agent version. Use 'latest' as the version to get the card for the latest version.",
		Tags:        tags,
	}, func(ctx context.Context, input *AgentCardVersionInput) (*types.Response[agentmodels.AgentCardResponse], error) {
		agentName, err := url.PathUnescape(input.AgentName)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid agent name encoding", err)
		}
		version, err := url.PathUnescape(input.Version)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid version encoding", err)
		}

		// Support "latest" as a version alias
		if version == "latest" {
			version = ""
		}

		card, resolvedVersion, err := registry.GetAgentCard(ctx, agentName, version)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) || errors.Is(err, auth.ErrForbidden) || errors.Is(err, auth.ErrUnauthenticated) {
				return nil, huma.Error404NotFound("Agent card not found")
			}
			return nil, huma.Error500InternalServerError("Failed to get agent card", err)
		}

		return &types.Response[agentmodels.AgentCardResponse]{
			Body: agentmodels.AgentCardResponse{
				AgentName: agentName,
				Version:   resolvedVersion,
				Card:      card,
			},
		}, nil
	})

	// GET /agents/{agentName}/card -- get latest
	huma.Register(api, huma.Operation{
		OperationID: "get-agent-card-latest" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/agents/{agentName}/card",
		Summary:     "Get A2A Agent Card for the latest version",
		Description: "Retrieve the A2A Agent Card for the latest version of an agent.",
		Tags:        tags,
	}, func(ctx context.Context, input *AgentCardLatestInput) (*types.Response[agentmodels.AgentCardResponse], error) {
		agentName, err := url.PathUnescape(input.AgentName)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid agent name encoding", err)
		}

		card, resolvedVersion, err := registry.GetAgentCard(ctx, agentName, "")
		if err != nil {
			if errors.Is(err, database.ErrNotFound) || errors.Is(err, auth.ErrForbidden) || errors.Is(err, auth.ErrUnauthenticated) {
				return nil, huma.Error404NotFound("Agent card not found")
			}
			return nil, huma.Error500InternalServerError("Failed to get agent card", err)
		}

		return &types.Response[agentmodels.AgentCardResponse]{
			Body: agentmodels.AgentCardResponse{
				AgentName: agentName,
				Version:   resolvedVersion,
				Card:      card,
			},
		}, nil
	})

	// DELETE /agents/{agentName}/versions/{version}/card -- remove card
	huma.Register(api, huma.Operation{
		OperationID: "delete-agent-card" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/agents/{agentName}/versions/{version}/card",
		Summary:     "Delete an A2A Agent Card",
		Description: "Remove the A2A Agent Card from a specific agent version.",
		Tags:        tags,
	}, func(ctx context.Context, input *AgentCardVersionInput) (*types.Response[types.EmptyResponse], error) {
		agentName, err := url.PathUnescape(input.AgentName)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid agent name encoding", err)
		}
		version, err := url.PathUnescape(input.Version)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid version encoding", err)
		}

		if err := registry.DeleteAgentCard(ctx, agentName, version); err != nil {
			if errors.Is(err, database.ErrNotFound) || errors.Is(err, auth.ErrForbidden) || errors.Is(err, auth.ErrUnauthenticated) {
				return nil, huma.Error404NotFound("Agent card not found")
			}
			return nil, huma.Error500InternalServerError("Failed to delete agent card", err)
		}

		return &types.Response[types.EmptyResponse]{
			Body: types.EmptyResponse{Message: "Agent card deleted successfully"},
		}, nil
	})
}

// RegisterWellKnownAgentCardHandler registers the A2A discovery endpoint on
// the raw HTTP mux. This avoids potential Huma path-matching issues with
// dots in the URL segment (.well-known).
//
// GET /v0/a2a/{agentName}/.well-known/agent-card.json
func RegisterWellKnownAgentCardHandler(mux *http.ServeMux, pathPrefix string, registry service.RegistryService) {
	pattern := pathPrefix + "/a2a/{agentName}/.well-known/agent-card.json"

	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rawName := r.PathValue("agentName")
		agentName, err := url.PathUnescape(rawName)
		if err != nil {
			http.Error(w, "invalid agent name encoding", http.StatusBadRequest)
			return
		}

		card, _, err := registry.GetAgentCard(r.Context(), agentName, "")
		if err != nil {
			if errors.Is(err, database.ErrNotFound) || errors.Is(err, auth.ErrForbidden) || errors.Is(err, auth.ErrUnauthenticated) {
				http.Error(w, "agent card not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(card)
	})
}
