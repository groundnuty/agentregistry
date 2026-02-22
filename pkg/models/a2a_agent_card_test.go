package models_test

import (
	"encoding/json"
	"testing"

	"github.com/agentregistry-dev/agentregistry/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validCardJSON() json.RawMessage {
	return json.RawMessage(`{
		"name": "test-agent",
		"description": "A test agent",
		"url": "https://example.com/agent",
		"version": "1.0.0",
		"capabilities": {},
		"skills": [
			{
				"id": "skill-1",
				"name": "Skill One",
				"description": "First skill",
				"tags": ["test"]
			}
		],
		"defaultInputModes": ["text/plain"],
		"defaultOutputModes": ["text/plain"]
	}`)
}

func TestValidateA2AAgentCard(t *testing.T) {
	tests := []struct {
		name    string
		card    json.RawMessage
		wantErr string
	}{
		{
			name: "valid card with camelCase",
			card: validCardJSON(),
		},
		{
			name: "valid card with snake_case modes",
			card: json.RawMessage(`{
				"name": "test-agent",
				"description": "A test agent",
				"url": "https://example.com/agent",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [{"id": "s1", "name": "Skill", "description": "desc"}],
				"default_input_modes": ["text/plain"],
				"default_output_modes": ["text/plain"]
			}`),
		},
		{
			name:    "invalid JSON",
			card:    json.RawMessage(`not json`),
			wantErr: "invalid JSON",
		},
		{
			name: "missing name",
			card: json.RawMessage(`{
				"description": "desc",
				"url": "https://example.com",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [{"id": "s1", "name": "S", "description": "d"}],
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `required field "name" is missing`,
		},
		{
			name: "missing description",
			card: json.RawMessage(`{
				"name": "test",
				"url": "https://example.com",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [{"id": "s1", "name": "S", "description": "d"}],
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `required field "description" is missing`,
		},
		{
			name: "missing url",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [{"id": "s1", "name": "S", "description": "d"}],
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `required field "url" is missing`,
		},
		{
			name: "missing version",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"url": "https://example.com",
				"capabilities": {},
				"skills": [{"id": "s1", "name": "S", "description": "d"}],
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `required field "version" is missing`,
		},
		{
			name: "missing capabilities",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"url": "https://example.com",
				"version": "1.0.0",
				"skills": [{"id": "s1", "name": "S", "description": "d"}],
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `required field "capabilities" is missing`,
		},
		{
			name: "missing skills",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"url": "https://example.com",
				"version": "1.0.0",
				"capabilities": {},
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `required field "skills" is missing`,
		},
		{
			name: "empty skills array",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"url": "https://example.com",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [],
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: "skills array must not be empty",
		},
		{
			name: "missing defaultInputModes (neither convention)",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"url": "https://example.com",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [{"id": "s1", "name": "S", "description": "d"}],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `required field "defaultInputModes" is missing`,
		},
		{
			name: "missing defaultOutputModes (neither convention)",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"url": "https://example.com",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [{"id": "s1", "name": "S", "description": "d"}],
				"defaultInputModes": ["text/plain"]
			}`),
			wantErr: `required field "defaultOutputModes" is missing`,
		},
		{
			name: "skill missing id",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"url": "https://example.com",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [{"name": "S", "description": "d"}],
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `skill[0] missing required field "id"`,
		},
		{
			name: "skill missing name",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"url": "https://example.com",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [{"id": "s1", "description": "d"}],
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `skill[0] missing required field "name"`,
		},
		{
			name: "skill missing description",
			card: json.RawMessage(`{
				"name": "test",
				"description": "desc",
				"url": "https://example.com",
				"version": "1.0.0",
				"capabilities": {},
				"skills": [{"id": "s1", "name": "S"}],
				"defaultInputModes": ["text/plain"],
				"defaultOutputModes": ["text/plain"]
			}`),
			wantErr: `skill[0] missing required field "description"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateA2AAgentCard(tt.card)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
