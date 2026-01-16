package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vault-sync/cmd"
	"vault-sync/internal/errors"
	"vault-sync/internal/logger"
)

func main() {
	// Set up context with cancellation for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Info("Received interrupt signal, shutting down gracefully...")
		cancel()
		
		// Give operations 5 seconds to finish gracefully
		time.Sleep(5 * time.Second)
		logger.Error("Forced shutdown after timeout")
		os.Exit(1)
	}()

	if err := cmd.Execute(); err != nil {
		handleError(err)
		os.Exit(1)
	}
}

func handleError(err error) {
	if vaultErr, ok := err.(*errors.VaultSyncError); ok {
		// Log structured error information
		logFields := []any{
			"op", vaultErr.Op,
			"error", vaultErr.Err.Error(),
			"file", vaultErr.File,
			"line", vaultErr.Line,
		}
		
		if vaultErr.Path != "" {
			logFields = append(logFields, "path", vaultErr.Path)
		}
		
		for k, v := range vaultErr.Context {
			logFields = append(logFields, k, v)
		}
		
		logger.Error("Operation failed", logFields...)
		
		// Print user-friendly error to stderr
		fmt.Fprintf(os.Stderr, "Error: %v\n", vaultErr)
	} else {
		// Handle other errors
		logger.Error("Unexpected error", "error", err.Error())
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}