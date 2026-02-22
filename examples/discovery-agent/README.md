# Discovery Agent Example

A [kagent](https://github.com/kagent-dev/kagent) Agent that uses the AgentRegistry MCP server to discover agents, MCP servers, and skills via natural language queries.

## Prerequisites

- A Kubernetes cluster with [kagent](https://github.com/kagent-dev/kagent) installed
- An AgentRegistry server running and accessible from the cluster
- An OpenAI API key stored as a Kubernetes secret

## Setup

1. Create the OpenAI API key secret (if not already present):

```bash
kubectl create secret generic kagent-openai \
  --namespace kagent \
  --from-literal=OPENAI_API_KEY=<your-key>
```

2. Update the AgentRegistry URL in `remote-mcp-server.yaml` to match your deployment.

3. Apply the resources:

```bash
kubectl apply -f model-config.yaml
kubectl apply -f remote-mcp-server.yaml
kubectl apply -f agent.yaml
```

## Usage

Once deployed, the Discovery Agent can answer questions like:

- "Find agents that can help with Kubernetes troubleshooting"
- "What MCP servers are available for GitHub?"
- "Show me the details and A2A card for the k8s-agent"
- "Deploy the filesystem MCP server"

The agent uses `search_agent_cards` to find agents by capability, and
`get_agent` to retrieve full details including the inline A2A Agent Card.

## MCP Tools Available

The agent has access to 16 AgentRegistry MCP tools:

| Tool | Description |
|------|-------------|
| `list_agents` | List published agents |
| `get_agent` | Get agent details (includes A2A card inline) |
| `search_agent_cards` | Search agents with A2A cards |
| `list_servers` | List MCP servers |
| `get_server` | Get server details |
| `get_server_readme` | Get server README |
| `list_skills` | List skills |
| `get_skill` | Get skill details |
| `list_deployments` | List deployments |
| `get_deployment` | Get deployment details |
| `deploy_server` | Deploy an MCP server |
| `deploy_agent` | Deploy an agent |
| `update_deployment_config` | Update deployment config |
| `remove_deployment` | Remove a deployment |
| `registry_health` | Health check |
| `registry_version` | Version info |
