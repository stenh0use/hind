package list

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

// DefaultListTimeout is the default timeout for listing clusters
const DefaultListTimeout = 30 * time.Second

// clusterStatus holds aggregated cluster status information
type clusterStatus struct {
	Status       string    // running, partial, stopped, degraded, not-found
	RunningNodes int       // Number of running containers
	TotalNodes   int       // Total expected containers
	Created      time.Time // Creation time of oldest container
}

// NewCommand creates the cluster list command
func NewCommand(logger *log.Logger) *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all hind clusters",
		Long:  "List all hind clusters and their status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, timeout)
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", DefaultListTimeout, "Timeout for listing clusters")

	return cmd
}

func runE(ctx context.Context, logger *log.Logger, timeout time.Duration) error {
	logger.WithField("timeout", timeout).Debug("Listing clusters with timeout")

	// Get list of cluster names
	clusters, err := cluster.List()
	if err != nil {
		return fmt.Errorf("failed getting cluster list: %w", err)
	}

	if len(clusters) == 0 {
		fmt.Println("No clusters found")
		return nil
	}

	// Get active cluster
	activeCluster, err := cluster.GetActiveCluster()
	if err != nil {
		logger.Warnf("Failed to get active cluster: %v", err)
	}

	// Retrieve status for each cluster
	clusterStatuses := make(map[string]*clusterStatus)
	for _, clusterName := range clusters {
		status, err := getClusterStatus(ctx, logger, clusterName, timeout)
		if err != nil {
			logger.Warnf("Failed to get status for cluster %s: %v", clusterName, err)
			// Use error status as fallback
			status = &clusterStatus{
				Status:     "error",
				TotalNodes: 0,
			}
		}
		clusterStatuses[clusterName] = status
	}

	// Print clusters in table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tACTIVE\tSTATUS\tNODES\tCREATED")

	for _, clusterName := range clusters {
		status := clusterStatuses[clusterName]

		activeIndicator := ""
		if clusterName == activeCluster {
			activeIndicator = "*"
		}

		nodesDisplay := fmt.Sprintf("%d/%d", status.RunningNodes, status.TotalNodes)
		if status.Status == "error" || status.Status == "not-found" {
			nodesDisplay = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			clusterName,
			activeIndicator,
			status.Status,
			nodesDisplay,
			formatCreatedTime(status.Created),
		)
	}

	w.Flush()
	return nil
}

// getClusterStatus retrieves the status of a cluster with timeout
func getClusterStatus(ctx context.Context, logger *log.Logger, clusterName string, timeout time.Duration) (*clusterStatus, error) {
	statusCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create cluster manager
	manager, err := cluster.New(logger, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Get cluster info from manager
	info, err := manager.Get(statusCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	// Aggregate status
	return aggregateClusterStatus(info, manager.Config()), nil
}

// aggregateClusterStatus computes cluster-level status from container statuses
func aggregateClusterStatus(info *provider.ClusterInfo, cfg *config.Cluster) *clusterStatus {
	status := &clusterStatus{
		TotalNodes: len(cfg.Nodes),
	}

	if len(info.Containers) == 0 {
		status.Status = "not-found"
		return status
	}

	var (
		runningCount = 0
		stoppedCount = 0
		errorCount   = 0
		oldestTime   = time.Now()
	)

	for _, container := range info.Containers {
		// Count status types
		switch container.Status {
		case provider.Running.String():
			runningCount++
		case provider.Stopped.String():
			stoppedCount++
		case provider.Error.String():
			errorCount++
		}

		// Track oldest creation time
		if created, err := parseCreatedTime(container.Created); err == nil {
			if created.Before(oldestTime) {
				oldestTime = created
			}
		}
	}

	status.RunningNodes = runningCount
	status.Created = oldestTime

	// Determine overall status
	if errorCount > 0 {
		status.Status = "degraded"
	} else if runningCount == len(info.Containers) && runningCount == status.TotalNodes {
		status.Status = "running"
	} else if stoppedCount == len(info.Containers) {
		status.Status = "stopped"
	} else {
		status.Status = "partial"
	}

	return status
}

// parseCreatedTime parses Docker's created time format
func parseCreatedTime(created string) (time.Time, error) {
	// Docker returns times in various formats, handle common ones
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05 -0700 MST",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, created); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", created)
}

// formatCreatedTime formats a timestamp as relative time
func formatCreatedTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("2006-01-02")
	}
}
