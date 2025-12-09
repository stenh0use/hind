Feature: HIND releases menu
    As a maintainer of the HIND CLI
    I want an easy way to maintain the hind version and the version of the hashicorp binaries that are include
    So that releases can easily be built and published

    Background:
        Given I have defined the hind version in the version configuration
        And the hind version has the defined consul version
        And the vault version
        And the nomad version

    Scenario: List available hind versions
        Given I run the cli versions menu
        When I execute the command
        Then the cli will list in a table the available hind versions
        And the hashistack component versions that are included in a specific version will be displayed on the same row
        And the names of the columns of the table will be listed on the first row
        And the first column will be the hind version
        And the remaining columns will be displayed in alphabetical order consul, nomad, vault
        And the latest version will be on the first row
        And the oldest version will be on the last row

    Scenario: Create new hind cluster
        Given I run the cli command create
        When I execute the command with the
        Then the


    Scenario: Run non existent hind version
