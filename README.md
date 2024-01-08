# Configure Dependency Management

This is a simple script to configure your repo for automatic dependency
management using Dependabot and Scala Steward as required.

It supports updates for:

- Github Actions
- Scala
- Typescript
- Go
- Python

The script adds a dependabot.yml file, commits it, and raises a PR.

To use, first ensure you have Go installed and also the Github CLI:

    $ brew install go
    $ brew install gh
    $ gh auth login

Make sure you add Go's bin to your path:

    $ echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.zshrc
    $ source ~/.zshrc

Then install and run:

    $ go install github.com/guardian/configure-dependency-management@latest
    $ cd [your-project]
    $ configure-dependency-management [--dry-run]

## Local Development

You can easily rebuild and install locally using:

    $ go install
