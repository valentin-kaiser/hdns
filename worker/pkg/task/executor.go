// Package task contains the action implementations and the executor that
// dispatches them.
package task

import (
	"context"
	"fmt"

	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns-worker/pkg/config"
)

// Payload carries the certificate material delivered by HDNS in the webhook body.
type Payload struct {
	Cert              string `json:"cert"`
	Chain             string `json:"chain"`
	Fullchain         string `json:"fullchain"`
	PrivateKey        string `json:"private_key"`
	CertificateFormat string `json:"certificate_format"`
	PKCS12            string `json:"pkcs12"`
	PKCS12Base64      string `json:"pkcs12_base64"`
}

// Execute runs every action defined on task sequentially.
// The first action that returns an error aborts the chain.
func Execute(ctx context.Context, task *config.TaskConfig, payload Payload) error {
	for i, action := range task.Actions {
		log.Info().Msgf("[WORKER] task %q: running action[%d] type=%s", task.Name, i, action.Type)

		var err error
		switch action.Type {
		case config.ActionCertSave:
			err = save(action, payload)
		case config.ActionServiceRestart:
			err = restartService(ctx, action)
		case config.ActionExec:
			err = run(ctx, action)
		default:
			// Should never happen – config.Load validates types.
			err = fmt.Errorf("unknown action type %q", action.Type)
		}

		if err != nil {
			return fmt.Errorf("task %q action[%d] (%s): %w", task.Name, i, action.Type, err)
		}

		log.Info().Msgf("[WORKER] task %q: action[%d] type=%s completed", task.Name, i, action.Type)
	}

	return nil
}
