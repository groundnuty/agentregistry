package card

import (
	"fmt"
	"os"

	"github.com/agentregistry-dev/agentregistry/pkg/printer"
	"github.com/spf13/cobra"
)

var (
	searchOutputFormat string
)

// SearchCmd lists agents that have A2A Agent Cards.
var SearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for agents with A2A Agent Cards",
	Long: `List all agents that have a published A2A Agent Card.

Examples:
  arctl agent card search
  arctl agent card search -o json`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE:          runSearch,
}

func init() {
	SearchCmd.Flags().StringVarP(&searchOutputFormat, "output", "o", "table", "Output format (table, json)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	agents, err := apiClient.GetAgentsWithCards()
	if err != nil {
		return fmt.Errorf("searching agent cards: %w", err)
	}

	if len(agents) == 0 {
		printer.PrintInfo("No agents with A2A cards found")
		return nil
	}

	switch searchOutputFormat {
	case "table", "json":
	default:
		return fmt.Errorf("unsupported output format %q: must be one of: table, json", searchOutputFormat)
	}

	if searchOutputFormat == "json" {
		p := printer.New(printer.OutputTypeJSON, false)
		if err := p.PrintJSON(agents); err != nil {
			return fmt.Errorf("printing results: %w", err)
		}
		return nil
	}

	t := printer.NewTablePrinter(os.Stdout)
	t.SetHeaders("Name", "Version", "Description")
	for _, a := range agents {
		t.AddRow(
			printer.TruncateString(a.Agent.Name, 40),
			a.Agent.Version,
			printer.TruncateString(printer.EmptyValueOrDefault(a.Agent.Description, "<none>"), 60),
		)
	}
	if err := t.Render(); err != nil {
		return fmt.Errorf("rendering table: %w", err)
	}

	printer.PrintInfo(fmt.Sprintf("%d agent(s) with A2A cards", len(agents)))
	return nil
}
