package main

import (
	"log"
	"os/exec"
)

/*
- Check if on `main` and exit with error if not
- branch to bot/configure-dependency-management
- if Go/Typescript/Rust, add the relevant Dependabot file
- commit this
- open a PR and return a link
- if Scala, output instructions for how to add
*/

func main() {
	if !isOnMain() {
		exit("switch to main branch before running this script.")
	}
}

func exit(msg string) {
	log.Fatal("Error: " + msg)
}

func isOnMain() bool {
	out, err := exec.Command("git", "branch", "--show-current").CombinedOutput()

	return err != nil && string(out) != "main"
}
