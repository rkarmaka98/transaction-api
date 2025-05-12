// A simple HTTP seerver keep account balances in
// a map[string]float64 balances protected by sync.Mutex
// to avoid concurrent access issues.

// GET /balance/{account} return accounts balance
// POST /transfer moves funds between accounts with validation

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// In-memory store
var (
	// protects balances ensure only one
	// coroutine can access at a time
	// maps in go are not safe for concurrent access
	// without a mutex to avoid race conditions
	mu       sync.Mutex
	balances = map[string]float64{
		"alice": 100,
		"bob":   50,
	}
)

// models the JSON body for POST /transfer
type transferRequest struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

func main() {
	// Register handler function and listen on port
	http.HandleFunc("/balance/", balanceHandler)
	http.HandleFunc("/transfer", transferHandler)
	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)
}

// handles GET /balance/{account} to read account balance
func balanceHandler(w http.ResponseWriter, r *http.Request) {
	account := r.URL.Path[len("/balance/"):]
	// blocks until safe to access the map
	mu.Lock()
	bal, ok := balances[account]
	// after reading balance unlock so coroutines no longer blocked
	mu.Unlock()

	if !ok {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, `{"account":"%s","balance":%.2f}`, account, bal)
}

// handles POST /transfer all other get 405
func transferHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST request allowed", http.StatusMethodNotAllowed)
		return
	}

	var req transferRequest

	// Reads and parses POST body into transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.Amount <= 0 {
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	// lock the store then defer ensures any return from
	// this function first unlocks the mutex avoiding deadlocks
	mu.Lock()
	defer mu.Unlock()
	if balances[req.From] < req.Amount {
		http.Error(w, "insufficient funds", http.StatusUnprocessableEntity)
		return
	}
	balances[req.From] -= req.Amount
	balances[req.To] += req.Amount

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "ok"}`)
}
