package main

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestGetLangs(t *testing.T) {
	testFs := fstest.MapFS{
		"README.md":            {},
		"foo/package.json":     {},
		"foo/bar/package.json": {}, // We want to ignore this and use the top-level when both present.
		"my-go-project/go.mod": {},
	}

	got := getLangs(testFs)
	want := map[string]string{
		"go":         "/my-go-project",
		"typescript": "/foo",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v; want %v", got, want)
	}
}

func TestFindFiles(t *testing.T) {
	testFs := fstest.MapFS{
		"README.md":                   {},
		"cdk/node_modules/foo/go.mod": {}, // We've seen real examples like this.
		"my-go-project/go.mod":        {},
	}

	got := findFiles(testFs, "go.mod", []string{"node_modules"})
	want := []string{"my-go-project/go.mod"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v; want %v", got, want)
	}
}
