package card

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/agentregistry-dev/agentregistry/pkg/models"
	"github.com/agentregistry-dev/agentregistry/pkg/printer"
	"github.com/spf13/cobra"
)

var (
	publishVersion string
	publishFile    string
)

// PublishCmd publishes an A2A Agent Card from a JSON file.
var PublishCmd = &cobra.Command{
	Use:   "publish <agent-name>",
	Short: "Publish an A2A Agent Card",
	Long: `Publish an A2A Agent Card from a JSON file.

The file should contain the raw Agent Card JSON (the same format
produced by 'arctl agent init'), not wrapped in {"card": ...}.

Examples:
  arctl agent card publish my-agent --version 1.0.0 -f agent-card.json`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE:          runPublish,
}

func init() {
	PublishCmd.Flags().StringVar(&publishVersion, "version", "", "Agent version to attach the card to (required)")
	_ = PublishCmd.MarkFlagRequired("version")
	PublishCmd.Flags().StringVarP(&publishFile, "file", "f", "", "Path to Agent Card JSON file (required)")
	_ = PublishCmd.MarkFlagRequired("file")
}

func runPublish(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	data, err := os.ReadFile(publishFile)
	if err != nil {
		return fmt.Errorf("reading card file: %w", err)
	}

	var card json.RawMessage
	if err := json.Unmarshal(data, &card); err != nil {
		return fmt.Errorf("parsing card JSON: %w", err)
	}

	if err := models.ValidateA2AAgentCard(card); err != nil {
		return fmt.Errorf("validating agent card: %w", err)
	}

	if err := apiClient.UpsertAgentCard(agentName, publishVersion, card); err != nil {
		return fmt.Errorf("publishing agent card: %w", err)
	}

	printer.PrintSuccess(fmt.Sprintf("Published agent card for %s v%s", agentName, publishVersion))
	return nil
}
