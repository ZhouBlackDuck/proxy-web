package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zwforum/proxy-web/internal/api"
	"github.com/zwforum/proxy-web/internal/config"
	"github.com/zwforum/proxy-web/internal/process"
	"github.com/zwforum/proxy-web/internal/store"
)

// Build info, injected via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

func main() {
	fmt.Printf("Proxy WebUI %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize data directories
	if err := config.InitDirs(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init data dirs: %v\n", err)
		os.Exit(1)
	}

	// Initialize file store
	fileStore := store.NewFileStore(cfg.DataDir)

	// Initialize process manager
	pm := process.NewManager(cfg)

	// Start subconverter
	if err := pm.StartSubConverter(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start subconverter: %v\n", err)
		fmt.Fprintf(os.Stderr, "subconverter not started, subscription features may be unavailable\n")
	}

	// Start mihomo
	if err := pm.StartMihomo(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start mihomo: %v\n", err)
		fmt.Fprintf(os.Stderr, "mihomo not started, you may need to configure it via WebUI\n")
	}

	// Create HTTP server
	router := api.NewRouter(cfg, fileStore, pm)
	server := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}

	// Start HTTP server
	go func() {
		fmt.Printf("WebUI backend listening on :3000\n")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	fmt.Printf("Received signal %v, shutting down...\n", sig)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Stop HTTP server
	if err := server.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "HTTP server shutdown error: %v\n", err)
	}

	// Stop child processes
	pm.StopAll()

	fmt.Println("Server exited")
}
