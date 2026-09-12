package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avatar31/halmidi/config"
)

var (
	rootCmd = &cobra.Command{
		Use:           config.APP_NAME,
		Short:         "Halmidi Object Storage Server",
		Long:          "Halmidi is a lightweight S3 compatible object storage server.",
		SilenceUsage:  true, // Don't show usage on error
		SilenceErrors: true, // Don't let Cobra print errors
	}

	// Starts the server
	startCmd = &cobra.Command{
		Use:    "start",
		Short:  "Start Halmidi server",
		Long:   "Starts the Halmidi Object Storage Server",
		Hidden: true,
		RunE:   startCmdHandler,
	}

	// Prints version information
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Halmidi version",
		Long:  "Print the version number of Halmidi",
		Args:  cobra.NoArgs,
		Run:   versionCmdHandler,
	}
)

func startCmdHandler(cmd *cobra.Command, args []string) error {
	configFile := config.CONFIG_FILE_PATH
	port := config.DEFAULT_REST_PORT
	if config.IsDevEnv() {
		var err error
		configFile, err = cmd.Flags().GetString("dev-config")
		if err != nil {
			return err
		}

		port, err = cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}
	}

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}

	if conf.Mode == config.MODE_CLUSTER {
		id, err := cmd.Flags().GetUint64("id")
		if err != nil {
			return err
		}
		if id == 0 {
			return fmt.Errorf("required flag(s) id not set")
		}

		nodename, err := cmd.Flags().GetString("nodename")
		if err != nil {
			return err
		}
		if nodename == "" {
			return fmt.Errorf("required flag(s) nodename not set")
		}

		peersStr, err := cmd.Flags().GetString("peers")
		if err != nil {
			return err
		}
		if peersStr == "" {
			return fmt.Errorf("required flag(s) peers not set")
		}

		peers := strings.Split(peersStr, ",")
		urlMap := make(map[uint64]string)
		for _, entry := range peers {
			var nodeID uint64
			var nodeURL string
			_, err := fmt.Sscanf(entry, "%d=%s", &nodeID, &nodeURL)
			if err != nil {
				return fmt.Errorf("invalid cluster entry: %s", entry)
			}
			urlMap[nodeID] = nodeURL
		}

		if _, ok := urlMap[id]; !ok {
			return fmt.Errorf("node ID %d not found in cluster configuration", id)
		}

		Start(port, id, nodename, urlMap)
		return nil
	}

	Start(port, 1, "node0", map[uint64]string{1: fmt.Sprintf("http://127.0.0.1:%d", port+1)})
	return nil
}

func versionCmdHandler(cmd *cobra.Command, args []string) {
	cmd.Printf("Halmidi v%s\n", config.APP_VERSION)
}
