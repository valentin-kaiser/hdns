package generate

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
func IsGenerateTaskCommand() bool {
	for _, arg := range os.Args[1:] {
		if arg == "generate-task" {
			return true
		}
	}
	return false
}

// runGenerateTask runs an interactive wizard that produces a ready-to-paste
// task YAML snippet for hdns-worker.yaml.
func RunGenerateTask() {
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
		fmt.Println("    fortios_upload  — upload PKCS#12 certificate to FortiGate API")
		fmt.Println("    fortios_profile_cert_replace     — replace cert references in FortiOS profiles")
		fmt.Println("    fortios_admin_server_cert_update — set system/global admin-server-cert")

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
			case config.ActionFortiOSUpload:
				action = wizardFortiOS(scanner)
			case config.ActionFortiOSProfileCertReplace:
				action = wizardFortiOSProfileCertReplace(scanner)
			case config.ActionFortiOSAdminServerCertUpdate:
				action = wizardFortiOSAdminServerCertUpdate(scanner)
			default:
				fmt.Fprintf(os.Stderr, "  Unknown type %q. Valid values: cert_save, service_restart, exec, fortios_upload, fortios_profile_cert_replace, fortios_admin_server_cert_update\n", t)
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
	a.PKCS12File = optPrompt(scanner, "  pkcs12_file    — PKCS#12 archive filename (e.g. cert.p12) [optional]: ")

	fmt.Println()
	fmt.Println("  combined_files — concatenated PEM bundles (e.g. key+fullchain for HAProxy).")
	fmt.Println("  Valid parts: cert, chain, fullchain, key")
	for i := 0; ; i++ {
		more := optPrompt(scanner, fmt.Sprintf("  Add combined file %d? [y/N]: ", i+1))
		if !strings.EqualFold(more, "y") {
			break
		}
		var cf config.CombinedFileConfig
		cf.Filename = mustPrompt(scanner, "    filename (e.g. haproxy.pem): ")
		for {
			raw := mustPrompt(scanner, "    parts (comma-separated, e.g. key,fullchain): ")
			for _, p := range strings.Split(raw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					cf.Parts = append(cf.Parts, p)
				}
			}
			if len(cf.Parts) > 0 {
				break
			}
			fmt.Fprintln(os.Stderr, "    (at least one part is required)")
		}
		a.CombinedFiles = append(a.CombinedFiles, cf)
	}
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

// wizardFortiOS collects all fields for a fortios_upload action.
func wizardFortiOS(scanner *bufio.Scanner) config.ActionConfig {
	a := config.ActionConfig{Type: config.ActionFortiOSUpload}
	a.FortiOS = promptFortiOSBase(scanner)
	return a
}

// wizardFortiOSProfileCertReplace collects fields for fortios_profile_cert_replace.
func wizardFortiOSProfileCertReplace(scanner *bufio.Scanner) config.ActionConfig {
	a := config.ActionConfig{Type: config.ActionFortiOSProfileCertReplace}
	a.FortiOS = promptFortiOSBase(scanner)
	for i := 0; ; i++ {
		fmt.Println()
		if i == 0 {
			fmt.Println("  Define profile update 1.")
		} else {
			fmt.Printf("  Define profile update %d.\n", i+1)
		}
		u := config.FortiOSProfileUpdate{}
		u.Path = mustPrompt(scanner, "    path   — CMDB path under /api/v2/cmdb (e.g. vpn.ssl.settings): ")
		u.MKey = optPrompt(scanner, "    mkey   — optional object key (leave empty for singleton path): ")
		u.Field = mustPrompt(scanner, "    field  — field name to set to certname (e.g. servercert): ")
		u.Method = strings.ToUpper(optPrompt(scanner, "    method — PUT/PATCH/POST [default: PUT]: "))
		if u.Method == "" {
			u.Method = "PUT"
		}
		a.ProfileUpdates = append(a.ProfileUpdates, u)
		more := optPrompt(scanner, "  Add another profile update? [y/N]: ")
		if !strings.EqualFold(more, "y") {
			break
		}
	}
	return a
}

