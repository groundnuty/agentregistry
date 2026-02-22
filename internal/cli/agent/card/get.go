package card

import (
	"fmt"

	"github.com/agentregistry-dev/agentregistry/pkg/models"
	"github.com/agentregistry-dev/agentregistry/pkg/printer"
	"github.com/spf13/cobra"
)

var (
	getVersion string
)

// GetCmd retrieves an A2A Agent Card.
var GetCmd = &cobra.Command{
	Use:   "get <agent-name>",
	Short: "Get an A2A Agent Card",
	Long: `Retrieve the A2A Agent Card for an agent.

Returns the latest version by default. Use --version to get a specific version.

Examples:
  arctl agent card get my-agent
  arctl agent card get my-agent --version 1.0.0`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE:          runGet,
}

func init() {
	GetCmd.Flags().StringVar(&getVersion, "version", "", "Specific agent version (default: latest)")
}

func runGet(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	resp, err := fetchAgentCard(agentName, getVersion)
	if err != nil {
		return err
	}

	p := printer.New(printer.OutputTypeJSON, false)
	if err := p.PrintJSON(resp.Card); err != nil {
		return fmt.Errorf("printing card: %w", err)
	}

	return nil
}

func fetchAgentCard(agentName, version string) (*models.AgentCardResponse, error) {
	if version != "" {
		resp, err := apiClient.GetAgentCard(agentName, version)
		if err != nil {
			return nil, fmt.Errorf("getting agent card: %w", err)
		}
		if resp == nil {
			return nil, fmt.Errorf("no agent card found for %s v%s", agentName, version)
		}
		return resp, nil
	}
	resp, err := apiClient.GetAgentCardLatest(agentName)
	if err != nil {
		return nil, fmt.Errorf("getting agent card: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("no agent card found for %s", agentName)
	}
	return resp, nil
}
