package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const defaultRandServerBase = "http://v1.chat2019.cn/Act/WebService.asmx/getRandServer?ServerInfo=serversdeskry&_="

func main() {
	timeout := flag.Duration("timeout", 5*time.Minute, "maximum probe duration")
	connectTimeout := flag.Duration("connect-timeout", 15*time.Second, "websocket dial timeout")
	readWindow := flag.Duration("read-window", 0, "read deadline extension after each message or pong; set <=0 to disable")
	pingInterval := flag.Duration("ping-interval", 0, "client ping interval; set <=0 to disable")
	upstreamURL := flag.String("url", strings.TrimSpace(os.Getenv("WEBSOCKET_UPSTREAM_URL")), "upstream websocket URL; defaults to WEBSOCKET_UPSTREAM_URL, then rand-server discovery")
	randServerBase := flag.String("rand-server-base", defaultRandServerBase, "rand-server HTTP endpoint base")
	signID := flag.String("id", strings.TrimSpace(os.Getenv("WS_PROBE_USER_ID")), "optional user id for sign message")
	signName := flag.String("name", envDefault("WS_PROBE_USER_NAME", "ws-probe"), "optional user nickname for sign message")
	sendSign := flag.Bool("send-sign", false, "send a sign message after websocket opens")
	flag.Parse()

	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "timeout must be > 0")
		os.Exit(2)
	}

	httpClient := &http.Client{Timeout: *connectTimeout}
	if strings.TrimSpace(*upstreamURL) == "" {
		discovered, err := discoverUpstream(httpClient, *randServerBase)
		if err != nil {
			fmt.Fprintf(os.Stderr, "discover upstream failed: %v\n", err)
			os.Exit(1)
		}
		*upstreamURL = discovered
	}

	started := time.Now()
	fmt.Printf("probe_start=%s timeout=%s url=%s\n", started.Format(time.RFC3339), timeout.String(), *upstreamURL)

	ctx, cancel := context.WithTimeout(context.Background(), *connectTimeout)
	defer cancel()

	dialer := *websocket.DefaultDialer
	conn, resp, err := dialer.DialContext(ctx, *upstreamURL, nil)
	if err != nil {
		status := ""
		if resp != nil {
			status = resp.Status
		}
		fmt.Fprintf(os.Stderr, "dial_failed elapsed=%s status=%s error=%v\n", time.Since(started).Round(time.Millisecond), status, err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("connected elapsed=%s\n", time.Since(started).Round(time.Millisecond))

	if *sendSign {
		if strings.TrimSpace(*signID) == "" {
			fmt.Fprintln(os.Stderr, "send-sign requires -id or WS_PROBE_USER_ID")
			os.Exit(2)
		}
		payload := map[string]any{
			"act":              "sign",
			"id":               *signID,
			"name":             *signName,
			"userSex":          "unknown",
			"address_show":     "false",
			"randomhealthmode": "0",
			"randomvipsex":     "0",
			"randomvipaddress": "0",
			"userip":           "",
			"useraddree":       "",
			"randomvipcode":    "",
		}
		body, _ := json.Marshal(payload)
		if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
			fmt.Fprintf(os.Stderr, "sign_failed elapsed=%s error=%v\n", time.Since(started).Round(time.Millisecond), err)
			os.Exit(1)
		}
		fmt.Printf("sign_sent elapsed=%s id=%s\n", time.Since(started).Round(time.Millisecond), *signID)
	}

	done := make(chan readResult, 1)
	go readLoop(conn, started, *readWindow, done)

	var pingTicker *time.Ticker
	var pingC <-chan time.Time
	if *pingInterval > 0 {
		pingTicker = time.NewTicker(*pingInterval)
		defer pingTicker.Stop()
		pingC = pingTicker.C
	}

	timeoutTimer := time.NewTimer(*timeout)
	defer timeoutTimer.Stop()

	pings := 0
	for {
		select {
		case <-timeoutTimer.C:
			fmt.Printf("probe_timeout elapsed=%s pings=%d result=still_connected\n", time.Since(started).Round(time.Millisecond), pings)
			return
		case <-pingC:
			pings++
			deadline := time.Now().Add(5 * time.Second)
			if err := conn.WriteControl(websocket.PingMessage, []byte("probe"), deadline); err != nil {
				fmt.Printf("ping_failed elapsed=%s pings=%d error=%v\n", time.Since(started).Round(time.Millisecond), pings, err)
				return
			}
			fmt.Printf("ping_sent elapsed=%s pings=%d\n", time.Since(started).Round(time.Millisecond), pings)
		case result := <-done:
			fmt.Printf("disconnected elapsed=%s messages=%d close_code=%d error=%v\n", time.Since(started).Round(time.Millisecond), result.messages, result.closeCode, result.err)
			return
		}
	}
}

type readResult struct {
	messages  int
	closeCode int
	err       error
}

func readLoop(conn *websocket.Conn, started time.Time, readWindow time.Duration, done chan<- readResult) {
	messages := 0
	if readWindow > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(readWindow))
		conn.SetPongHandler(func(appData string) error {
			fmt.Printf("pong_received elapsed=%s data=%q\n", time.Since(started).Round(time.Millisecond), appData)
			return conn.SetReadDeadline(time.Now().Add(readWindow))
		})
	}

	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			closeCode := 0
			if closeErr, ok := err.(*websocket.CloseError); ok {
				closeCode = closeErr.Code
			}
			done <- readResult{messages: messages, closeCode: closeCode, err: err}
			return
		}
		messages++
		if readWindow > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(readWindow))
		}
		preview := string(payload)
		if len(preview) > 240 {
			preview = preview[:240] + "...(truncated)"
		}
		fmt.Printf("message elapsed=%s count=%d type=%d bytes=%d preview=%q\n", time.Since(started).Round(time.Millisecond), messages, msgType, len(payload), preview)
	}
}

func discoverUpstream(client *http.Client, base string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return "", fmt.Errorf("empty rand-server base")
	}
	reqURL := base + strconv.FormatInt(time.Now().UnixMilli(), 10)
	resp, err := client.Get(reqURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("rand-server status %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}

	var node struct {
		State string `json:"state"`
		Msg   struct {
			Server string `json:"server"`
		} `json:"msg"`
	}
	if err := json.Unmarshal(body, &node); err != nil {
		return "", err
	}
	if strings.TrimSpace(node.State) != "OK" {
		return "", fmt.Errorf("rand-server state %q", node.State)
	}
	server := strings.TrimSpace(node.Msg.Server)
	if server == "" {
		return "", fmt.Errorf("rand-server returned empty server")
	}
	return server, nil
}

func envDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
