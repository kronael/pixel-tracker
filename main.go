package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var pixel1x1 = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00,
	0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b,
}

type Config struct {
	DisableCookies bool
	MaxAge         int
	CookieName     string
	TrackIP        bool
	Port           string
}

type TrackingData struct {
	Cookies   map[string]string `json:"cookies"`
	Host      string            `json:"host"`
	Path      string            `json:"path"`
	Referer   string            `json:"referer"`
	Params    map[string]string `json:"params"`
	Query     map[string]string `json:"query"`
	IP        string            `json:"ip,omitempty"`
	Decay     int64             `json:"decay"`
	UserAgent BrowserInfo       `json:"useragent"`
	Language  []string          `json:"language"`
	Geo       GeoInfo           `json:"geo"`
	Domain    string            `json:"domain"`
	Timestamp time.Time         `json:"timestamp"`
}

type BrowserInfo struct {
	Browser string `json:"browser"`
	Version string `json:"version"`
}

type GeoInfo struct {
	IP string `json:"ip"`
}

type PixelTracker struct {
	config    Config
	handlers  []func(data *TrackingData)
	dataStore *DataStore
	mu        sync.RWMutex
}

type DataStore struct {
	data []TrackingData
	mu   sync.RWMutex
}

func NewPixelTracker() *PixelTracker {
	return &PixelTracker{
		config: Config{
			DisableCookies: false,
			MaxAge:         2592000,
			CookieName:     "_tracker",
			TrackIP:        true,
			Port:           "8080",
		},
		handlers:  []func(data *TrackingData){},
		dataStore: &DataStore{data: []TrackingData{}},
	}
}

func (pt *PixelTracker) Configure(config Config) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.config = config
}

func (pt *PixelTracker) Use(handler func(data *TrackingData)) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.handlers = append(pt.handlers, handler)
}

func (pt *PixelTracker) PixelHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	cookie, err := r.Cookie(pt.config.CookieName)
	if !pt.config.DisableCookies && (err != nil || cookie == nil) {
		token := generateUserToken()
		http.SetCookie(w, &http.Cookie{
			Name:     pt.config.CookieName,
			Value:    token,
			MaxAge:   pt.config.MaxAge,
			HttpOnly: true,
			Path:     "/",
		})
	}

	w.Write(pixel1x1)

	go pt.processRequest(r, cookie)
}

func (pt *PixelTracker) processRequest(r *http.Request, cookie *http.Cookie) {
	trackingData := &TrackingData{
		Cookies:   extractCookies(r),
		Host:      r.Host,
		Path:      r.URL.Path,
		Referer:   getReferer(r),
		Params:    mux.Vars(r),
		Query:     extractQueryParams(r),
		Timestamp: time.Now(),
	}

	if pt.config.TrackIP {
		trackingData.IP = getClientIP(r)
	}

	trackingData.Decay = getDecay(r.URL.Query().Get("decay"))
	trackingData.UserAgent = parseUserAgent(r.UserAgent())
	trackingData.Language = parseLanguage(r.Header.Get("Accept-Language"))
	trackingData.Geo = GeoInfo{IP: getClientIP(r)}
	trackingData.Domain = extractDomain(r.Host)

	pt.dataStore.mu.Lock()
	pt.dataStore.data = append(pt.dataStore.data, *trackingData)
	pt.dataStore.mu.Unlock()

	pt.mu.RLock()
	handlers := pt.handlers
	pt.mu.RUnlock()

	for _, handler := range handlers {
		handler(trackingData)
	}
}

func (pt *PixelTracker) GetTrackingData() []TrackingData {
	pt.dataStore.mu.RLock()
	defer pt.dataStore.mu.RUnlock()
	dataCopy := make([]TrackingData, len(pt.dataStore.data))
	copy(dataCopy, pt.dataStore.data)
	return dataCopy
}

func (pt *PixelTracker) StatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data := pt.GetTrackingData()
	json.NewEncoder(w).Encode(data)
}

func generateUserToken() string {
	rand.Seed(time.Now().UnixNano())
	val := fmt.Sprintf("%f", rand.Float64())
	hash := md5.Sum([]byte(val))
	return hex.EncodeToString(hash[:])
}

func extractCookies(r *http.Request) map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range r.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	return cookies
}

func extractQueryParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}

func getReferer(r *http.Request) string {
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = r.Header.Get("Referrer")
	}
	if referer == "" {
		referer = "direct"
	}
	return referer
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func getDecay(decay string) int64 {
	if decay == "" {
		return time.Now().Add(5*time.Minute).Unix() * 1000
	}
	return 0
}

func parseUserAgent(userAgent string) BrowserInfo {
	if userAgent == "" {
		return BrowserInfo{Browser: "unknown", Version: ""}
	}

	// Check browsers in specific order (Edge before Chrome since Edge contains Chrome string)
	browserTests := []struct {
		name  string
		regex *regexp.Regexp
	}{
		{"Edge", regexp.MustCompile(`Edg/(\S+)`)},
		{"Firefox", regexp.MustCompile(`Firefox/(\S+)`)},
		{"Safari", regexp.MustCompile(`Version/(\S+).*?Safari/`)},
		{"Chrome", regexp.MustCompile(`Chrome/(\S+)`)},
		{"Opera", regexp.MustCompile(`Opera/(\S+)`)},
		{"MSIE", regexp.MustCompile(`MSIE (\S+);`)},
	}

	for _, test := range browserTests {
		matches := test.regex.FindStringSubmatch(userAgent)
		if len(matches) > 1 {
			return BrowserInfo{
				Browser: test.name,
				Version: matches[1],
			}
		}
	}

	return BrowserInfo{Browser: "other", Version: ""}
}

func parseLanguage(acceptLanguage string) []string {
	if acceptLanguage == "" {
		return []string{}
	}

	parts := strings.Split(acceptLanguage, ",")
	var languages []string
	for _, part := range parts {
		lang := strings.Split(part, ";")[0]
		lang = strings.TrimSpace(lang)
		if lang != "" {
			languages = append(languages, lang)
		}
	}
	return languages
}

func extractDomain(host string) string {
	if host == "" {
		return ""
	}

	u, err := url.Parse("http://" + host)
	if err != nil {
		return host
	}
	return u.Hostname()
}

func main() {
	tracker := NewPixelTracker()

	tracker.Use(func(data *TrackingData) {
		log.Printf("Tracking event: %s from %s", data.Path, data.IP)
	})

	r := mux.NewRouter()
	r.HandleFunc("/pixel.gif", tracker.PixelHandler).Methods("GET", "HEAD")
	r.HandleFunc("/stats", tracker.StatsHandler).Methods("GET")
	r.HandleFunc("/", serveTestPage).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting pixel tracker server on port %s", port)
	log.Printf("Test page: http://localhost:%s/", port)
	log.Printf("Pixel endpoint: http://localhost:%s/pixel.gif", port)
	log.Printf("Stats endpoint: http://localhost:%s/stats", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
