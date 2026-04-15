package main

import (
"encoding/json"
"log"
"math/rand"
"net/http"
"os"
"strconv"
"time"
)

type verifyResponse struct {
Eligible           bool   `json:"eligible"`
Tier               string `json:"tier"`
DepositRequiredCents int64 `json:"depositrequiredcents"`
}

func main() {
port := getenv("IDENTITY_VERIFICATION_PORT", "8082")

mux := http.NewServeMux()
mux.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
if r.Method != http.MethodGet {
http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
return
}

q := r.URL.Query()
bidderID := q.Get("bidderid")
auctionID := q.Get("auctionid")
if bidderID == "" || auctionID == "" {
http.Error(w, "missing bidderid or auctionid", http.StatusBadRequest)
return
}

rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
p := rnd.Float64()

var resp verifyResponse
switch {
case p < 0.6:
resp = verifyResponse{
Eligible:             true,
Tier:                 "basic",
DepositRequiredCents: 0,
}
case p < 0.9:
resp = verifyResponse{
Eligible:             true,
Tier:                 "verified",
DepositRequiredCents: 50000,
}
default:
resp = verifyResponse{
Eligible:             false,
Tier:                 "basic",
DepositRequiredCents: 0,
}
}

w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(resp)
})

server := &http.Server{
Addr:    ":" + port,
Handler: mux,
}

log.Printf("identity verification mock listening on :%s", port)
if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("server error: %v", err)
}
}

func getenv(key, def string) string {
if v := os.Getenv(key); v != "" {
return v
}
return def
}
