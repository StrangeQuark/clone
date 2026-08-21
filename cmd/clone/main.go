// Command clone expands repository shortcuts into Git remotes and runs git clone.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var version = "dev"

type config struct {
	Domain   string
	Username string
}

type cloneOptions struct {
	target      string
	destination string
	ssh         bool
	background  bool
	help        bool
	gitOptions  []string
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "clone:", err)
		os.Exit(1)
	}
}

func run(args []string, in io.Reader, out, errOut io.Writer) error {
	if len(args) == 0 {
		return cloneRepository(args, in, out, errOut)
	}

	switch args[0] {
	case "init":
		return initConfig(args[1:], in, out)
	case "update":
		return updateConfig(args[1:], out)
	case "config":
		if len(args) != 1 {
			return errors.New("config takes no arguments")
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if cfg.Domain == "" || cfg.Username == "" {
			_, err = fmt.Fprintf(out, "No complete configuration at %s\n", configPath())
			return err
		}
		_, err = fmt.Fprintf(out, "domain=%s\nusername=%s\nconfig=%s\n", cfg.Domain, cfg.Username, configPath())
		return err
	case "--help", "-h", "help":
		printUsage(out)
		return nil
	case "--version":
		_, err := fmt.Fprintf(out, "clone %s\n", version)
		return err
	default:
		return cloneRepository(args, in, out, errOut)
	}
}

func printUsage(out io.Writer) {
	fmt.Fprint(out, "Usage:\n"+
		"  clone <repository> [--ssh] [--into <directory>] [--background] [-- <git clone options>]\n"+
		"  clone init [--domain <domain>] [--username <username>]\n"+
		"  clone update domain <domain>\n"+
		"  clone update username <username>\n"+
		"  clone config\n\n"+
		"Repository forms:\n"+
		"  repository                Uses the configured domain and username\n"+
		"  owner/repository          Uses the configured domain\n"+
		"  domain/owner/repository   Uses the supplied domain\n"+
		"  https://... or git@...    Passed to Git unchanged\n")
}

func configPath() string {
	if path := os.Getenv("CLONE_CONFIG_FILE"); path != "" {
		return path
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".config", "clone", "config.toml")
	}
	return filepath.Join(dir, "clone", "config.toml")
}

func loadConfig() (config, error) {
	path := configPath()
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return config{}, nil
	}
	if err != nil {
		return config{}, fmt.Errorf("read configuration: %w", err)
	}
	defer file.Close()

	var cfg config
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		value = strings.TrimSpace(value)
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return config{}, fmt.Errorf("invalid value for %q in %s", strings.TrimSpace(key), path)
		}
		switch strings.TrimSpace(key) {
		case "domain":
			cfg.Domain = unquoted
		case "username":
			cfg.Username = unquoted
		}
	}
	if err := scanner.Err(); err != nil {
		return config{}, fmt.Errorf("read configuration: %w", err)
	}
	return cfg, nil
}

func saveConfig(cfg config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	contents := fmt.Sprintf("# clone configuration\ndomain = %s\nusername = %s\n",
		strconv.Quote(cfg.Domain), strconv.Quote(cfg.Username))
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}
	return nil
}

