package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestGenerateUserToken(t *testing.T) {
	token1 := generateUserToken()
	token2 := generateUserToken()

	if len(token1) != 32 {
		t.Errorf("Expected token length of 32, got %d", len(token1))
	}

	if token1 == token2 {
		t.Error("Tokens should be unique")
	}

	if !isHexString(token1) {
		t.Error("Token should be a valid hex string")
	}
}

func TestParseUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  BrowserInfo
	}{
		{
			name:      "Chrome",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
			expected:  BrowserInfo{Browser: "Chrome", Version: "116.0.0.0"},
		},
		{
			name:      "Firefox",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/118.0",
			expected:  BrowserInfo{Browser: "Firefox", Version: "118.0"},
		},
		{
			name:      "Safari",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
			expected:  BrowserInfo{Browser: "Safari", Version: "16.6"},
		},
		{
			name:      "Edge",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.69",
			expected:  BrowserInfo{Browser: "Edge", Version: "116.0.1938.69"},
		},
		{
			name:      "Empty",
			userAgent: "",
			expected:  BrowserInfo{Browser: "unknown", Version: ""},
		},
		{
			name:      "Unknown",
			userAgent: "CustomBot/1.0",
			expected:  BrowserInfo{Browser: "other", Version: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUserAgent(tt.userAgent)
			if result.Browser != tt.expected.Browser || result.Version != tt.expected.Version {
				t.Errorf("parseUserAgent(%s) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestParseLanguage(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		expected       []string
	}{
		{
			name:           "Single language",
			acceptLanguage: "en-US",
			expected:       []string{"en-US"},
		},
		{
			name:           "Multiple languages",
			acceptLanguage: "en-US,en;q=0.9,fr;q=0.8",
			expected:       []string{"en-US", "en", "fr"},
		},
		{
			name:           "Empty",
			acceptLanguage: "",
			expected:       []string{},
		},
		{
			name:           "With quality values",
			acceptLanguage: "da, en-gb;q=0.8, en;q=0.7",
			expected:       []string{"da", "en-gb", "en"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLanguage(tt.acceptLanguage)
			if !slicesEqual(result, tt.expected) {
				t.Errorf("parseLanguage(%s) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		host     string
		expected string
	}{
		{"localhost:8080", "localhost"},
		{"example.com", "example.com"},
		{"sub.example.com:3000", "sub.example.com"},
		{"192.168.1.1:8080", "192.168.1.1"},
		{"", ""},
	}

	for _, tt := range tests {
		result := extractDomain(tt.host)
		if result != tt.expected {
			t.Errorf("extractDomain(%s) = %s, want %s", tt.host, result, tt.expected)
		}
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 198.51.100.2"},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "203.0.113.5"},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "203.0.113.5",
		},
		{
			name:       "RemoteAddr with port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1",
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := getClientIP(req)
			if result != tt.expectedIP {
				t.Errorf("getClientIP() = %s, want %s", result, tt.expectedIP)
			}
		})
	}
}

func TestPixelHandler(t *testing.T) {
	tracker := NewPixelTracker()

	tests := []struct {
		name           string
		path           string
		queryParams    string
		headers        map[string]string
		checkCookie    bool
		expectedStatus int
	}{
		{
			name:           "Basic pixel request",
			path:           "/pixel.gif",
			queryParams:    "",
			headers:        map[string]string{},
			checkCookie:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Pixel with query parameters",
			path:           "/pixel.gif",
			queryParams:    "?campaign=test&user_id=123",
			headers:        map[string]string{},
			checkCookie:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Pixel with referrer",
			path:        "/pixel.gif",
			queryParams: "",
			headers: map[string]string{
				"Referer": "https://example.com/page",
			},
			checkCookie:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Pixel with user agent",
			path:        "/pixel.gif",
			queryParams: "",
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/116.0.0.0",
			},
			checkCookie:    true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path+tt.queryParams, nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(tracker.PixelHandler)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			contentType := rr.Header().Get("Content-Type")
			if contentType != "image/gif" {
				t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "image/gif")
			}

			if tt.checkCookie {
				cookies := rr.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == tracker.config.CookieName {
						found = true
						if len(cookie.Value) != 32 {
							t.Errorf("Cookie value should be 32 characters, got %d", len(cookie.Value))
						}
						break
					}
				}
				if !found && !tracker.config.DisableCookies {
					t.Error("Expected tracking cookie to be set")
				}
			}

			body := rr.Body.Bytes()
			if len(body) != 43 {
				t.Errorf("Expected pixel size of 43 bytes, got %d", len(body))
			}
		})
	}
}

func TestStatsHandler(t *testing.T) {
	tracker := NewPixelTracker()

	req1 := httptest.NewRequest("GET", "/pixel.gif?test=1", nil)
	req1.Header.Set("User-Agent", "TestBot/1.0")
	rr1 := httptest.NewRecorder()
	tracker.PixelHandler(rr1, req1)

	time.Sleep(100 * time.Millisecond)

	req2 := httptest.NewRequest("GET", "/stats", nil)
	rr2 := httptest.NewRecorder()
	tracker.StatsHandler(rr2, req2)

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rr2.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
	}

	var data []TrackingData
	err := json.Unmarshal(rr2.Body.Bytes(), &data)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if len(data) < 1 {
		t.Error("Expected at least one tracking entry")
	} else {
		if data[0].Query["test"] != "1" {
			t.Error("Query parameter not captured correctly")
		}
	}
}

