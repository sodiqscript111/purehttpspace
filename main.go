package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type User struct {
	Name string `json:"name"`
}

var (
	userCache  = make(map[int]User)
	cacheMutex sync.RWMutex
	nextID     int
)

func main() {
	// -------- CLIENT PART --------
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", "https://randomuser.me/api/?results=5", nil)
	if err != nil {
		log.Fatal("request creation failed:", err)
	}
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("client request failed:", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("read body failed:", err)
	}

	fmt.Println("Fetched JSON from API:")
	fmt.Println(string(body))

	// -------- SERVER PART --------
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("POST /users", createUsers)
	mux.HandleFunc("GET /users/{id}", getUser)

	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello World")
}

func createUsers(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		log.Println("decode error:", err)
		return
	}

	if user.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	cacheMutex.Lock()
	nextID++
	userCache[nextID] = user
	cacheMutex.Unlock()

	// Return JSON response with new ID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := map[string]any{"id": nextID, "name": user.Name}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "could not encode response", http.StatusInternalServerError)
	}
}

func getUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	cacheMutex.RLock()
	user, ok := userCache[id]
	cacheMutex.RUnlock()

	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "could not encode user", http.StatusInternalServerError)
	}
}