func initConfig(args []string, in io.Reader, out io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	for len(args) > 0 {
		if args[0] == "--help" || args[0] == "-h" {
			printUsage(out)
			return nil
		}
		if len(args) < 2 {
			return fmt.Errorf("%s needs a value", args[0])
		}
		switch args[0] {
		case "--domain":
			cfg.Domain = args[1]
		case "--username":
			cfg.Username = args[1]
		default:
			return fmt.Errorf("unknown init option: %s", args[0])
		}
		args = args[2:]
	}

	if cfg.Domain == "" || cfg.Username == "" {
		if !isTerminal(in) {
			return errors.New("use 'clone init --domain <domain> --username <username>' outside an interactive terminal")
		}
		reader := bufio.NewReader(in)
		if cfg.Domain == "" {
			fmt.Fprint(out, "Default domain (for example github.com): ")
			value, err := reader.ReadString('\n')
			if err != nil && len(value) == 0 {
				return err
			}
			cfg.Domain = strings.TrimSpace(value)
		}
		if cfg.Username == "" {
			fmt.Fprint(out, "Default username: ")
			value, err := reader.ReadString('\n')
			if err != nil && len(value) == 0 {
				return err
			}
			cfg.Username = strings.TrimSpace(value)
		}
	}
	cfg.Domain = normalizeDomain(cfg.Domain)
	if err := saveConfig(cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Saved configuration to %s\n", configPath())
	return err
}

func updateConfig(args []string, out io.Writer) error {
	if len(args) != 2 {
		return errors.New("usage: clone update <domain|username> <value>")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	switch args[0] {
	case "domain":
		cfg.Domain = normalizeDomain(args[1])
	case "username":
		cfg.Username = args[1]
	default:
		return errors.New("setting must be 'domain' or 'username'")
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Updated %s in %s\n", args[0], configPath())
	return err
}

func isTerminal(in io.Reader) bool {
	file, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func normalizeDomain(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	return strings.TrimSuffix(value, "/")
}

func validateConfig(cfg config) error {
	if !validDomain(cfg.Domain) {
		return errors.New("domain must be a host name, without a scheme or path")
	}
	if !validComponent(cfg.Username) {
		return errors.New("username is invalid")
	}
	return nil
}

func validDomain(value string) bool {
	if value == "" || strings.ContainsAny(value, "/@ 	\n\r") || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return false
	}
	for _, char := range value {
		if !(char == '.' || char == '-' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}

func validComponent(value string) bool {
	return value != "" &&
		!strings.ContainsAny(value, "/ 	\n\r") &&
		!strings.HasPrefix(value, ".") &&
		!strings.HasPrefix(value, "-")
}

func parseCloneOptions(args []string) (cloneOptions, error) {
	var options cloneOptions
	for len(args) > 0 {
		switch args[0] {
		case "--ssh":
			options.ssh = true
		case "--background", "-b":
			options.background = true
		case "--into":
			if len(args) < 2 {
				return options, errors.New("--into needs a directory")
			}
			options.destination = args[1]
			args = args[1:]
		case "--":
			options.gitOptions = append(options.gitOptions, args[1:]...)
			return options, nil
		case "--help", "-h":
			options.help = true
			return options, nil
		default:
			if strings.HasPrefix(args[0], "-") {
				return options, fmt.Errorf("unknown option: %s (pass Git options after --)", args[0])
			}
			if options.target != "" {
				return options, errors.New("only one repository can be cloned at a time")
			}
			options.target = args[0]
		}
		args = args[1:]
	}
	if options.target == "" {
		return options, errors.New("a repository is required")
	}
	return options, nil
}

func resolveRepository(target string, cfg config, useSSH bool) (string, error) {
	if isExplicitRemote(target) {
		if useSSH {
			return "", errors.New("--ssh cannot be used with an explicit remote")
		}
		return target, nil
	}
	if err := validateConfig(cfg); err != nil {
		return "", errors.New("not configured; run 'clone init --domain <domain> --username <username>'")
	}

	parts := strings.Split(target, "/")
	var domain, owner, repository string
	switch len(parts) {
	case 1:
		domain, owner, repository = cfg.Domain, cfg.Username, parts[0]
	case 2:
		domain, owner, repository = cfg.Domain, parts[0], parts[1]
	case 3:
		domain, owner, repository = parts[0], parts[1], parts[2]
	default:
		return "", errors.New("repository must be repository, owner/repository, or domain/owner/repository")
	}
	domain = normalizeDomain(domain)
	if !validDomain(domain) {
		return "", errors.New("repository domain is invalid")
	}
	if !validComponent(owner) {
		return "", errors.New("repository owner is invalid")
	}
	if !validComponent(repository) {
		return "", errors.New("repository name is invalid")
	}

	if useSSH {
		if !strings.HasSuffix(repository, ".git") {
			repository += ".git"
		}
		return fmt.Sprintf("git@%s:%s/%s", domain, owner, repository), nil
	}
	return fmt.Sprintf("https://%s/%s/%s", domain, owner, repository), nil
}

func isExplicitRemote(target string) bool {
	return strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "http://") ||
		strings.HasPrefix(target, "ssh://") || strings.HasPrefix(target, "git@")
}

func cloneRepository(args []string, in io.Reader, out, errOut io.Writer) error {
	options, err := parseCloneOptions(args)
	if err != nil {
		return err
	}
	if options.help || options.target == "" {
		printUsage(out)
		return nil
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if !isExplicitRemote(options.target) && (cfg.Domain == "" || cfg.Username == "") {
		if !isTerminal(in) {
			return errors.New("not configured; run 'clone init --domain <domain> --username <username>'")
		}
		if err := initConfig(nil, in, out); err != nil {
			return err
		}
		cfg, err = loadConfig()
		if err != nil {
			return err
		}
	}
	remote, err := resolveRepository(options.target, cfg, options.ssh)
	if err != nil {
		return err
	}

	commandArgs := append([]string{"clone"}, options.gitOptions...)
	commandArgs = append(commandArgs, remote)
	if options.destination != "" {
		commandArgs = append(commandArgs, options.destination)
	}
	cmd := exec.Command("git", commandArgs...)

	if options.background {
		logPath := backgroundLogPath()
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return fmt.Errorf("create background log: %w", err)
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		if err := cmd.Start(); err != nil {
			logFile.Close()
			return fmt.Errorf("start git clone: %w", err)
		}
		pid := cmd.Process.Pid
		if err := cmd.Process.Release(); err != nil {
			logFile.Close()
			return err
		}
		logFile.Close()
		_, err = fmt.Fprintf(out, "Started clone (PID %d). Output: %s\n", pid, logPath)
		return err
	}

	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

func backgroundLogPath() string {
	return filepath.Join(".", "clone-"+time.Now().Format("20060102-150405")+".log")
}
