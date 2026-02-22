package v0_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v0 "github.com/agentregistry-dev/agentregistry/internal/registry/api/handlers/v0"
	fakereg "github.com/agentregistry-dev/agentregistry/internal/registry/service/testing"
	"github.com/agentregistry-dev/agentregistry/pkg/models"
	"github.com/agentregistry-dev/agentregistry/pkg/registry/database"
)

func testCardJSON() json.RawMessage {
	return json.RawMessage(`{
		"name": "test-agent",
		"description": "A test agent",
		"url": "https://example.com/agent",
		"version": "1.0.0",
		"capabilities": {},
		"skills": [{"id":"s1","name":"Skill","description":"desc","tags":["test"]}],
		"defaultInputModes": ["text/plain"],
		"defaultOutputModes": ["text/plain"]
	}`)
}

func setupAgentAPI(t *testing.T, registry *fakereg.FakeRegistry) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Test API", "1.0.0"))
	v0.RegisterAgentsEndpoints(api, "/v0", registry)
	v0.RegisterAgentsCreateEndpoint(api, "/v0", registry)
	return mux
}

// validAgentBody returns a minimal valid agent JSON body string.
// Huma validates required AgentManifest fields (framework, image, language,
// modelProvider, modelName), so test bodies must include them.
func validAgentBody(extra string) string {
	base := `"name":"my-agent","version":"1.0.0","description":"test","framework":"adk","image":"test:latest","language":"python","modelProvider":"openai","modelName":"gpt-4o"`
	if extra != "" {
		return `{` + base + `,` + extra + `}`
	}
	return `{` + base + `}`
}

func fakeCreateAgentFn(_ context.Context, req *models.AgentJSON) (*models.AgentResponse, error) {
	return &models.AgentResponse{
		Agent: *req,
		Meta: models.AgentResponseMeta{
			Official: &models.AgentRegistryExtensions{
				Status:   "active",
				IsLatest: true,
			},
		},
	}, nil
}

// -- Card-through-agent endpoint tests --

