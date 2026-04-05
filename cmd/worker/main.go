package main

import (
	"go.uber.org/fx"
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

func main() {
	fx.New(
	// Worker Fx modules will be added in future tasks
	).Run()
}
