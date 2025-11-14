package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
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

type SMTPSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"`
	Enabled  bool   `json:"enabled"`
}

var (
	checksMu sync.RWMutex
	checks   = make(map[string]APICheck)

	resultsMu  sync.RWMutex
	results    = make([]CheckResult, 0)
	maxResults = 200

	tokenMu    sync.RWMutex
	tokenCache = make(map[string]string)

	smtpMu       sync.RWMutex
	smtpSettings SMTPSettings
	smtpPathOnce sync.Once
)

func getSMTPSettingsPath() string {
	// Allow override via env var, default to /data/smtp_settings.json
	if p := os.Getenv("SMTP_SETTINGS_PATH"); p != "" {
		return p
	}
	return "/data/smtp_settings.json"
}

func loadSMTPSettings() {
	path := getSMTPSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		// no file yet; that's okay
		log.Printf("[INFO] SMTP settings file not found, will create on save: %s", path)
		return
	}
	var s SMTPSettings
	if err := json.Unmarshal(data, &s); err != nil {
		log.Printf("[ERROR] Failed to parse SMTP settings file: %v", err)
		return
	}
	smtpMu.Lock()
	smtpSettings = s
	smtpMu.Unlock()
	log.Printf("[INFO] SMTP settings loaded from %s", path)
}

func persistSMTPSettings() error {
	path := getSMTPSettingsPath()
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	smtpMu.RLock()
	s := smtpSettings
	smtpMu.RUnlock()
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	// 0600 since it may contain a password
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	log.Printf("[INFO] SMTP settings persisted to %s", path)
	return nil
}

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
			return "", fmt.Errorf("invalid token path at '%s'", p)
		}
	}

	// Try different types
	switch v := cur.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case int, int64:
		return fmt.Sprintf("%d", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	default:
		// If it's a complex object, try to marshal it as string
		if cur != nil {
			if b, err := json.Marshal(cur); err == nil {
				return string(b), nil
			}
		}
		return "", fmt.Errorf("token value type %T not convertible to string", cur)
	}
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
	log.Printf("[DEBUG] Token response for %s: %+v", check.Name, doc)
	token, err := parseJSONPath(check.TokenPath, doc)
	if err != nil {
		log.Printf("[ERROR] Failed to parse token path '%s': %v", check.TokenPath, err)
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

	// Send email alert if check failed
	if (r.Error != "" || r.StatusCode >= 400) && r.StatusCode != 0 {
		go sendEmailAlert(r)
	}
}

func sendEmailAlert(r CheckResult) {
	smtpMu.RLock()
	settings := smtpSettings
	smtpMu.RUnlock()

	if !settings.Enabled || settings.Host == "" || settings.To == "" {
		return
	}

	subject := fmt.Sprintf("🚨 API Monitor Alert: %s Failed", r.Name)
	body := fmt.Sprintf(`
API Monitoring Alert

Check Name: %s
URL: %s
Status Code: %d
Duration: %.3fs
Error: %s
Timestamp: %s

---
Doğuş Teknoloji API Monitoring
`, r.Name, r.URL, r.StatusCode, r.Duration, r.Error, r.Timestamp.Format("2006-01-02 15:04:05"))

	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", settings.From, settings.To, subject, body))

	auth := smtp.PlainAuth("", settings.Username, settings.Password, settings.Host)
	addr := fmt.Sprintf("%s:%d", settings.Host, settings.Port)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         settings.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		log.Printf("[ERROR] Failed to connect to SMTP server: %v", err)
		return
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, settings.Host)
	if err != nil {
		log.Printf("[ERROR] Failed to create SMTP client: %v", err)
		return
	}
	defer client.Quit()

	if err := client.Auth(auth); err != nil {
		log.Printf("[ERROR] SMTP auth failed: %v", err)
		return
	}

	if err := client.Mail(settings.From); err != nil {
		log.Printf("[ERROR] SMTP MAIL command failed: %v", err)
		return
	}

	if err := client.Rcpt(settings.To); err != nil {
		log.Printf("[ERROR] SMTP RCPT command failed: %v", err)
		return
	}

	w, err := client.Data()
	if err != nil {
		log.Printf("[ERROR] SMTP DATA command failed: %v", err)
		return
	}

	_, err = w.Write(msg)
	if err != nil {
		log.Printf("[ERROR] Failed to write email body: %v", err)
		return
	}

	err = w.Close()
	if err != nil {
		log.Printf("[ERROR] Failed to close email writer: %v", err)
		return
	}

	log.Printf("[INFO] Email alert sent for %s to %s", r.Name, settings.To)
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
	// Load persisted SMTP settings once at startup
	smtpPathOnce.Do(loadSMTPSettings)
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

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

	r.POST("/api/checks/:name/trigger", func(c *gin.Context) {
		name := c.Param("name")
		checksMu.RLock()
		chk, exists := checks[name]
		checksMu.RUnlock()
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "check not found"})
			return
		}
		go func() {
			token, err := getToken(chk)
			if err != nil {
				saveResult(CheckResult{Name: chk.Name, Error: err.Error(), Timestamp: time.Now()})
				return
			}
			client := &http.Client{Timeout: time.Duration(chk.Timeout) * time.Second}
			for _, rc := range chk.Requests {
				start := time.Now()
				var body []byte
				if rc.Body != nil {
					if b, err := json.Marshal(rc.Body); err == nil {
						body = b
					}
				}
				req, err := http.NewRequest(rc.Method, rc.URL, bytes.NewBuffer(body))
				if err != nil {
					saveResult(CheckResult{Name: chk.Name, URL: rc.URL, Error: err.Error(), Timestamp: time.Now()})
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
				res := CheckResult{Name: chk.Name, URL: rc.URL, Duration: duration, Timestamp: time.Now()}
				if err != nil {
					res.Error = err.Error()
				} else {
					res.StatusCode = resp.StatusCode
					resp.Body.Close()
				}
				saveResult(res)
			}
		}()
		c.JSON(http.StatusOK, gin.H{"message": "triggered"})
	})

	r.DELETE("/api/checks/:name", func(c *gin.Context) {
		name := c.Param("name")
		checksMu.Lock()
		if _, exists := checks[name]; !exists {
			checksMu.Unlock()
			c.JSON(http.StatusNotFound, gin.H{"error": "check not found"})
			return
		}
		delete(checks, name)
		checksMu.Unlock()

		// Also delete all results for this check
		resultsMu.Lock()
		filteredResults := []CheckResult{}
		for _, r := range results {
			if r.Name != name {
				filteredResults = append(filteredResults, r)
			}
		}
		results = filteredResults
		resultsMu.Unlock()

		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	})

	r.GET("/api/smtp", func(c *gin.Context) {
		smtpMu.RLock()
		settings := smtpSettings
		smtpMu.RUnlock()
		// Mask password
		settings.Password = "****"
		c.JSON(http.StatusOK, settings)
	})

	r.POST("/api/smtp", func(c *gin.Context) {
		var settings SMTPSettings
		if err := c.BindJSON(&settings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Preserve existing password if masked or empty comes from UI
		smtpMu.Lock()
		if settings.Password == "" || settings.Password == "****" {
			settings.Password = smtpSettings.Password
		}
		smtpSettings = settings
		smtpMu.Unlock()
		if err := persistSMTPSettings(); err != nil {
			log.Printf("[ERROR] Failed to persist SMTP settings: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "SMTP settings saved"})
	})

	log.Println("Starting backend on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
