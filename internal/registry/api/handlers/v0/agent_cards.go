package v0

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/agentregistry-dev/agentregistry/internal/registry/service"
	"github.com/agentregistry-dev/agentregistry/pkg/registry/auth"
	"github.com/agentregistry-dev/agentregistry/pkg/registry/database"
)

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
		_, _ = w.Write(card)
	})
}
