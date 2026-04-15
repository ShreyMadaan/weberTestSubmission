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

type captureRequest struct {
InvoiceID string `json:"invoiceid"`
BidderID  string `json:"bidderid"`
Amount    int64  `json:"amountcents"`
Currency  string `json:"currency"`
Description string `json:"description"`
}

type captureResponse struct {
PaymentRef   string `json:"paymentref,omitempty"`
Status       string `json:"status"`
DeclineReason string `json:"declinereason,omitempty"`
}

func main() {
port := getenv("PAYMENT_GATEWAY_PORT", "8081")
declineRateStr := getenv("PAYMENT_DECLINE_RATE", "0.1")

declineRate, err := strconv.ParseFloat(declineRateStr, 64)
if err != nil {
log.Fatalf("invalid PAYMENT_DECLINE_RATE: %v", err)
}

mux := http.NewServeMux()
mux.HandleFunc("/capture", func(w http.ResponseWriter, r *http.Request) {
if r.Method != http.MethodPost {
http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
return
}

auth := r.Header.Get("Authorization")
if auth == "" {
http.Error(w, "missing Authorization header", http.StatusUnauthorized)
return
}

var req captureRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "invalid json", http.StatusBadRequest)
return
}

rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
if rnd.Float64() < declineRate {
resp := captureResponse{
Status:       "declined",
DeclineReason: "insufficientfunds",
}
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusPaymentRequired)
_ = json.NewEncoder(w).Encode(resp)
return
}

ref := "PAY-" + strconv.FormatInt(time.Now().UnixNano(), 10)
resp := captureResponse{
PaymentRef: ref,
Status:     "approved",
}
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
_ = json.NewEncoder(w).Encode(resp)
})

server := &http.Server{
Addr:    ":" + port,
Handler: mux,
}

log.Printf("payment gateway mock listening on :%s", port)
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
