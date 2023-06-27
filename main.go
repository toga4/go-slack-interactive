package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/slack-go/slack"
	"github.com/toga4/go-slack-interactive/configs"
	"github.com/toga4/go-slack-interactive/handlers"
	"github.com/toga4/go-slack-interactive/middleware"
)

func main() {
	ctx := context.Background()

	envConfig, err := configs.New(ctx)
	if err != nil {
		log.Fatalf("configs.New: %v", err)
	}

	slackClient := slack.New(envConfig.SlackOauthToken)
	res, err := slackClient.AuthTestContext(ctx)
	if err != nil {
		log.Fatalf("slackClient.AuthTestContext: %#v, %v", res, err)
	}
	log.Printf("slackClient.AuthTestContext: %#v", res)

	eventHandler := handlers.NewEvent(slackClient, res.UserID, envConfig.SlackChannelId)

	router := chi.NewRouter()
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	router.Route("/slack", func(r chi.Router) {
		r.Use(middleware.RequireSlackSignature(envConfig.SlackSigningSecret)) // リクエスト検証用のミドルウェアを設定
		r.Post("/events", eventHandler.HandleEvent)                           // Events API
		r.Handle("/interactions", nil)                                        // Interaction Components
	})

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	server := &http.Server{Addr: fmt.Sprintf(":%d", envConfig.Port), Handler: router}
	go func() {
		log.Printf("Server started on port %d", envConfig.Port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server.ListenAndServe: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("Server stopping gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Failed to stop server. err: %v", err)
	}

	log.Println("Server stopped.")
}
