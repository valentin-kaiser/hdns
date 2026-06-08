// Package handler provides the HTTP handlers for incoming HDNS webhook calls.
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns-worker/pkg/config"
	"github.com/valentin-kaiser/hdns-worker/pkg/task"
)

// Task returns an http.HandlerFunc for a specific named task, registered at
// /<name>/. Authentication is checked on every request via config.Get() so
// a secret rotation written through config.Write() takes effect immediately.
func Task(taskCfg *config.TaskConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// --- authentication ---
		if !isAuthorized(r, config.Get().Secret) {
			log.Warn().Msgf("[WORKER] unauthorized call to /%s/ from %s", taskCfg.Name, r.RemoteAddr)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// --- payload decoding ---
		var payload task.Payload
		if r.ContentLength != 0 {
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4*1024*1024))
			if err := dec.Decode(&payload); err != nil {
				log.Error().Err(err).Msgf("[WORKER] failed to decode webhook body for task %q", taskCfg.Name)
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
		}

		// --- execution ---
		log.Info().Msgf("[WORKER] executing task %q (triggered by %s)", taskCfg.Name, r.RemoteAddr)
		if err := task.Execute(r.Context(), taskCfg, payload); err != nil {
			log.Error().Err(err).Msgf("[WORKER] task %q failed", taskCfg.Name)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// isAuthorized checks whether the request carries the expected Bearer token.
func isAuthorized(r *http.Request, secret string) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}

	token := strings.TrimPrefix(auth, prefix)
	// Constant-time comparison to prevent timing attacks.
	return secureEqual(token, secret)
}

// secureEqual compares two strings in constant time relative to the length of
// the expected secret to prevent timing-based secret enumeration.
func secureEqual(provided, expected string) bool {
	if len(provided) != len(expected) {
		return false
	}
	var diff byte
	for i := range provided {
		diff |= provided[i] ^ expected[i]
	}
	return diff == 0
}
