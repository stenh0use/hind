package list

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cmd"
	cluster1 "github.com/stenh0use/hind/pkg/cmd/hind/internal/cluster"
)

// DefaultListTimeout is the default timeout for listing clusters
const DefaultListTimeout = 30 * time.Second

// clusterStatus holds aggregated cluster status information
type clusterStatus = cluster.ClusterStatusSnapshotResult

var newClusterStatusSnapshotService = func() cluster.ClusterStatusSnapshotService {
	return cluster.NewClusterStatusSnapshotService()
}

func getClusterStatusFromInspect(ctx context.Context, inspect orchestration.InspectResult) (*clusterStatus, error) {
	result, err := newClusterStatusSnapshotService().Build(ctx, cluster.ClusterStatusSnapshotRequest{Inspect: inspect})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type clusterListService interface {
	List(ctx context.Context) (orchestration.ListResult, error)
}

type clusterStatusService interface {
	Inspect(ctx context.Context) (orchestration.InspectResult, error)
}

type listServices struct {
	Orchestration clusterListService
	Active        persistence.ActiveRepository
}

var newClusterListServices = func(logger *log.Logger) (listServices, error) {
	svc, err := cluster1.NewClusterServices(logger, "default")
	if err != nil {
		return listServices{}, err
	}
	return listServices{Orchestration: svc.Orchestration, Active: svc.Active}, nil
}

var newClusterStatusService = func(logger *log.Logger, clusterName string) (clusterStatusService, error) {
	svc, err := cluster1.NewClusterServices(logger, clusterName)
	if err != nil {
		return nil, err
	}
	return svc.Orchestration, nil
}

var getClusterStatusFn = getClusterStatus

// NewCommand creates the cluster list command
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	var timeout time.Duration

	command := &cobra.Command{
		Use:   "list",
		Short: "List all hind clusters",
		Long:  "List all hind clusters and their status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, streams, timeout)
		},
	}

	command.Flags().DurationVar(&timeout, "timeout", DefaultListTimeout, "Timeout for listing clusters")

	return command
}

func runE(ctx context.Context, logger *log.Logger, streams cmd.IOStreams, timeout time.Duration) error {
	listSvc, err := newClusterListServices(logger)
	if err != nil {
		return fmt.Errorf("failed to create cluster list service: %w", err)
	}

	logger.WithField("timeout", timeout).Debug("Listing clusters with timeout")

	// Get list of cluster names
	listResult, err := listSvc.Orchestration.List(ctx)
	if err != nil {
		return fmt.Errorf("failed getting cluster list: %w", err)
	}
	clusters := listResult.Names

	if len(clusters) == 0 {
		fmt.Fprintln(streams.ErrOut, "No clusters found")
		return nil
	}

	// Get active cluster
	activeCluster, err := listSvc.Active.GetActive(ctx)
	if err != nil {
		logger.Warnf("Failed to get active cluster: %v", err)
	}

	// Retrieve status for each cluster
	clusterStatuses := make(map[string]*clusterStatus)
	for _, clusterName := range clusters {
		status, err := getClusterStatusFn(ctx, logger, clusterName, timeout)
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
	w := tabwriter.NewWriter(streams.Out, 0, 0, 3, ' ', 0)
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

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush list output: %w", err)
	}
	return nil
}

// getClusterStatus retrieves the status of a cluster with timeout
func getClusterStatus(ctx context.Context, logger *log.Logger, clusterName string, timeout time.Duration) (*clusterStatus, error) {
	statusCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	svc, err := newClusterStatusService(logger, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster status service: %w", err)
	}
	inspect, err := svc.Inspect(statusCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect cluster: %w", err)
	}

	status, buildErr := getClusterStatusFromInspect(statusCtx, inspect)
	if buildErr != nil {
		return nil, fmt.Errorf("failed to build status snapshot: %w", buildErr)
	}
	return status, nil
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
