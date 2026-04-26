Feature: hind stop cluster
    As a user of the hind cli
    I want an easy way to stop a running cluster
    So that I can pause my work and free up resources without losing my cluster configuration

    Background:
        Given I have hind cli installed
        And a docker daemon is running

    # Cluster Name Argument
    Scenario: Stop command uses default cluster name when no name specified
        Given a cluster named "default" exists
        And the cluster containers are running
        When I run `hind stop`
        Then the CLI should use cluster name "default"

    Scenario: Stop command uses specified cluster name
        Given a cluster named "dev" exists
        And the cluster containers are running
        When I run `hind stop dev`
        Then the CLI should use cluster name "dev"

    Scenario: Stop command accepts cluster name as positional argument
        Given a cluster named "my-test-cluster" exists
        And the cluster containers are running
        When I run `hind stop my-test-cluster`
        Then the CLI should use cluster name "my-test-cluster"

    # Basic Stop Flow
    Scenario: Stop stops all containers in a running cluster
        Given a cluster named "default" exists with the following:
            | Component      | Count |
            | Nomad Server   | 1     |
            | Nomad Client   | 3     |
            | Consul Server  | 1     |
        And all cluster containers are running
        When I run `hind stop`
        Then the CLI should stop all cluster containers
        And all containers should be in stopped state
        And the CLI should output "Cluster 'default' stopped successfully"
        And the cluster configuration should be preserved

    Scenario: Stop stops a named cluster
        Given a cluster named "staging" exists
        And the cluster containers are running
        When I run `hind stop staging`
        Then the CLI should stop all containers for cluster "staging"
        And all containers should be in stopped state
        And the CLI should output "Cluster 'staging' stopped successfully"

    Scenario: Stop command is idempotent when cluster already stopped
        Given a cluster named "default" exists
        And all cluster containers are stopped
        When I run `hind stop`
        Then the CLI should detect the cluster is already stopped
        And the CLI should output "Cluster 'default' is already stopped"
        And no containers should be modified

    Scenario: Stop preserves cluster configuration for future restart
        Given a cluster named "default" exists with 5 client nodes
        And the cluster containers are running
        When I run `hind stop`
        Then the CLI should stop all cluster containers
        And the cluster configuration should be preserved
        And the configuration should show 5 client nodes
        And a subsequent `hind start` should resume with the same configuration

    # Partial States
    Scenario: Stop handles partially running cluster
        Given a cluster named "default" exists
        And some cluster containers are running
        And some cluster containers are stopped
        When I run `hind stop`
        Then the CLI should stop all running containers
        And all containers should be in stopped state
        And the CLI should output "Cluster 'default' stopped successfully"

    Scenario: Stop handles unhealthy containers gracefully
        Given a cluster named "default" exists
        And some containers are in failed state
        And some containers are running
        When I run `hind stop`
        Then the CLI should stop all running containers
        And the CLI should not attempt to stop already failed containers
        And the CLI should output "Cluster 'default' stopped (some containers were already failed)"

    # Error Scenarios
    Scenario: Stop fails when cluster does not exist
        Given no cluster named "nonexistent" exists
        When I run `hind stop nonexistent`
        Then the CLI should output an error "Cluster 'nonexistent' not found"
        And the CLI should exit with code 1

    Scenario: Stop fails when Docker daemon is not running
        Given a cluster named "default" exists
        And the docker daemon is not running
        When I run `hind stop`
        Then the CLI should output an error "Docker daemon is not accessible"
        And the CLI should exit with code 1

    Scenario: Stop continues despite container stop failures
        Given a cluster named "default" exists with 3 client nodes
        And the cluster containers are running
        And container "hind.default.nomad-client.02" cannot be stopped
        When I run `hind stop`
        Then the CLI should attempt to stop all containers
        And the CLI should stop containers 1 and 3 successfully
        And the CLI should output a warning "Failed to stop container 'hind.default.nomad-client.02'"
        And the CLI should output "Cluster 'default' partially stopped"
        And the CLI should exit with code 0

    # Force Stop
    Scenario: Stop with force flag kills containers immediately
        Given a cluster named "default" exists
        And the cluster containers are running
        When I run `hind stop --force`
        Then the CLI should kill all containers without graceful shutdown
        And all containers should be in stopped state
        And the CLI should output "Cluster 'default' force stopped"

    Scenario: Stop with timeout flag waits specified duration
        Given a cluster named "default" exists
        And the cluster containers are running
        When I run `hind stop --timeout 30`
        Then the CLI should wait up to 30 seconds for graceful shutdown
        And all containers should be in stopped state
        And the CLI should output "Cluster 'default' stopped successfully"

    # Verbose Output
    Scenario: Stop with verbose flag shows detailed progress
        Given a cluster named "default" exists with 2 client nodes
        And the cluster containers are running
        When I run `hind stop --verbose`
        Then the CLI should output detailed progress including:
            | Log Entry                                    |
            | Checking cluster 'default' status            |
            | Stopping container 'hind.default.nomad.01'   |
            | Stopping container 'hind.default.nomad.02'   |
            | Stopping container 'hind.default.nomad.03'   |
            | Stopping container 'hind.default.consul.01'  |
            | All containers stopped successfully          |

    # Integration with Other Commands
    Scenario: Stop followed by start resumes cluster with same configuration
        Given a cluster named "prod" exists with 4 client nodes
        And the cluster containers are running
        When I run `hind stop prod`
        And I run `hind start prod`
        Then the cluster should start with 4 client nodes
        And all containers should be in running state
        And the CLI should output "Cluster 'prod' started successfully"

    Scenario: Stop does not affect other running clusters
        Given a cluster named "dev" exists and is running
        And a cluster named "staging" exists and is running
        When I run `hind stop dev`
        Then cluster "dev" containers should be stopped
        And cluster "staging" containers should still be running
        And the CLI should output "Cluster 'dev' stopped successfully"
