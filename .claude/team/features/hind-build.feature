Feature: hind build container images
    As a maintainer or user of the hind cli
    I want an easy way to build specific versions of the hind container images
    So that they are can be built

    Background:
        Given I have defined the hind version in the version configuration
        And the hind version has the defined consul version
        And the vault version
        And the nomad version

    Scenario: Build consul image without version
        Given I run the cli `hind build consul` command
        When I execute the command without a version provided
        Then the build package will leverage the versions package
        And the versions package will provide the latest hind version
        And in the hind version will be the consul version for that release
        And the consul version will be passed to the build command as a build arg
        And the consul image will be built
        And the consul image will be tagged as `hind.consul:<hind_version>`

    Scenario: Build image dependencies met
        Given I run a build command for a target image with dependent base images
        When the target image is built
        Then the build functionality will check the target for base image dependencies
        And the base image dependencies will be checked to confirm they exist
        And the build functionality will build the target image

    Scenario: Build image dependencies not met
        Given I run a build command for a target image with dependent base images
        When the target image is built
        Then the build functionality will check the target for base image dependencies
        And the base image dependencies will be checked to confirm they exist
        And the build will fail with an error message for the missing dependencies
        And the error message will include instructions to resolved the missing dependencies

    Scenario: Build all images
        Given I run a build command for the target image `all`
        When the command is executed
        Then the build will determine the order of the build chain
        And the first image built with no build dependencies will be built
        And the next image built that has it's dependencies met wil be built
        And the remaining images will be built once all of their dependencies are met
