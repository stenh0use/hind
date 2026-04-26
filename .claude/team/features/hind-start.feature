Feature: hind start cluster
    As a user of the hind cli
    I want an easy way to start or create a cluster
    So that I can quickly begin working with HashiCorp services

    Background:
        Given I have hind cli installed
        And a docker daemon is running
        And no clusters are currently running

    # Cluster Name Argument
    Scenario: Start command uses default cluster name when no name specified
        When I run `hind start`
        Then the CLI should use cluster name "default"

    Scenario: Start command uses specified cluster name
        When I run `hind start dev`
        Then the CLI should use cluster name "dev"

    Scenario: Start command accepts cluster name as positional argument
        When I run `hind start my-test-cluster`
        Then the CLI should use cluster name "my-test-cluster"

    # Cluster Creation Flow
    Scenario: Start creates a new cluster when none exists
        Given no cluster named "default" exists
        When I run `hind start`
        Then the CLI should detect no existing cluster
        And the CLI should create a new cluster with the following:
            | Component      | Count |
            | Nomad Server   | 1     |
            | Nomad Client   | 1     |
            | Consul Server  | 1     |
        And all containers should be in running state
        And the CLI should output "Cluster 'default' started successfully"
        And the CLI should display connection information

    Scenario: Start creates a named cluster when none exists
        Given no cluster named "dev" exists
        When I run `hind start dev`
        Then the CLI should detect no existing cluster
        And the CLI should create a new cluster named "dev"
        And all containers should be in running state
        And the CLI should output "Cluster 'dev' started successfully"

    Scenario: Start resumes a stopped cluster
        Given a cluster named "default" exists
        And the cluster containers are stopped
        When I run `hind start`
        Then the CLI should detect existing cluster containers
        And the CLI should start all stopped containers
        And all containers should be in running state
        And the CLI should output "Cluster 'default' started successfully"

    Scenario: Start command is idempotent when cluster already running
        Given a cluster named "default" exists
        And the cluster containers are running
        When I run `hind start`
        Then the CLI should detect the cluster is already running
        And the CLI should output "Cluster 'default' is already running"
        And no containers should be created or restarted

    # Configuration Options
    Scenario: Start cluster with custom node count
        Given no cluster named "default" exists
        When I run `hind start --clients 3`
        Then the CLI should create a cluster with 3 client nodes
        And all 3 client containers should be running

    Scenario: Start named cluster with custom node count
        Given no cluster named "staging" exists
        When I run `hind start staging --clients 5`
        Then the CLI should create a cluster named "staging" with 5 client nodes
        And all 5 client containers should be running

    Scenario: Start uses existing cluster configuration when no flags provided
        Given a cluster named "default" exists with 3 client nodes
        And the cluster containers are stopped
        When I run `hind start`
        Then the CLI should start the cluster with 3 client nodes
        And the CLI should not modify the cluster configuration
        And the CLI should output "Cluster 'default' started successfully"

    Scenario: Start scales existing cluster when clients flag provided
        Given a cluster named "default" exists with 3 client nodes
        And the cluster containers are running
        When I run `hind start --clients 5`
        Then the CLI should scale the cluster to 5 client nodes
        And the CLI should create 2 additional client containers
        And all 5 client containers should be running
        And the cluster configuration should be updated

    Scenario: Start scales down existing cluster when clients flag is lower
        Given a cluster named "default" exists with 5 client nodes
        And the cluster containers are running
        When I run `hind start --clients 2`
        Then the CLI should scale the cluster down to 2 client nodes
        And the CLI should remove 3 client containers
        And 2 client containers should be running
        And the cluster configuration should be updated

    # Error Scenarios
    Scenario: Start fails when Docker daemon is not running
        Given the docker daemon is not running
        When I run `hind start`
        Then the CLI should output an error "Docker daemon is not accessible"
        And the CLI should exit with code 1

    Scenario: Start fails when port conflicts exist
        Given port 4646 is already in use
        When I run `hind start`
        Then the CLI should output an error "Port conflict detected: 4646"
        And the CLI should suggest "Stop the conflicting service or use a different profile"
        And the CLI should exit with code 1

    Scenario: Start partially recovers from unhealthy containers
        Given a cluster named "default" exists
        And some containers are in failed state
        When I run `hind start`
        Then the CLI should detect unhealthy containers
        And the CLI should recreate failed containers
        And all containers should be in running state

    # Verbose Output
    Scenario: Start with verbose flag shows detailed progress
        Given no cluster named "default" exists
        When I run `hind start --verbose`
        Then the CLI should output detailed progress including:
            | Log Entry                          |
            | Checking for existing cluster      |
            | Creating network 'hind-default'    |
            | Pulling image 'hind/nomad:latest'  |
            | Starting container 'nomad-server'  |
            | Waiting for Nomad API readiness    |
            | Cluster health check passed        |
