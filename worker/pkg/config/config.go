// Package config manages the worker configuration via the go-core config manager.
// The configuration is stored at <data-dir>/hdns-worker.yaml and is read once
// at startup. Call Init() early (e.g. from main init()), then use Get() to
// retrieve the current configuration.
package config

import (
	"fmt"
	"strings"

	"github.com/valentin-kaiser/go-core/config"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/go-core/security"
)

// ActionType identifies which action to perform.
type ActionType string

const (
	// ActionCertSave writes the certificate and private key to a directory.
	ActionCertSave ActionType = "cert_save"
	// ActionServiceRestart restarts a named OS service.
	ActionServiceRestart ActionType = "service_restart"
	// ActionExec runs an arbitrary shell command.
	ActionExec ActionType = "exec"
)

// ActionConfig describes a single action inside a task.
type ActionConfig struct {
	// Type selects the action implementation.
	Type ActionType `yaml:"type" usage:"action type: cert_save | service_restart | exec"`

	// --- cert_save fields ---

	// CertDir is the directory to write the certificate files into.
	CertDir string `yaml:"cert_dir" usage:"(cert_save) directory to write certificate files"`
	// CertFile is the filename for the leaf certificate only (cert.pem content).
	// Leave empty to skip writing this file.
	CertFile string `yaml:"cert_file" usage:"(cert_save) filename for the leaf certificate (omit to skip)"`
	// ChainFile is the filename for the intermediate certificate(s) only (chain.pem content).
	// Leave empty to skip writing this file.
	ChainFile string `yaml:"chain_file" usage:"(cert_save) filename for the intermediate certificate(s) (omit to skip)"`
	// FullchainFile is the filename for the full chain – leaf + intermediates (fullchain.pem content).
	// Leave empty to skip writing this file.
	FullchainFile string `yaml:"fullchain_file" usage:"(cert_save) filename for the full chain cert+intermediates (omit to skip)"`
	// KeyFile is the filename for the private key (privkey.pem content).
	// Leave empty to skip writing this file.
	KeyFile string `yaml:"key_file" usage:"(cert_save) filename for the private key (omit to skip)"`
	// PKCS12File is the filename for the PKCS#12 archive payload (pkcs12 / pkcs12_base64).
	// Leave empty to skip writing this file.
	PKCS12File string `yaml:"pkcs12_file" usage:"(cert_save) filename for the PKCS#12 archive (omit to skip)"`

	// --- service_restart fields ---

	// ServiceName is the OS service name to restart.
	ServiceName string `yaml:"service_name" usage:"(service_restart) OS service name to restart"`

	// --- exec fields ---

	// Command is the command string to execute (passed to the system shell).
	Command string `yaml:"command" usage:"(exec) command to run via cmd /C (Windows) or sh -c (Linux)"`
}

// TaskConfig groups a set of actions under a unique name.
// The name maps directly to the webhook URL path segment
// (POST /<name>).
type TaskConfig struct {
	// Name uniquely identifies this task.
	Name string `yaml:"name" usage:"unique task identifier; used as the URL path segment /webhook/<name>"`
	// Actions are executed sequentially; the first failure aborts the chain.
	Actions []ActionConfig `yaml:"actions" usage:"list of actions to execute in order"`
}

// WorkerConfig is the root configuration structure.
// It is registered with the go-core config manager under the name "hdns-worker"
// and persisted to <data-dir>/hdns-worker.yaml.
type WorkerConfig struct {
	// LogLevel controls the zerolog log level.
	LogLevel int64 `yaml:"log_level" usage:"logging level (0=debug, 1=info, 2=warn, 3=error, 4=fatal)"`
	// Port is the HTTP port the worker listens on.
	Port uint16 `yaml:"port" usage:"HTTP port the webhook server listens on"`
	// Secret is the shared Bearer token required in every webhook request.
	// Requests without a matching Authorization header are rejected with 401.
	Secret string `yaml:"secret" usage:"shared Bearer token; every request must send 'Authorization: Bearer <secret>'"`
	// Tasks is the list of named webhook tasks.
	Tasks []*TaskConfig `yaml:"tasks" usage:"list of named webhook tasks"`
}

