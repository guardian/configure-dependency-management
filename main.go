package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/exp/slices"
)

var dependabotFilePath = ".github/dependabot.yml"

func main() {
	assert(isOnMain(), "switch to main branch before running this script.")
	assert(ghCLIInstalled(), "please install the GitHub CLI and authenticate before running this script.")

	dryRun := flag.Bool("dry-run", false, "When set, will output the config to stdout instead of writing to the repo.")
	flag.Parse()

	langs := getLangs(os.DirFS("."))
	assert(len(langs) > 0, "unable to configure as no languages detected.")

	if fileExists(dependabotFilePath) && !*dryRun {
		ok := askYN("existing Dependabot config found. Do you want to overwrite it?")
		if !ok {
			exit("existing Dependabot config found. Please remove this before running to continue.")
		}
	}

	config := dependabotConfig(langs)
	if *dryRun {
		fmt.Println(config)
		os.Exit(0)
	}

	check(writeWithDir(dependabotFilePath, []byte(config), 0644), "unable to write Dependabot config")
	msg("Dependabot config written to " + dependabotFilePath)

	check(commit(), "unable to commit Dependabot config")

	link, err := createPR()
	check(err, "unable to create PR but config committed. Please create a PR manually.")

	msg("PR raised at: " + link)

	if langs["scala"] != "" {
		msg("Please follow the instructions at https://github.com/guardian/scala-steward-public-repos (or the private equivalent) to add Scala Steward to this repo. Unfortunately, this is configured via the UI so cannot be done here.")
	}
}

func writeWithDir(filePath string, data []byte, perm fs.FileMode) error {
	dir := path.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, perm)
}

func ghCLIInstalled() bool {
	return exec.Command("gh", "version").Run() == nil
}

func commit() error {
	err := exec.Command("git", "switch", "-c", "bot/configure-dependency-management").Run()
	if err != nil {
		return err
	}

	err = exec.Command("git", "add", dependabotFilePath).Run()
	if err != nil {
		return err
	}

	return exec.Command("git", "commit", "-m", "feat: add Dependabot config").Run()
}

func createPR() (string, error) {
	err := exec.Command("git", "push", "--set-upstream", "origin", "bot/configure-dependency-management", "-f").Run()
	if err != nil {
		return "", err
	}

	out, err := exec.Command("gh", "pr", "create", "--head", "bot/configure-dependency-management", "--base", "main", "--title", "feat: add Dependabot config", "--body", "This PR was created by [a script](https://github.com/guardian/configure-dependency-management) to configure Dependabot. Please review and merge if appropriate.").CombinedOutput()
	return string(out), err
}

func assert(condition bool, msg string) {
	if !condition {
		exit(msg)
	}
}

func check(err error, msg string) {
	if err != nil {
		exit(msg + ": " + err.Error())
	}
}

func msg(msg string) {
	log.Print(msg)
}

func exit(msg string) {
	log.Fatal("Error: " + msg)
}

func askYN(msg string) bool {
	got := ""
	fmt.Print(msg + " (y/n) ")
	fmt.Scanln(&got)
	switch got {
	case "y":
		return true
	case "n":
		return false
	default:
		return askYN("Please enter y or n")
	}
}

func getLangs(fSys fs.FS) map[string]string {
	candidates := map[string]string{
		"go":         "go.mod",
		"typescript": "package.json",
		"rust":       "Cargo.toml",
		"scala":      "build.sbt",
		"python":     "requirements.txt",
	}

	ignoreDirs := []string{"node_modules"}

	langs := map[string]string{}
	for lang, file := range candidates {
		filePaths := findFiles(fSys, file, ignoreDirs)

		// If more than one file per language, we taken the shortest path as the
		// root.
		sort.Slice(filePaths, func(a, b int) bool {
			return len(filePaths[a]) < len(filePaths[b])
		})

		if len(filePaths) > 0 {
			langs[lang] = dependabotRoot(filePaths[0])
		}
	}

	return langs
}

// Returns 'directory' as required by Dependabot for a file. Which means:
// - begins with '/'
// - does not include the file name itself
func dependabotRoot(filePath string) string {
	d := path.Dir(filePath)
	switch d {
	case ".":
		return "/"
	default:
		return "/" + d
	}
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func findFiles(fsys fs.FS, name string, ignoreDirs []string) []string {
	paths := []string{}

	inIgnoreDir := func(path string) bool {
		return slices.ContainsFunc(ignoreDirs, func(dir string) bool {
			return strings.Contains(path, dir)
		})
	}

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type().IsRegular() && d.Name() == name && !inIgnoreDir(path) {
			paths = append(paths, path)
		}

		return nil
	})

	if err != nil {
		log.Print("err: " + err.Error())
	}

	return paths
}

func isOnMain() bool {
	out, err := exec.Command("git", "branch", "--show-current").CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) == "main"
}

type Langs struct {
	Typescript string
	Go         string
	Scala      string
}

func dependabotConfig(langs map[string]string) string {
	tpl := `
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "monthly"
    commit-message:
      prefix: "chore(deps): "
    labels:
      - "dependencies"
    groups:
      all:
        patterns: ["*"]
{{ if .typescript }}
  - package-ecosystem: "npm"
    directory: "{{ .typescript }}"
    schedule:
      interval: "monthly"
    commit-message:
      prefix: "chore(deps): "
    # The version of AWS CDK libraries must match those from @guardian/cdk.
    # We'd never be able to update them here independently, so just ignore them.
    ignore:
      - dependency-name: "aws-cdk"
      - dependency-name: "aws-cdk-lib"
      - dependency-name: "constructs"
    labels:
      - "dependencies"
    groups:
      all:
        patterns: ["*"]
{{ end }}
{{ if .go }}
  - package-ecosystem: "go"
    directory: "{{ .go }}"
    schedule:
      interval: "monthly"
    commit-message:
      prefix: "chore(deps): "
    labels:
      - "dependencies"
    groups:
      all:
        patterns: ["*"]
{{ end }}
{{ if .python }}
  - package-ecosystem: "pip"
    directory: "{{ .python }}"
    schedule:
      interval: "monthly"
    commit-message:
      prefix: "chore(deps): "
    labels:
      - "dependencies"
    groups:
      all:
        patterns: ["*"]
{{ end }}
`

	w := &bytes.Buffer{}
	template.Must(template.New("dependabot").Parse(tpl)).Execute(w, langs)

	return w.String()
}
