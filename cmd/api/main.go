package main

import (
"context"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/your-username/auctioncore/internal/app"
)

func main() {
application, err := app.New()
if err != nil {
log.Fatalf("failed to initialize app: %v", err)
}

server := &http.Server{
Addr:              ":" + application.Config.HTTPPort,
Handler:           application.Router,
ReadHeaderTimeout: 5 * time.Second,
}

go func() {
log.Printf("api listening on :%s", application.Config.HTTPPort)
if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("server error: %v", err)
}
}()

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
<-stop

log.Println("shutting down api server")

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
log.Printf("http shutdown error: %v", err)
}

application.Close()
log.Println("shutdown complete")
}
