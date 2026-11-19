package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	redisURL   string
	redisToken string
)

func init() {
	redisURL = os.Getenv("UPSTASH_REDIS_REST_URL")
	redisToken = os.Getenv("UPSTASH_REDIS_REST_TOKEN")
	if redisURL == "" || redisToken == "" {
		fmt.Println("WARNING: Upstash environment variables not set. App will not work.")
	}
}

func redisCmd(args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no command")
	}
	endpoint := redisURL + "/" + strings.ToLower(args[0])
	for _, arg := range args[1:] {
		endpoint += "/" + url.PathEscape(arg)
	}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+redisToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	// This unwraps the Upstash response
	var upstashResp struct {
		Result string `json:"result"`
	}
	json.Unmarshal(body, &upstashResp)
	
	if upstashResp.Result != "" {
		return upstashResp.Result, nil
	}

	return string(body), nil
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	u, err := url.ParseRequestURI(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "Invalid URL (must start with http:// or https://)", http.StatusBadRequest)
		return
	}
	code := generateCode(7)
	_, err = redisCmd("SET", code, req.URL)
	if err != nil {
		http.Error(w, "Storage error", http.StatusInternalServerError)
		return
	}
	scheme := "https"
	host := r.Host
	shortURL := scheme + "://" + host + "/" + code
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"short": shortURL})
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}
	longURL, err := redisCmd("GET", code)
	if err != nil || longURL == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	clicks, _ := redisCmd("GET", "clicks:"+code)
	if clicks == "" {
		clicks = "0"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"long":   longURL,
		"clicks": clicks,
	})
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")
	longURL, err := redisCmd("GET", code)
	if err != nil || longURL == "" {
		http.NotFound(w, r)
		return
	}
	redisCmd("INCR", "clicks:"+code)
	http.Redirect(w, r, longURL, http.StatusFound)
}

func generateCode(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if len(path) == 8 && path[0] == '/' && isAlnum(path[1:]) {
		redirectHandler(w, r)
		return
	}
	http.FileServer(http.Dir("public")).ServeHTTP(w, r)
}

func isAlnum(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/shorten", shortenHandler)
	mux.HandleFunc("/api/stats", statsHandler)
	mux.HandleFunc("/", mainHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	fmt.Println("Listening on :" + port)
	http.ListenAndServe(":"+port, mux)
}
