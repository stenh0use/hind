Feature: HIND releases
    As a maintainer of the HIND CLI
    I want an easy way to view all hind releases and the HashiCorp binary versions they include
    So that I can quickly identify which release to build or publish

    Background:
        Given I have defined the hind version in the version configuration
        And the hind version has the defined consul version
        And the vault version
        And the nomad version

    Scenario: List available hind versions
        Given I run hind releases
        When I execute the command
        Then the CLI lists all available hind versions in a table
        And the column headers are printed on the first row
        And the first column is the hind version
        And the remaining columns are displayed in alphabetical order: consul, nomad, vault
        And the latest version is on the first row
        And the oldest version is on the last row
