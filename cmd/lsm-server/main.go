package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sshloktiwarii/lsm-tree-db/pkg/lsm"
	"github.com/sshloktiwarii/lsm-tree-db/pkg/web"
)

func main() {
	port := flag.Int("port", 5001, "port to run embedded web workbench")
	dir := flag.String("dir", "./data", "directory for database files")
	flag.Parse()

	log.Printf("[LSM-Engine] Initializing storage engine in %s...", *dir)
	opts := lsm.DefaultOptions(*dir)
	db, err := lsm.Open(opts)
	if err != nil {
		log.Fatalf("[LSM-Engine] FATAL: failed to open database: %v", err)
	}

	srv := web.NewServer(db, *port)

	// Graceful shutdown channel
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[LSM-Engine] Workbench live on http://localhost:%d", *port)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[LSM-Engine] Server error: %v", err)
		}
	}()

	<-stop
	log.Println("[LSM-Engine] Shutdown signal received. Draining connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = srv.Close()
	if err := db.Close(); err != nil {
		log.Printf("[LSM-Engine] Error closing database: %v", err)
	} else {
		log.Println("[LSM-Engine] Database flushed, closed cleanly. Bye!")
	}
	<-ctx.Done()
	fmt.Println("Exit.")
}
