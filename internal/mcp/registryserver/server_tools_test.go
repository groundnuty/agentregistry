package registryserver

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	servicetesting "github.com/agentregistry-dev/agentregistry/internal/registry/service/testing"
	"github.com/agentregistry-dev/agentregistry/pkg/models"
	"github.com/agentregistry-dev/agentregistry/pkg/registry/database"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"github.com/modelcontextprotocol/registry/pkg/model"
)

func TestServerTools_ListAndReadme(t *testing.T) {
	ctx := context.Background()

	readme := &database.ServerReadme{
		ServerName:  "com.example/echo",
		Version:     "1.0.0",
		Content:     []byte("# Echo"),
		ContentType: "text/markdown",
		SizeBytes:   6,
		SHA256:      []byte{0xaa, 0xbb},
		FetchedAt:   time.Now(),
	}
	reg := servicetesting.NewFakeRegistry()
	reg.Servers = []*apiv0.ServerResponse{
		{
			Server: apiv0.ServerJSON{
				Schema:      model.CurrentSchemaURL,
				Name:        "com.example/echo",
				Description: "Echo server",
				Title:       "Echo",
				Version:     "1.0.0",
			},
		},
	}
	reg.ServerReadme = readme

	server := NewServer(reg)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, serverSession.Wait())
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	// list_servers
	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_servers",
		Arguments: map[string]any{"limit": 10},
	})
	require.NoError(t, err)
	raw, _ := json.Marshal(res.StructuredContent)
	var listOut apiv0.ServerListResponse
	require.NoError(t, json.Unmarshal(raw, &listOut))
	require.Len(t, listOut.Servers, 1)
	assert.Equal(t, "com.example/echo", listOut.Servers[0].Server.Name)

	// get_server_readme
	res, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_server_readme",
		Arguments: map[string]any{
			"name": "com.example/echo",
		},
	})
	require.NoError(t, err)
	raw, _ = json.Marshal(res.StructuredContent)
	var readmeOut ServerReadmePayload
	require.NoError(t, json.Unmarshal(raw, &readmeOut))
	assert.Equal(t, "com.example/echo", readmeOut.Server)
	assert.Equal(t, "1.0.0", readmeOut.Version)
	assert.Equal(t, "text/markdown", readmeOut.ContentType)
	assert.Equal(t, "aabb", readmeOut.SHA256[:4])

	// get_server (defaults to latest)
	res, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_server",
		Arguments: map[string]any{
			"name": "com.example/echo",
		},
	})
	require.NoError(t, err)
	raw, _ = json.Marshal(res.StructuredContent)
	var serverOut apiv0.ServerListResponse
	require.NoError(t, json.Unmarshal(raw, &serverOut))
	require.Len(t, serverOut.Servers, 1)
	assert.Equal(t, "com.example/echo", serverOut.Servers[0].Server.Name)
	assert.Equal(t, "1.0.0", serverOut.Servers[0].Server.Version)
}

func TestAgentAndSkillTools_ListAndGet(t *testing.T) {
	ctx := context.Background()

	reg := servicetesting.NewFakeRegistry()
	reg.Agents = []*models.AgentResponse{
		{
			Agent: models.AgentJSON{
				AgentManifest: models.AgentManifest{
					Name:      "com.example/agent",
					Language:  "go",
					Framework: "none",
				},
				Title:   "Agent",
				Version: "1.0.0",
				Status:  string(model.StatusActive),
			},
		},
	}
	reg.Skills = []*models.SkillResponse{
		{
			Skill: models.SkillJSON{
				Name:    "com.example/skill",
				Title:   "Skill",
				Version: "2.0.0",
				Status:  string(model.StatusActive),
			},
		},
	}

	server := NewServer(reg)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, serverSession.Wait())
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	// list_agents
	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_agents",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	raw, _ := json.Marshal(res.StructuredContent)
	var agentList models.AgentListResponse
	require.NoError(t, json.Unmarshal(raw, &agentList))
	require.Len(t, agentList.Agents, 1)
	assert.Equal(t, "com.example/agent", agentList.Agents[0].Agent.Name)

	// get_agent
	res, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_agent",
		Arguments: map[string]any{
			"name": "com.example/agent",
		},
	})
	require.NoError(t, err)
	raw, _ = json.Marshal(res.StructuredContent)
	var agentOne models.AgentResponse
	require.NoError(t, json.Unmarshal(raw, &agentOne))
	assert.Equal(t, "com.example/agent", agentOne.Agent.Name)

	// list_skills
	res, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_skills",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	raw, _ = json.Marshal(res.StructuredContent)
	var skillList models.SkillListResponse
	require.NoError(t, json.Unmarshal(raw, &skillList))
	require.Len(t, skillList.Skills, 1)
	assert.Equal(t, "com.example/skill", skillList.Skills[0].Skill.Name)

	// get_skill
	res, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_skill",
		Arguments: map[string]any{
			"name": "com.example/skill",
		},
	})
	require.NoError(t, err)
	raw, _ = json.Marshal(res.StructuredContent)
	var skillOne models.SkillResponse
	require.NoError(t, json.Unmarshal(raw, &skillOne))
	assert.Equal(t, "com.example/skill", skillOne.Skill.Name)
}

