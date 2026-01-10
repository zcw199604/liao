package app

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

var detectAvailablePort = detectAvailablePortDefault

func detectAvailablePortDefault(imgServerHost string) string {
	ports := []string{"9006", "9005", "9003", "9002", "9001", "8006", "8005", "8003", "8002", "8001"}
	client := &http.Client{Timeout: 800 * time.Millisecond}

	for _, port := range ports {
		testURL := fmt.Sprintf("http://%s:%s/useripaddressv23.js", imgServerHost, port)
		resp, err := client.Get(testURL)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return port
		}
	}

	return "9006"
}
