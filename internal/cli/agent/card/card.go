package card

import (
	"github.com/agentregistry-dev/agentregistry/internal/client"

	"github.com/spf13/cobra"
)

var apiClient *client.Client

// SetAPIClient sets the API client for card subcommands.
func SetAPIClient(c *client.Client) {
	apiClient = c
}

// CardCmd is the root command for agent card operations.
var CardCmd = &cobra.Command{
	Use:   "card",
	Short: "Manage A2A Agent Cards",
	Long:  `Commands for managing A2A Agent Cards on registered agents.`,
}

func init() {
	CardCmd.AddCommand(SearchCmd)
}
