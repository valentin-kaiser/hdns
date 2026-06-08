package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/valentin-kaiser/hdns-worker/pkg/config"
)

// isGenerateTaskCommand returns true when the user passed "generate-task" as a
// positional argument, before any flag parsing occurs. This is used by init()
// to skip service / config initialisation for this purely interactive command.
func isGenerateTaskCommand() bool {
	for _, arg := range os.Args[1:] {
		if arg == "generate-task" {
			return true
		}
	}
	return false
}

// runGenerateTask runs an interactive wizard that produces a ready-to-paste
// task YAML snippet for hdns-worker.yaml.
func runGenerateTask() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("HDNS Worker — Task Configuration Generator")
	fmt.Println("===========================================")
	fmt.Println("Answer the prompts to build a task entry for hdns-worker.yaml.")
	fmt.Println()

	task := &config.TaskConfig{}
	task.Name = mustPrompt(scanner, "Task name (webhook URL path segment, e.g. renewal): ")

	for i := 0; ; i++ {
		fmt.Println()
		if i == 0 {
			fmt.Println("Define action 1.")
		} else {
			fmt.Printf("Define action %d.\n", i+1)
		}
		fmt.Println("  Available types:")
		fmt.Println("    cert_save       — write certificate / key files to disk")
		fmt.Println("    service_restart — restart an OS service")
		fmt.Println("    exec            — run an arbitrary shell command")

		var action config.ActionConfig
		for {
			t := mustPrompt(scanner, "  Action type: ")
			switch config.ActionType(t) {
			case config.ActionCertSave:
				action = wizardCertSave(scanner)
			case config.ActionServiceRestart:
				action = wizardServiceRestart(scanner)
			case config.ActionExec:
				action = wizardExec(scanner)
			default:
				fmt.Fprintf(os.Stderr, "  Unknown type %q. Valid values: cert_save, service_restart, exec\n", t)
				continue
			}
			break
		}

		task.Actions = append(task.Actions, action)

		more := optPrompt(scanner, "\nAdd another action? [y/N]: ")
		if !strings.EqualFold(more, "y") {
			break
		}
	}

	fmt.Println()
	fmt.Println("=== Add the following to the 'tasks:' section of hdns-worker.yaml ===")
	fmt.Println()
	printTaskYAML(task)
}

// wizardCertSave collects all fields for a cert_save action.
func wizardCertSave(scanner *bufio.Scanner) config.ActionConfig {
	a := config.ActionConfig{Type: config.ActionCertSave}
	fmt.Println("  (Leave optional filenames empty to skip writing that file.)")
	a.CertDir = mustPrompt(scanner, "  cert_dir       — directory to write files (e.g. /etc/certs/example.com): ")
	a.CertFile = optPrompt(scanner, "  cert_file      — leaf certificate filename (e.g. cert.pem) [optional]: ")
	a.ChainFile = optPrompt(scanner, "  chain_file     — intermediate chain filename (e.g. chain.pem) [optional]: ")
	a.FullchainFile = optPrompt(scanner, "  fullchain_file — full chain filename (e.g. fullchain.pem) [optional]: ")
	a.KeyFile = optPrompt(scanner, "  key_file       — private key filename (e.g. privkey.pem) [optional]: ")
	return a
}

// wizardServiceRestart collects all fields for a service_restart action.
func wizardServiceRestart(scanner *bufio.Scanner) config.ActionConfig {
	a := config.ActionConfig{Type: config.ActionServiceRestart}
	a.ServiceName = mustPrompt(scanner, "  service_name — OS service name (e.g. nginx or \"My Web Service\"): ")
	return a
}

// wizardExec collects all fields for an exec action.
func wizardExec(scanner *bufio.Scanner) config.ActionConfig {
	a := config.ActionConfig{Type: config.ActionExec}
	a.Command = mustPrompt(scanner, "  command — shell command to run (e.g. systemctl reload nginx): ")
	return a
}

// optPrompt writes msg and returns the trimmed value; empty input is accepted.
func optPrompt(scanner *bufio.Scanner, msg string) string {
	fmt.Print(msg)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

// mustPrompt is like optPrompt but repeats until a non-empty value is entered.
func mustPrompt(scanner *bufio.Scanner, msg string) string {
	for {
		v := optPrompt(scanner, msg)
		if v != "" {
			return v
		}
		fmt.Fprintln(os.Stderr, "  (this field is required)")
	}
}

// printTaskYAML writes the task as a YAML snippet to stdout.
// Only action fields relevant to the chosen type are emitted.
func printTaskYAML(task *config.TaskConfig) {
	fmt.Printf("  - name: %q\n", task.Name)
	fmt.Println("    actions:")
	for _, a := range task.Actions {
		fmt.Printf("      - type: %q\n", string(a.Type))
		switch a.Type {
		case config.ActionCertSave:
			fmt.Printf("        cert_dir: %q\n", a.CertDir)
			if a.CertFile != "" {
				fmt.Printf("        cert_file: %q\n", a.CertFile)
			}
			if a.ChainFile != "" {
				fmt.Printf("        chain_file: %q\n", a.ChainFile)
			}
			if a.FullchainFile != "" {
				fmt.Printf("        fullchain_file: %q\n", a.FullchainFile)
			}
			if a.KeyFile != "" {
				fmt.Printf("        key_file: %q\n", a.KeyFile)
			}
		case config.ActionServiceRestart:
			fmt.Printf("        service_name: %q\n", a.ServiceName)
		case config.ActionExec:
			fmt.Printf("        command: %q\n", a.Command)
		}
	}
}
