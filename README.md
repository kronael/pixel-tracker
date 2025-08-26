# Go Pixel Tracker

A lightweight 1x1 pixel tracking server written in Go, inspired by node-pixel-tracker.

## Features

- 1x1 transparent GIF pixel serving
- Cookie-based user tracking
- Browser detection and user agent parsing
- Query parameter tracking
- Referrer tracking
- IP address tracking
- Language detection
- In-memory data storage
- RESTful stats API
- Built-in test page

## Installation

```bash
go mod tidy
```

## Usage

### Run the server

```bash
go run main.go
```

Or build and run:

```bash
go build -o pixel-tracker
./pixel-tracker
```

The server will start on port 8080 by default. You can override this with the PORT environment variable:

```bash
PORT=3000 go run main.go
```

## Endpoints

- `GET /` - Test page with example tracking pixels
- `GET /pixel.gif` - The tracking pixel endpoint
- `GET /stats` - JSON API to view collected tracking data

## Embedding the Pixel

### Basic HTML
```html
<img src="http://localhost:8080/pixel.gif" width="1" height="1" alt="">
```

### With tracking parameters
```html
<img src="http://localhost:8080/pixel.gif?campaign=email&user_id=123" width="1" height="1" alt="">
```

### JavaScript dynamic loading
```javascript
var img = new Image();
img.src = 'http://localhost:8080/pixel.gif?event=pageview&page=' + encodeURIComponent(window.location.href);
```

## Tracked Data

Each pixel request captures:

- **Cookies**: All HTTP cookies
- **Host**: Request host
- **Path**: Request path
- **Query Parameters**: All query string parameters
- **Referrer**: HTTP referrer
- **User Agent**: Parsed browser and version
- **IP Address**: Client IP (supports X-Forwarded-For)
- **Language**: Accept-Language header
- **Timestamp**: Time of request

## Example Tracking Data

```json
{
  "cookies": {
    "_tracker": "a3f5b8c912d4e6f8a1b2c3d4e5f6a7b8"
  },
  "host": "localhost:8080",
  "path": "/pixel.gif",
  "referer": "http://localhost:8080/",
  "params": {},
  "query": {
    "campaign": "email",
    "user_id": "12345"
  },
  "ip": "127.0.0.1",
  "decay": 1693424400000,
  "useragent": {
    "browser": "Chrome",
    "version": "116.0.0.0"
  },
  "language": ["en-US", "en"],
  "geo": {
    "ip": "127.0.0.1"
  },
  "domain": "localhost",
  "timestamp": "2023-08-26T10:30:00Z"
}
```

## Customization

### Configure the tracker

```go
tracker := NewPixelTracker()
tracker.Configure(Config{
    DisableCookies: false,
    MaxAge:         2592000,  // 30 days in seconds
    CookieName:     "_tracker",
    TrackIP:        true,
    Port:           "8080",
})
```

### Add custom handlers

```go
tracker.Use(func(data *TrackingData) {
    // Custom processing logic
    fmt.Printf("New tracking event: %+v\n", data)
})
```

## License

MIT