func TestCreateAgentWithCard(t *testing.T) {
	card := testCardJSON()
	registry := fakereg.NewFakeRegistry()
	registry.CreateAgentFn = fakeCreateAgentFn
	mux := setupAgentAPI(t, registry)

	body := validAgentBody(`"card":` + string(card))
	req := httptest.NewRequest(http.MethodPost, "/v0/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp models.AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "my-agent", resp.Agent.Name)
	assert.NotNil(t, resp.Agent.Card, "card should be present in response")
	assert.JSONEq(t, string(card), string(resp.Agent.Card))
}

func TestCreateAgentWithoutCard(t *testing.T) {
	registry := fakereg.NewFakeRegistry()
	registry.CreateAgentFn = fakeCreateAgentFn
	mux := setupAgentAPI(t, registry)

	body := validAgentBody("")
	req := httptest.NewRequest(http.MethodPost, "/v0/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp models.AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "my-agent", resp.Agent.Name)
	assert.Nil(t, resp.Agent.Card, "card should be absent when not provided")
}

func TestCreateAgentWithInvalidCard(t *testing.T) {
	registry := fakereg.NewFakeRegistry()
	registry.CreateAgentFn = func(_ context.Context, _ *models.AgentJSON) (*models.AgentResponse, error) {
		return nil, database.ErrInvalidInput
	}
	mux := setupAgentAPI(t, registry)

	body := validAgentBody(`"card":{"bad":"card"}`)
	req := httptest.NewRequest(http.MethodPost, "/v0/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAgentVersionReturnsCard(t *testing.T) {
	card := testCardJSON()
	registry := fakereg.NewFakeRegistry()
	registry.Agents = []*models.AgentResponse{
		{
			Agent: models.AgentJSON{
				AgentManifest: models.AgentManifest{
					Name:        "my-agent",
					Description: "test agent",
				},
				Version: "1.0.0",
				Card:    card,
			},
			Meta: models.AgentResponseMeta{
				Official: &models.AgentRegistryExtensions{
					Status:   "active",
					IsLatest: true,
				},
			},
		},
	}
	mux := setupAgentAPI(t, registry)

	t.Run("specific version returns card inline", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v0/agents/my-agent/versions/1.0.0", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp models.AgentResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Equal(t, "my-agent", resp.Agent.Name)
		assert.NotNil(t, resp.Agent.Card, "card should be present in GET response")
		assert.JSONEq(t, string(card), string(resp.Agent.Card))
	})

	t.Run("latest version returns card inline", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v0/agents/my-agent/versions/latest", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp models.AgentResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.NotNil(t, resp.Agent.Card, "card should be present for latest version")
		assert.JSONEq(t, string(card), string(resp.Agent.Card))
	})
}

func TestListAgentsWithCardFilter(t *testing.T) {
	card := testCardJSON()

	agentWithCard := &models.AgentResponse{
		Agent: models.AgentJSON{
			AgentManifest: models.AgentManifest{
				Name:        "agent-with-card",
				Description: "has a card",
			},
			Version: "1.0.0",
			Card:    card,
		},
		Meta: models.AgentResponseMeta{
			Official: &models.AgentRegistryExtensions{
				Status:   "active",
				IsLatest: true,
			},
		},
	}
	agentWithoutCard := &models.AgentResponse{
		Agent: models.AgentJSON{
			AgentManifest: models.AgentManifest{
				Name:        "agent-no-card",
				Description: "no card",
			},
			Version: "1.0.0",
		},
		Meta: models.AgentResponseMeta{
			Official: &models.AgentRegistryExtensions{
				Status:   "active",
				IsLatest: true,
			},
		},
	}

	registry := fakereg.NewFakeRegistry()
	registry.Agents = []*models.AgentResponse{agentWithCard, agentWithoutCard}
	// Seed the card map so the FakeRegistry's HasCard filter works
	registry.AgentCards["agent-with-card:1.0.0"] = card

	mux := setupAgentAPI(t, registry)

	t.Run("has_card=true returns only agents with cards", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v0/agents?has_card=true", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp models.AgentListResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Len(t, resp.Agents, 1)
		assert.Equal(t, "agent-with-card", resp.Agents[0].Agent.Name)
	})

	t.Run("no filter returns all agents", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v0/agents", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp models.AgentListResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Len(t, resp.Agents, 2)
	})
}

// -- .well-known endpoint tests (kept from v1) --

func TestWellKnownAgentCardHandler(t *testing.T) {
	card := testCardJSON()

	t.Run("card found", func(t *testing.T) {
		registry := fakereg.NewFakeRegistry()
		// well-known endpoint calls GetAgentCard with version=""
		registry.AgentCards["my-agent:"] = card
		mux := http.NewServeMux()
		v0.RegisterWellKnownAgentCardHandler(mux, "/v0", registry)

		req := httptest.NewRequest(http.MethodGet, "/v0/a2a/my-agent/.well-known/agent-card.json", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		// The well-known endpoint returns raw card JSON (no envelope)
		assert.JSONEq(t, string(card), w.Body.String())
	})

	t.Run("card not found", func(t *testing.T) {
		registry := fakereg.NewFakeRegistry()
		mux := http.NewServeMux()
		v0.RegisterWellKnownAgentCardHandler(mux, "/v0", registry)

		req := httptest.NewRequest(http.MethodGet, "/v0/a2a/unknown/.well-known/agent-card.json", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("method not allowed", func(t *testing.T) {
		registry := fakereg.NewFakeRegistry()
		mux := http.NewServeMux()
		v0.RegisterWellKnownAgentCardHandler(mux, "/v0", registry)

		req := httptest.NewRequest(http.MethodPost, "/v0/a2a/my-agent/.well-known/agent-card.json", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}
