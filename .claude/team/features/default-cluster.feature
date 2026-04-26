Feature: hind active cluster selection
    As a user of the hind cli
    I want to be able to select an active selected cluster
    So that when I run any cli commands I do not need to specify the profile name of the cluster

    Scenario:
        Given I run the command `hind start`
        And the command is successfull
        When the command is executed
        Then the active cluster profile name will be set as the newly active cluster

    Scenario:
        Given I run the command `hind delete`
        And the cluster to be deleted is the active selected cluster
        And the command is successfull
        When the command is executed
        Then the active cluster profile name will be reset to "default"

    Scenario:
        Given I run the command `hind set profile [name]`
        And the cluster [name] exists
        When the command is executed
        Then the active selected cluster will be change to name

    Scenario:
        Given I run the command `hind set profile [name]`
        And the cluster [name] does not exist
        When the command is executed
        Then the command will fail
        And the command will print a message that lets the user no that the profile does not exist
