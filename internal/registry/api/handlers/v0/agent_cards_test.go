package v0_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func setupAgentCardAPI(t *testing.T, registry *fakereg.FakeRegistry) (*http.ServeMux, huma.API) {
	t.Helper()
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Test API", "1.0.0"))
	v0.RegisterAgentCardEndpoints(api, "/v0", registry)
	return mux, api
}

func TestUpsertAgentCard(t *testing.T) {
	tests := []struct {
		name           string
		agentName      string
		version        string
		body           string
		setupFn        func(f *fakereg.FakeRegistry)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful upsert",
			agentName:      "my-agent",
			version:        "1.0.0",
			body:           `{"card":` + string(testCardJSON()) + `}`,
			expectedStatus: http.StatusOK,
			expectedBody:   "Agent card saved successfully",
		},
		{
			name:           "empty card field",
			agentName:      "my-agent",
			version:        "1.0.0",
			body:           `{"card":null}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "agent not found",
			agentName: "missing-agent",
			version:   "1.0.0",
			body:      `{"card":` + string(testCardJSON()) + `}`,
			setupFn: func(f *fakereg.FakeRegistry) {
				f.UpsertAgentCardFn = func(_ context.Context, _, _ string, _ json.RawMessage) error {
					return database.ErrNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := fakereg.NewFakeRegistry()
			if tt.setupFn != nil {
				tt.setupFn(registry)
			}
			mux, _ := setupAgentCardAPI(t, registry)

			encName := url.PathEscape(tt.agentName)
			encVersion := url.PathEscape(tt.version)
			path := "/v0/agents/" + encName + "/versions/" + encVersion + "/card"
			req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestGetAgentCardVersion(t *testing.T) {
	card := testCardJSON()

	tests := []struct {
		name           string
		agentName      string
		version        string
		setupCard      bool
		expectedStatus int
	}{
		{
			name:           "card found",
			agentName:      "my-agent",
			version:        "1.0.0",
			setupCard:      true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "card not found",
			agentName:      "my-agent",
			version:        "2.0.0",
			setupCard:      false,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "latest version alias",
			agentName:      "my-agent",
			version:        "latest",
			setupCard:      false,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := fakereg.NewFakeRegistry()
			if tt.setupCard {
				registry.AgentCards[tt.agentName+":"+tt.version] = card
			}
			mux, _ := setupAgentCardAPI(t, registry)

			encName := url.PathEscape(tt.agentName)
			encVersion := url.PathEscape(tt.version)
			path := "/v0/agents/" + encName + "/versions/" + encVersion + "/card"
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp models.AgentCardResponse
				require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
				assert.Equal(t, tt.agentName, resp.AgentName)
				assert.JSONEq(t, string(card), string(resp.Card))
			}
		})
	}
}

func TestGetAgentCardLatest(t *testing.T) {
	card := testCardJSON()

	t.Run("card found", func(t *testing.T) {
		registry := fakereg.NewFakeRegistry()
		// FakeRegistry's GetAgentCard looks up by "name:version",
		// and the latest endpoint passes version="" to the service
		registry.AgentCards["my-agent:"] = card
		mux, _ := setupAgentCardAPI(t, registry)

		req := httptest.NewRequest(http.MethodGet, "/v0/agents/my-agent/card", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp models.AgentCardResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Equal(t, "my-agent", resp.AgentName)
		assert.JSONEq(t, string(card), string(resp.Card))
	})

	t.Run("card not found", func(t *testing.T) {
		registry := fakereg.NewFakeRegistry()
		mux, _ := setupAgentCardAPI(t, registry)

		req := httptest.NewRequest(http.MethodGet, "/v0/agents/unknown/card", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDeleteAgentCard(t *testing.T) {
	card := testCardJSON()

	t.Run("successful delete", func(t *testing.T) {
		registry := fakereg.NewFakeRegistry()
		registry.AgentCards["my-agent:1.0.0"] = card
		mux, _ := setupAgentCardAPI(t, registry)

		req := httptest.NewRequest(http.MethodDelete, "/v0/agents/my-agent/versions/1.0.0/card", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Agent card deleted successfully")

		// Verify card was removed
		_, exists := registry.AgentCards["my-agent:1.0.0"]
		assert.False(t, exists)
	})

	t.Run("card not found", func(t *testing.T) {
		registry := fakereg.NewFakeRegistry()
		mux, _ := setupAgentCardAPI(t, registry)

		req := httptest.NewRequest(http.MethodDelete, "/v0/agents/my-agent/versions/1.0.0/card", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

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
