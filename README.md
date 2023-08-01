# Configure Dependency Management

This is a simple script to configure your repo for automatic dependency
management using Dependabot and Scala Steward as required.

It supports updates for:

- Github Actions
- Scala
- Typescript
- Go

The script adds some files, commits them, and raises a PR.

To use, first ensure you have Go installed and also the Github CLI:

    $ brew install go
    $ brew install gh
    $ gh auth login

Then install and run:

    $ go install github.com/guardian/configure-dependency-management
    $ cd [your-project]
    $ configure-dependency-management
