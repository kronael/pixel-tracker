package main

import "net/http"

func serveTestPage(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Pixel Tracker Test Page</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #333; }
        .info { background: #f0f0f0; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .tracking-pixels { margin: 30px 0; }
        .pixel-container { border: 1px solid #ddd; padding: 10px; margin: 10px 0; }
        button { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background: #0056b3; }
        pre { background: #f8f8f8; padding: 10px; border-radius: 4px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Pixel Tracker Test Page</h1>
        
        <div class="info">
            <h2>How it works:</h2>
            <p>This page includes tracking pixels that send data to the server when loaded.</p>
            <p>Each pixel below will trigger a tracking event with different parameters.</p>
        </div>

        <div class="tracking-pixels">
            <h2>Tracking Pixels:</h2>
            
            <div class="pixel-container">
                <h3>Basic Pixel</h3>
                <img src="/pixel.gif" alt="tracking pixel" width="1" height="1">
                <p>Simple tracking pixel with no extra parameters</p>
            </div>

            <div class="pixel-container">
                <h3>Pixel with Campaign Data</h3>
                <img src="/pixel.gif?campaign=email&source=newsletter" alt="tracking pixel" width="1" height="1">
                <p>Tracking pixel with campaign parameters</p>
            </div>

            <div class="pixel-container">
                <h3>Pixel with User ID</h3>
                <img src="/pixel.gif?user_id=12345&action=view" alt="tracking pixel" width="1" height="1">
                <p>Tracking pixel with user identification</p>
            </div>

            <div class="pixel-container">
                <h3>Pixel with Custom Event</h3>
                <img src="/pixel.gif?event=button_click&value=header_cta" alt="tracking pixel" width="1" height="1">
                <p>Tracking pixel for custom events</p>
            </div>
        </div>

        <div style="margin: 30px 0;">
            <h2>View Tracking Data:</h2>
            <button onclick="fetchStats()">Load Tracking Stats</button>
            <div id="stats" style="margin-top: 20px;"></div>
        </div>

        <script>
        function fetchStats() {
            fetch('/stats')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('stats').innerHTML = '<pre>' + JSON.stringify(data, null, 2) + '</pre>';
                })
                .catch(error => {
                    document.getElementById('stats').innerHTML = '<p style="color: red;">Error loading stats: ' + error + '</p>';
                });
        }

        // Load a dynamic pixel after 2 seconds
        setTimeout(() => {
            var img = new Image();
            img.src = '/pixel.gif?event=delayed_load&delay=2000';
            console.log('Delayed tracking pixel loaded');
        }, 2000);
        </script>
    </div>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
