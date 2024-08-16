package proxy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgeflare/fabric-oidc-proxy/internal/config"
	"github.com/edgeflare/pgo"
	mw "github.com/edgeflare/pgo/middleware"
	"go.uber.org/zap"
)

var (
	cfg config.Config
)

func StartServer(conf *config.Config, logger *zap.Logger) error {
	// TODO: package scoped config
	cfg = *conf

	// Create a new pgo Router
	r := pgo.NewRouter()

	// middleware
	r.Use(mw.RequestID)
	r.Use(mw.CORSWithOptions(nil)) // TODO: improve this
	r.Use(mw.LoggerWithOptions(&mw.LoggerOptions{Logger: logger}))

	// OIDC middleware for authentication
	oidcConfig := mw.OIDCProviderConfig{
		ClientID:     cfg.OIDC.ClientID,
		ClientSecret: cfg.OIDC.ClientSecret,
		Issuer:       cfg.OIDC.Issuer,
	}

	// API v1 routes
	apiv1 := r.Group("/api/v1")
	apiv1.Use(mw.VerifyOIDCToken(oidcConfig))

	apiv1.Handle("POST /account/enroll", http.HandlerFunc(enrollUserHandler))
	apiv1.Handle("POST /{channel}/{chaincode}/submit-transaction", http.HandlerFunc(submitTxHandler))

	// Set up signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start the server
	go func() {
		logger.Info("Starting server", zap.Int("port", cfg.HTTP.Port))
		if err := r.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTP.Port)); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Wait for SIGINT or SIGTERM
	<-stop

	logger.Info("Shutting down server...")

	// Create a deadline for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := r.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	logger.Info("Server gracefully stopped")
	return nil
}