// Validate is called by the go-core config manager after reading the YAML file.
// It enforces required fields and sets defaults for optional ones.
// An empty secret is allowed here — Init() generates and persists one automatically.
func (c *WorkerConfig) Validate() error {
	if c.Port == 0 {
		return fmt.Errorf("config: port must be a valid TCP port number")
	}

	names := make(map[string]struct{}, len(c.Tasks))
	for i, t := range c.Tasks {
		if strings.TrimSpace(t.Name) == "" {
			return fmt.Errorf("config: task[%d]: name must not be empty", i)
		}
		if _, dup := names[t.Name]; dup {
			return fmt.Errorf("config: task %q is defined more than once", t.Name)
		}
		names[t.Name] = struct{}{}

		if len(t.Actions) == 0 {
			return fmt.Errorf("config: task %q: must define at least one action", t.Name)
		}

		for j := range t.Actions {
			if err := t.Actions[j].defaults(t.Name, j); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyDefaults fills in optional fields and rejects invalid action types.
func (a *ActionConfig) defaults(taskName string, idx int) error {
	loc := fmt.Sprintf("config: task %q action[%d]", taskName, idx)
	switch a.Type {
	case ActionCertSave:
		if strings.TrimSpace(a.CertDir) == "" {
			return fmt.Errorf("%s (cert_save): cert_dir must not be empty", loc)
		}
		if a.CertFile == "" && a.ChainFile == "" && a.FullchainFile == "" && a.KeyFile == "" && a.PKCS12File == "" {
			return fmt.Errorf("%s (cert_save): at least one of cert_file, chain_file, fullchain_file, key_file, pkcs12_file must be set", loc)
		}
	case ActionServiceRestart:
		if strings.TrimSpace(a.ServiceName) == "" {
			return fmt.Errorf("%s (service_restart): service_name must not be empty", loc)
		}
	case ActionExec:
		if strings.TrimSpace(a.Command) == "" {
			return fmt.Errorf("%s (exec): command must not be empty", loc)
		}
	default:
		return fmt.Errorf("%s: unknown action type %q (valid: cert_save, service_restart, exec)", loc, a.Type)
	}
	return nil
}

// Init registers the worker configuration with the go-core config manager and
// reads it from disk. It must be called after flag.Init() has run (the go-core
// service package calls flag.Init via its own init(), so calling Init() from
// the application's init() is safe).
// If no secret is present in the config file a cryptographically random one is
// generated and written back to hdns-worker.yaml automatically.
func Init() {
	cfg := &WorkerConfig{
		Port:     8080,
		LogLevel: 1,
		Secret:   "",
		Tasks:    []*TaskConfig{},
	}

	err := config.Manager().WithName("hdns-worker").Register(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to register worker configuration")
	}

	flag.Init()

	err = config.Read()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read configuration from %s", flag.Path)
	}

	// Auto-generate a secure random secret on first run (or when left empty).
	if strings.TrimSpace(Get().Secret) == "" {
		generated, genErr := security.GetRandomBytesBase64(32)
		if genErr != nil {
			log.Fatal().Err(genErr).Msg("failed to generate worker secret")
		}

		updated := Get()
		updated.Secret = generated

		if writeErr := config.Write(updated); writeErr != nil {
			log.Fatal().Err(writeErr).Msg("failed to persist generated secret to config")
		}

		log.Info().Msgf("[WORKER] generated secret written to hdns-worker.yaml — configure this value as the Authorization Bearer token in HDNS")
	}
}

// Get returns the current WorkerConfig. Returns an empty config if the type
// assertion fails (should never happen after Init() succeeds).
func Get() *WorkerConfig {
	c, ok := config.Get().(*WorkerConfig)
	if !ok || c == nil {
		return &WorkerConfig{}
	}
	return c
}

// OnChange registers a callback that is invoked whenever the configuration is
// rewritten via config.Write(). Both the old and new values are provided.
func OnChange(f func(o *WorkerConfig, n *WorkerConfig) error) {
	config.OnChange(func(o config.Config, n config.Config) error {
		if o == nil || n == nil {
			return fmt.Errorf("config: OnChange received nil config")
		}
		oc, ok := o.(*WorkerConfig)
		if !ok {
			return fmt.Errorf("config: OnChange old value is not a *WorkerConfig")
		}
		nc, ok := n.(*WorkerConfig)
		if !ok {
			return fmt.Errorf("config: OnChange new value is not a *WorkerConfig")
		}
		return f(oc, nc)
	})
}

// TaskByName returns the TaskConfig with the given name, or nil if not found.
func (c *WorkerConfig) TaskByName(name string) *TaskConfig {
	for i := range c.Tasks {
		if c.Tasks[i].Name == name {
			return c.Tasks[i]
		}
	}
	return nil
}
