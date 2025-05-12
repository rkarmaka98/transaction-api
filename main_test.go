package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBalanceHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/balance/alice", nil)
	w := httptest.NewRecorder()
	balanceHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTransferHandler(t *testing.T) {
	// reset balances for test
	balances = map[string]float64{"alice": 100, "bob": 0}

	body := `{"from":"alice","to":"bob","amount":25}`
	// Lets me test handler without live server
	req := httptest.NewRequest("POST", "/transfer", strings.NewReader(body))
	w := httptest.NewRecorder()
	transferHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if balances["alice"] != 75 || balances["bob"] != 25 {
		t.Errorf("balances not updated correctly: %+v", balances)
	}
}
