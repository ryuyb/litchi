package main

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// Worker is an optional standalone process for background task processing.
// It can be deployed independently from the main HTTP server.
//
// TODO: Implement worker modules (refer to task documentation):
//   - Task queue consumer (T2.4.3 TaskScheduler)
//   - Agent execution worker (T4.3.* Agent integration)
//   - Recovery service (T5.2.1 Service restart recovery)
//   - Webhook delivery retry worker (T4.1.4 Webhook handling)
//
// For now, all processing happens within the main server process.
// Deploy this worker only when horizontal scaling of task processing is needed.

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start the background worker",
	Long: `Start the Litchi background worker for task processing.
This is an optional standalone process that can be deployed independently
from the main HTTP server for horizontal scaling of task processing.

Currently in placeholder mode - all processing happens within the server.`,
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			fx.WithLogger(func() fxevent.Logger { return getFxLogger("") }),
			// Supply pre-loaded config (loaded in rootCmd.PersistentPreRunE)
			fx.Supply(loadedCfg),
			// Worker Fx modules will be added in future tasks
		).Run()
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
}