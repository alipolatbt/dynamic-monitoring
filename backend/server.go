package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type TokenRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
}

type RequestConfig struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	Body        interface{}       `json:"body"`
	TokenHeader string            `json:"token_header"`
}

type APICheck struct {
	Name         string          `json:"name"`
	TokenRequest TokenRequest    `json:"token_request"`
	TokenPath    string          `json:"token_path"`
	Requests     []RequestConfig `json:"requests"`
	Interval     int             `json:"interval"` // seconds
	Timeout      int             `json:"timeout"`  // seconds
}

type CheckResult struct {
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	StatusCode int       `json:"status_code"`
	Duration   float64   `json:"duration"`
	Timestamp  time.Time `json:"timestamp"`
	Error      string    `json:"error,omitempty"`
}

var (
	checksMu sync.RWMutex
	checks   = make(map[string]APICheck)

	resultsMu  sync.RWMutex
	results    = make([]CheckResult, 0)
	maxResults = 200

	tokenMu    sync.RWMutex
	tokenCache = make(map[string]string)
)

func parseJSONPath(path string, doc map[string]interface{}) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty token path")
	}
	parts := strings.Split(path, ".")
	var cur interface{} = doc
	for _, p := range parts {
		if m, ok := cur.(map[string]interface{}); ok {
			cur = m[p]
		} else {
			return "", fmt.Errorf("invalid token path")
		}
	}
	if s, ok := cur.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("token value not a string")
}

func getToken(check APICheck) (string, error) {
	tokenMu.RLock()
	if t, ok := tokenCache[check.Name]; ok && t != "" {
		tokenMu.RUnlock()
		return t, nil
	}
	tokenMu.RUnlock()

	client := &http.Client{Timeout: time.Duration(check.Timeout) * time.Second}
	var body []byte
	if check.TokenRequest.Body != nil {
		if b, err := json.Marshal(check.TokenRequest.Body); err == nil {
			body = b
		}
	}
	req, err := http.NewRequest(check.TokenRequest.Method, check.TokenRequest.URL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	for k, v := range check.TokenRequest.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token request failed status %d", resp.StatusCode)
	}
	var doc map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", err
	}
	token, err := parseJSONPath(check.TokenPath, doc)
	if err != nil {
		return "", err
	}
	tokenMu.Lock()
	tokenCache[check.Name] = token
	tokenMu.Unlock()
	return token, nil
}

func saveResult(r CheckResult) {
	resultsMu.Lock()
	defer resultsMu.Unlock()
	results = append([]CheckResult{r}, results...)
	if len(results) > maxResults {
		results = results[:maxResults]
	}
}

func monitor(check APICheck) {
	for {
		token, err := getToken(check)
		if err != nil {
			saveResult(CheckResult{Name: check.Name, Error: err.Error(), Timestamp: time.Now()})
			time.Sleep(time.Duration(check.Interval) * time.Second)
			continue
		}
		client := &http.Client{Timeout: time.Duration(check.Timeout) * time.Second}
		for _, rc := range check.Requests {
			start := time.Now()
			var body []byte
			if rc.Body != nil {
				if b, err := json.Marshal(rc.Body); err == nil {
					body = b
				}
			}
			req, err := http.NewRequest(rc.Method, rc.URL, bytes.NewBuffer(body))
			if err != nil {
				saveResult(CheckResult{Name: check.Name, URL: rc.URL, Error: err.Error(), Timestamp: time.Now()})
				continue
			}
			for k, v := range rc.Headers {
				req.Header.Set(k, v)
			}
			if rc.TokenHeader != "" {
				req.Header.Set(rc.TokenHeader, token)
			}
			resp, err := client.Do(req)
			duration := time.Since(start).Seconds()
			res := CheckResult{Name: check.Name, URL: rc.URL, Duration: duration, Timestamp: time.Now()}
			if err != nil {
				res.Error = err.Error()
			} else {
				res.StatusCode = resp.StatusCode
				resp.Body.Close()
			}
			saveResult(res)
		}
		time.Sleep(time.Duration(check.Interval) * time.Second)
	}
}

func main() {
	r := gin.Default()

	r.POST("/api/checks", func(c *gin.Context) {
		var chk APICheck
		if err := c.BindJSON(&chk); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if chk.Interval <= 0 {
			chk.Interval = 60
		}
		if chk.Timeout <= 0 {
			chk.Timeout = 10
		}
		if chk.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		checksMu.Lock()
		checks[chk.Name] = chk
		checksMu.Unlock()
		go monitor(chk)
		c.JSON(http.StatusCreated, chk)
	})

	r.GET("/api/checks", func(c *gin.Context) {
		checksMu.RLock()
		copy := make(map[string]APICheck, len(checks))
		for k, v := range checks {
			copy[k] = v
		}
		checksMu.RUnlock()
		c.JSON(http.StatusOK, copy)
	})

	r.GET("/api/results", func(c *gin.Context) {
		resultsMu.RLock()
		copy := make([]CheckResult, len(results))
		copy = append(copy[:0], results...)
		resultsMu.RUnlock()
		c.JSON(http.StatusOK, copy)
	})

	log.Println("Starting backend on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
