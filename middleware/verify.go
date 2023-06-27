package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

func RequireSlackSignature(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				log.Printf("authmiddleware.RequireSlackSignature: io.ReadAll: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			secretsVerifier, err := slack.NewSecretsVerifier(r.Header, secret)
			if err != nil {
				log.Printf("authmiddleware.RequireSlackSignature: slack.NewSecretsVerifier: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if _, err := secretsVerifier.Write(b); err != nil {
				log.Printf("authmiddleware.RequireSlackSignature: secretsVerifier.Write: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err := secretsVerifier.Ensure(); err != nil {
				log.Printf("authmiddleware.RequireSlackSignature: secretsVerifier.Ensure: %v", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(b))
			next.ServeHTTP(w, r)
		})
	}
}
