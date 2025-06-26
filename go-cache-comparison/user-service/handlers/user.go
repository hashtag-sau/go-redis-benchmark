package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"user-service/cache"
	"user-service/db"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id", 400)
		return
	}

	val, err := cache.Get(id) // Try to get from cache

	if err == nil {
		fmt.Fprintf(w, "From Cache: %s\n", val)
		return
	}

	row := db.DB.QueryRow("SELECT name, email FROM users WHERE id = ?", id)
	var user User
	user.ID = 0
	if err := row.Scan(&user.Name, &user.Email); err != nil {
		http.Error(w, "User not found", 404)
		return
	}

	user.ID = 0
	res, _ := json.Marshal(user)
	_ = cache.Set(id, string(res), 10*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}