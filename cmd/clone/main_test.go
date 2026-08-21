package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRepository(t *testing.T) {
	cfg := config{Domain: "github.com", Username: "StrangeQuark"}
	cases := []struct {
		input string
		ssh   bool
		want  string
	}{
		{"authservice", false, "https://github.com/StrangeQuark/authservice"},
		{"testUser/testService", false, "https://github.com/testUser/testService"},
		{"testWeb.com/testUser/testService", false, "https://testWeb.com/testUser/testService"},
		{"testWeb.com/testUser/testService", true, "git@testWeb.com:testUser/testService.git"},
		{"https://github.com/openai/openai-go.git", false, "https://github.com/openai/openai-go.git"},
	}
	for _, test := range cases {
		got, err := resolveRepository(test.input, cfg, test.ssh)
		if err != nil {
			t.Fatalf("resolveRepository(%q): %v", test.input, err)
		}
		if got != test.want {
			t.Errorf("resolveRepository(%q) = %q, want %q", test.input, got, test.want)
		}
	}
	got, err := resolveRepository("https://example.com/team/repo.git", config{}, false)
	if err != nil {
		t.Fatalf("explicit remote without configuration: %v", err)
	}
	if got != "https://example.com/team/repo.git" {
		t.Errorf("explicit remote = %q", got)
	}
}

func TestResolveRepositoryRejectsInvalidInput(t *testing.T) {
	cfg := config{Domain: "github.com", Username: "StrangeQuark"}
	for _, input := range []string{"one/two/three/four", "owner/", "/repo"} {
		if _, err := resolveRepository(input, cfg, false); err == nil {
			t.Errorf("resolveRepository(%q) succeeded, expected failure", input)
		}
	}
}

func TestInitAndUpdateConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clone", "config.toml")
	t.Setenv("CLONE_CONFIG_FILE", path)

	var output bytes.Buffer
	if err := run([]string{"init", "--domain", "https://github.com/", "--username", "StrangeQuark"}, strings.NewReader(""), &output, &output); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"update", "username", "other-user"}, strings.NewReader(""), &output, &output); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Domain != "github.com" || cfg.Username != "other-user" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestParseCloneOptions(t *testing.T) {
	options, err := parseCloneOptions([]string{"repo", "--ssh", "--into", "target", "--", "--depth", "1"})
	if err != nil {
		t.Fatal(err)
	}
	if options.target != "repo" || !options.ssh || options.destination != "target" {
		t.Fatalf("unexpected options: %#v", options)
	}
	if got := strings.Join(options.gitOptions, " "); got != "--depth 1" {
		t.Errorf("git options = %q", got)
	}
}
