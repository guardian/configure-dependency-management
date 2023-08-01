package main

import (
	"bytes"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"golang.org/x/exp/maps"
)

/*
- Check if on `main` and exit with error if not
- branch to bot/configure-dependency-management
- if Go/Typescript/Rust, add the relevant Dependabot file
- commit this
- open a PR and return a link
- if Scala, output instructions for how to add
*/

var dependabotFilePath = ".github/workflows/dependabot.yml"

func main() {
	if !isOnMain() {
		exit("switch to main branch before running this script.")
	}

	if !ghCLIInstalled() {
		exit("please install the GitHub CLI and authenticate before running this script.")
	}

	langs := getLangs()
	msg("Detected the following languages: " + strings.Join(maps.Keys(langs), ", "))

	if len(langs) == 0 {
		exit("unable to configure as no languages detected.")
	}

	if fileExists(dependabotFilePath) {
		exit("existing Dependabot config found. Please remove this before running to continue.")
	}

	config := dependabotConfig(langs)
	err := writeWithDir(dependabotFilePath, []byte(config), 0644)
	check(err, "unable to write Dependabot config")

	msg("Dependabot config written to " + dependabotFilePath)

	err = commit()
	check(err, "unable to commit Dependabot config")

	//err = createPR()
	//check(err, "unable to create PR - but you can do this manually of course")

	if langs["scala"] != "" {
		msg("Please follow the instructions at https://github.com/guardian/scala-steward-public-repos to add Scala Steward to this repo. This is configured via the UI so cannot be done here.")
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

func createPR() error {
	return exec.Command("gh", "pr", "create", "--head", "bot/configure-dependency-management", "--base", "main", "--title", "feat: add Dependabot config", "--body", "This PR was created by [a script](https://github.com/guardian/configure-dependency-management) to configure Dependabot. Please review and merge if appropriate.").Run()
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

func getLangs() map[string]string {
	candidates := map[string]string{
		"go":         "go.mod",
		"typescript": "package.json",
		"rust":       "Cargo.toml",
		"scala":      "build.sbt",
	}

	langs := map[string]string{}
	for lang, file := range candidates {
		filePath, exists := findFile(os.DirFS("."), file)

		if exists {
			langs[lang] = dependabotRoot(filePath)
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

func findFile(fsys fs.FS, name string) (string, bool) {
	filePath := ""
	found := false

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type().IsRegular() && d.Name() == name {
			filePath = path + d.Name()
			found = true
		}

		return nil
	})

	if err != nil {
		log.Print("err: " + err.Error())
	}

	return filePath, found
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
      interval: "weekly"
    commit-message:
      prefix: "chore(deps): "
    labels:
      - "dependencies"
	groups:
	  all:
	    patterns: "*"
{{ if .typescript }}
  - package-ecosystem: "npm"
    directory: "{{ .typescript }}"
    schedule:
      interval: "weekly"
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
	    patterns: "*"
{{ end }}
{{ if .go }}
  - package-ecosystem: "go"
    directory: "{{ .go }}"
	schedule:
	  interval: "weekly"
	commit-message:
	  prefix: "chore(deps): "
	labels:
	  - "dependencies"
	groups:
	  all:
	    patterns: "*"
{{ end }}
`

	w := &bytes.Buffer{}
	template.Must(template.New("dependabot").Parse(tpl)).Execute(w, langs)

	return w.String()
}