// wizardFortiOSAdminServerCertUpdate collects fields for fortios_admin_server_cert_update.
func wizardFortiOSAdminServerCertUpdate(scanner *bufio.Scanner) config.ActionConfig {
	a := config.ActionConfig{Type: config.ActionFortiOSAdminServerCertUpdate}
	a.FortiOS = promptFortiOSBase(scanner)
	return a
}

func promptFortiOSBase(scanner *bufio.Scanner) *config.FortiOSConfig {
	cfg := &config.FortiOSConfig{}
	cfg.Host = mustPrompt(scanner, "  fortios.host         — FortiGate host[:port] or URL (e.g. fortigate.local): ")
	cfg.AccessToken = mustPrompt(scanner, "  fortios.access_token — FortiGate API token: ")
	cfg.CertName = mustPrompt(scanner, "  fortios.certname     — certificate name in FortiOS (e.g. api_crt): ")
	scope := optPrompt(scanner, "  fortios.scope        — vdom or global [default: vdom]: ")
	if scope == "" {
		scope = "vdom"
	}
	cfg.Scope = scope
	insecure := optPrompt(scanner, "  fortios.tls_insecure — skip TLS verify for self-signed certs? [y/N]: ")
	cfg.TLSInsecure = strings.EqualFold(insecure, "y")
	dryRun := optPrompt(scanner, "  fortios.dry_run      — only verify API/TLS/auth connectivity? [y/N]: ")
	cfg.DryRun = strings.EqualFold(dryRun, "y")
	return cfg
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
			if a.PKCS12File != "" {
				fmt.Printf("        pkcs12_file: %q\n", a.PKCS12File)
			}
			if len(a.CombinedFiles) > 0 {
				fmt.Println("        combined_files:")
				for _, cf := range a.CombinedFiles {
					fmt.Printf("          - filename: %q\n", cf.Filename)
					fmt.Printf("            parts: [%s]\n", strings.Join(cf.Parts, ", "))
				}
			}
		case config.ActionServiceRestart:
			fmt.Printf("        service_name: %q\n", a.ServiceName)
		case config.ActionExec:
			fmt.Printf("        command: %q\n", a.Command)
		case config.ActionFortiOSUpload:
			if a.FortiOS == nil {
				continue
			}
			printFortiOSYAML(a.FortiOS)
		case config.ActionFortiOSProfileCertReplace:
			if a.FortiOS == nil {
				continue
			}
			printFortiOSYAML(a.FortiOS)
			fmt.Println("        profile_updates:")
			for _, u := range a.ProfileUpdates {
				fmt.Printf("          - path: %q\n", u.Path)
				if u.MKey != "" {
					fmt.Printf("            mkey: %q\n", u.MKey)
				}
				fmt.Printf("            field: %q\n", u.Field)
				if u.Method != "" {
					fmt.Printf("            method: %q\n", u.Method)
				}
			}
		case config.ActionFortiOSAdminServerCertUpdate:
			if a.FortiOS == nil {
				continue
			}
			printFortiOSYAML(a.FortiOS)
		}
	}
}

func printFortiOSYAML(f *config.FortiOSConfig) {
	fmt.Println("        fortios:")
	fmt.Printf("          host: %q\n", f.Host)
	fmt.Printf("          access_token: %q\n", f.AccessToken)
	fmt.Printf("          certname: %q\n", f.CertName)
	if f.Scope != "" {
		fmt.Printf("          scope: %q\n", f.Scope)
	}
	if f.TLSInsecure {
		fmt.Printf("          tls_insecure: %t\n", f.TLSInsecure)
	}
	if f.DryRun {
		fmt.Printf("          dry_run: %t\n", f.DryRun)
	}
}
