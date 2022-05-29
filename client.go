package artifact

import (
	"context"
)

// Client represents a minimal viable interface that would
// be needed to provide async updates of Artifacts.
type Client interface {
	// Register will associate a name that can help with identifying
	// updates (ie. name = URI(/path/to/schema/definition)) that the client
	// is aware of and can pass those updates to the Artifact.
	// Register MUST fail when the initial load is not possible, any failures
	// after that point is treated as transitive and should be handled
	Register(ctx context.Context, name string, art Artifact) error

	// MonitorUpdates is used to update all registered components
	// that can be done as background task
	MonitorUpdates(ctx context.Context) error
}