func TestTrackerWithCustomHandler(t *testing.T) {
	tracker := NewPixelTracker()

	handlerChan := make(chan *TrackingData, 1)

	tracker.Use(func(data *TrackingData) {
		// Create a copy to avoid race conditions
		dataCopy := *data
		handlerChan <- &dataCopy
	})

	req := httptest.NewRequest("GET", "/pixel.gif?custom=handler", nil)
	req.Header.Set("Referer", "https://test.com")
	rr := httptest.NewRecorder()

	tracker.PixelHandler(rr, req)

	select {
	case capturedData := <-handlerChan:
		if capturedData.Query["custom"] != "handler" {
			t.Error("Query parameter not passed to custom handler")
		}
		if capturedData.Referer != "https://test.com" {
			t.Error("Referer not passed to custom handler")
		}
	case <-time.After(1 * time.Second):
		t.Error("Custom handler was not called within timeout")
	}
}

func TestConcurrentRequests(t *testing.T) {
	tracker := NewPixelTracker()

	numRequests := 100
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/pixel.gif?id=%d", id), nil)
			rr := httptest.NewRecorder()
			tracker.PixelHandler(rr, req)
			done <- true
		}(i)
	}

	for i := 0; i < numRequests; i++ {
		<-done
	}

	time.Sleep(200 * time.Millisecond)

	data := tracker.GetTrackingData()
	if len(data) != numRequests {
		t.Errorf("Expected %d tracking entries, got %d", numRequests, len(data))
	}
}

func TestIntegrationWithRouter(t *testing.T) {
	tracker := NewPixelTracker()
	r := mux.NewRouter()
	r.HandleFunc("/pixel.gif", tracker.PixelHandler).Methods("GET", "HEAD")
	r.HandleFunc("/stats", tracker.StatsHandler).Methods("GET")

	testServer := httptest.NewServer(r)
	defer testServer.Close()

	resp1, err := http.Get(testServer.URL + "/pixel.gif?integration=test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp1.StatusCode)
	}

	body1, _ := io.ReadAll(resp1.Body)
	if len(body1) != 43 {
		t.Errorf("Expected pixel size of 43 bytes, got %d", len(body1))
	}

	time.Sleep(100 * time.Millisecond)

	resp2, err := http.Get(testServer.URL + "/stats")
	if err != nil {
		t.Fatalf("Failed to make stats request: %v", err)
	}
	defer resp2.Body.Close()

	var stats []TrackingData
	json.NewDecoder(resp2.Body).Decode(&stats)

	if len(stats) < 1 {
		t.Error("No tracking data recorded")
	} else {
		if stats[0].Query["integration"] != "test" {
			t.Error("Query parameter not tracked correctly")
		}
	}
}

func BenchmarkPixelHandler(b *testing.B) {
	tracker := NewPixelTracker()
	req := httptest.NewRequest("GET", "/pixel.gif", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		tracker.PixelHandler(rr, req)
	}
}

func BenchmarkParseUserAgent(b *testing.B) {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseUserAgent(userAgent)
	}
}

func BenchmarkGenerateUserToken(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateUserToken()
	}
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
