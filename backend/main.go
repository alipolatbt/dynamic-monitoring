// Placeholder file kept to allow edits/history. No declarations here.
package main


package main

// main.go (placeholder) — real server implementation lives in server.go
package main

// Intentionally empty: server.go contains the implementation for package main.
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
    Name         string        `json:"name"`
    TokenRequest TokenRequest  `json:"token_request"`
    TokenPath    string        `json:"token_path"`
    Requests     []RequestConfig `json:"requests"`
    Interval     int           `json:"interval"`
    Timeout      int           `json:"timeout"`
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

    resultsMu sync.RWMutex
    results   = make([]CheckResult, 0)
    maxResults = 200

    tokenMu sync.RWMutex
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
        b, err := json.Marshal(check.TokenRequest.Body)
        if err == nil {
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
                if b, err := json.Marshal(rc.Body); err == nil { body = b }
            }
            req, err := http.NewRequest(rc.Method, rc.URL, bytes.NewBuffer(body))
            if err != nil {
                saveResult(CheckResult{Name: check.Name, URL: rc.URL, Error: err.Error(), Timestamp: time.Now()})
                continue
            }
            for k, v := range rc.Headers { req.Header.Set(k, v) }
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
        if chk.Interval <= 0 { chk.Interval = 60 }
        if chk.Timeout <= 0 { chk.Timeout = 10 }
        if chk.Name == "" { c.JSON(http.StatusBadRequest, gin.H{"error":"name required"}); return }
        checksMu.Lock()
        checks[chk.Name] = chk
        checksMu.Unlock()
        go monitor(chk)
        c.JSON(http.StatusCreated, chk)
    })

    r.GET("/api/checks", func(c *gin.Context) {
        checksMu.RLock()
        copy := make(map[string]APICheck, len(checks))
        for k,v := range checks { copy[k]=v }
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

type APICheck struct {
	Name         string          `json:"name"`
	TokenRequest TokenRequest    `json:"token_request"`
	TokenPath    string          `json:"token_path"` // JSON path to extract token
	Requests     []RequestConfig `json:"requests"`
	Interval     time.Duration   `json:"interval"`
	Timeout      time.Duration   `json:"timeout"`
}

type RequestConfig struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	Body        interface{}       `json:"body"`
	TokenHeader string            `json:"token_header"` // Header name for token (e.g. "Authorization")
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
	checks     = make(map[string]APICheck)
	results    = make([]CheckResult, 0)
	maxResults = 100
	tokenCache = make(map[string]string)
	tokenMutex sync.RWMutex
)

func getToken(check APICheck) (string, error) {
	tokenMutex.RLock()
	if token, exists := tokenCache[check.Name]; exists {
		tokenMutex.RUnlock()
		return token, nil
	}
	tokenMutex.RUnlock()

	client := &http.Client{
		Timeout: check.Timeout * time.Second,
	}

	var bodyBytes []byte
	if check.TokenRequest.Body != nil {
		var err error
		bodyBytes, err = json.Marshal(check.TokenRequest.Body)
		if err != nil {
			return "", fmt.Errorf("error marshaling token request body: %v", err)
		}
	}

	req, err := http.NewRequest(check.TokenRequest.Method, check.TokenRequest.URL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	for key, value := range check.TokenRequest.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	var tokenResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", err
	}

	// Extract token using the specified path
	parts := parseJSONPath(check.TokenPath)
	value := interface{}(tokenResponse)
	for _, part := range parts {
		if m, ok := value.(map[string]interface{}); ok {
			value = m[part]
		} else {
			return "", fmt.Errorf("invalid token path: %s", check.TokenPath)
		}
	}

	token, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("token is not a string")
	}

	tokenMutex.Lock()
	tokenCache[check.Name] = token
	tokenMutex.Unlock()

	return token, nil
}

func parseJSONPath(path string) []string {
	return strings.Split(strings.Trim(path, "."), ".")
}

func monitorAPI(check APICheck) {
	client := &http.Client{
		Timeout: check.Timeout * time.Second,
	}

	for {
		// Get token first
		token, err := getToken(check)
		if err != nil {
			result := CheckResult{
				Name:      check.Name,
				Error:     fmt.Sprintf("Token error: %v", err),
				Timestamp: time.Now(),
			}
			saveResult(result)
			time.Sleep(check.Interval * time.Second)
			continue
		}

		// Execute each request in the sequence
		for _, reqConfig := range check.Requests {
			start := time.Now()

			var bodyBytes []byte
			if reqConfig.Body != nil {
				var err error
				bodyBytes, err = json.Marshal(reqConfig.Body)
				if err != nil {
					saveResult(CheckResult{
						Name:      check.Name,
						URL:       reqConfig.URL,
						Error:     fmt.Sprintf("Body marshal error: %v", err),
						Timestamp: time.Now(),
					})
					continue
				}
			}

			req, err := http.NewRequest(reqConfig.Method, reqConfig.URL, bytes.NewBuffer(bodyBytes))
			if err != nil {
				saveResult(CheckResult{
					Name:      check.Name,
					URL:       reqConfig.URL,
					Error:     err.Error(),
					Timestamp: time.Now(),
				})
				continue
			}

			// Set headers
			for key, value := range reqConfig.Headers {
				req.Header.Set(key, value)
			}

			// Set token in header
			if reqConfig.TokenHeader != "" {
				req.Header.Set(reqConfig.TokenHeader, token)
			}

			resp, err := client.Do(req)
			duration := time.Since(start).Seconds()

			result := CheckResult{
				Name:      check.Name,
				URL:       reqConfig.URL,
				Duration:  duration,
				Timestamp: time.Now(),
			}

			if err != nil {
				result.Error = err.Error()
			} else {
				result.StatusCode = resp.StatusCode
				resp.Body.Close()
			}

			saveResult(result)
		}

		time.Sleep(check.Interval * time.Second)
	}
}

func saveResult(result CheckResult) {
	results = append([]CheckResult{result}, results...)
	if len(results) > maxResults {
		results = results[:maxResults]
	}
}

func main() {
	r := gin.Default()

	r.POST("/api/checks", func(c *gin.Context) {
		var check APICheck
		if err := c.BindJSON(&check); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if check.Interval < 1 {
			check.Interval = 60
		}
		if check.Timeout < 1 {
			check.Timeout = 10
		}

		checks[check.Name] = check
		go monitorAPI(check)

		c.JSON(http.StatusCreated, check)
	})

	r.GET("/api/checks", func(c *gin.Context) {
		c.JSON(http.StatusOK, checks)
	})

	r.GET("/api/results", func(c *gin.Context) {
		c.JSON(http.StatusOK, results)
	})

	fmt.Println("Server starting on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
