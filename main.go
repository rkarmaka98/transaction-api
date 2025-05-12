// A simple HTTP seerver keep account balances in
// a map[string]float64 protected by sync.Mutex
// to avoid concurrent access issues.

// GET /balance/{account} return accounts balance
// POST /transfer moves funds between accounts with validation

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// In-memory store
var (
	// protects balances ensure only one 
	// coroutine can access at a time
	mu sync.Mutex   
	balances = map[string]float64{
		"alice": 100,
		"bob": 50,
	}
)

// models the JSON body for POST /transfer
type transferRequest struct {
	From string `json:"from"`
	To string `json:"to"`
	Amount float64 `json:"amount"`
}

func main() {
	
}

// handles GET /balance/{account}
func balanceHandler(w http.ResponseWriter, r *http.Request) {
	account := r.URL.Path[len("/balance/"):]
	mu.Lock()
	bal, ok := balances[account]
	mu.Unlock()

	if !ok {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, `{"account":"%s","balance":%.2f}`, account)
}

// handles POST /transfer
func transferHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST request allowed", http.StatusMethodNotAllowed)
		return
	}

	var req transferRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err!=nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.Amount <=0 {
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	if balances[req.From] < req.Amount {
		http.Error(w, "insufficient funds", http.StatusUnprocessableEntity)
		return
	}
	balances[req.From] -= req.Amount
	balances[req.To] += req.Amount

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, {"status": "ok"})
}