func TestAgentCardTools_GetAndSearch(t *testing.T) {
	ctx := context.Background()

	testCard := json.RawMessage(`{
		"name": "Test Agent",
		"description": "A test agent",
		"url": "https://example.com/agent",
		"version": "1.0.0",
		"capabilities": {},
		"defaultInputModes": ["text"],
		"defaultOutputModes": ["text"],
		"skills": [{"id": "s1", "name": "Skill One", "description": "Does things"}]
	}`)

	reg := servicetesting.NewFakeRegistry()
	reg.Agents = []*models.AgentResponse{
		{
			Agent: models.AgentJSON{
				AgentManifest: models.AgentManifest{
					Name:      "com.example/agent",
					Language:  "go",
					Framework: "none",
				},
				Title:   "Agent",
				Version: "1.0.0",
				Status:  string(model.StatusActive),
			},
		},
		{
			Agent: models.AgentJSON{
				AgentManifest: models.AgentManifest{
					Name:      "com.example/no-card",
					Language:  "python",
					Framework: "adk",
				},
				Title:   "No Card Agent",
				Version: "2.0.0",
				Status:  string(model.StatusActive),
			},
		},
	}
	reg.AgentCards["com.example/agent:1.0.0"] = testCard
	// Support latest-version lookup (empty version means latest).
	// AgentCards map is populated at setup before the server starts; no lock needed.
	reg.GetAgentCardFn = func(_ context.Context, agentName, version string) (json.RawMessage, string, error) {
		resolvedVersion := version
		if version == "" || version == "latest" {
			resolvedVersion = "1.0.0"
		}
		key := agentName + ":" + resolvedVersion
		if card, ok := reg.AgentCards[key]; ok {
			return card, resolvedVersion, nil
		}
		return nil, "", database.ErrNotFound
	}

	server := NewServer(reg)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, serverSession.Wait())
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	// search_agent_cards - returns only agents with cards
	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_agent_cards",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var searchResp models.AgentListResponse
	require.NoError(t, json.Unmarshal(raw, &searchResp))
	require.Len(t, searchResp.Agents, 1)
	assert.Equal(t, "com.example/agent", searchResp.Agents[0].Agent.Name)
	assert.Equal(t, 1, searchResp.Metadata.Count)

	// search_agent_cards - with search and limit params accepted
	res, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_agent_cards",
		Arguments: map[string]any{"search": "example", "limit": 5},
	})
	require.NoError(t, err)
	raw, err = json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var filteredResp models.AgentListResponse
	require.NoError(t, json.Unmarshal(raw, &filteredResp))
	// FakeRegistry ignores SubstringName; HasCard filter is applied, so count == 1.
	assert.Equal(t, 1, filteredResp.Metadata.Count)

	// search_agent_cards - with version=latest filter
	res, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_agent_cards",
		Arguments: map[string]any{"version": "latest"},
	})
	require.NoError(t, err)
	raw, err = json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var latestResp models.AgentListResponse
	require.NoError(t, json.Unmarshal(raw, &latestResp))
	assert.Equal(t, 1, latestResp.Metadata.Count)
}
