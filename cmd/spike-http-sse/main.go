// spike-http-sse is a standalone spike that:
// 1. Starts OpenCode via `opencode serve` in the background
// 2. Creates a new session via the Go SDK
// 3. Sends a prompt to the session
// 4. Listens to the /event SSE stream and records event shapes
// 5. Waits for session idle
// 6. Tests abort
// 7. Records everything to a local run record
//
// This is a spike only — no production code is modified.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	opencode "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

const (
	spikeRunID   = "R-20260524-018"
	spikeIssueID = "I-0015"
)

var (
	baseURL string
	logDir  string
)

func main() {
	logDir = filepath.Join(os.Getenv("HOME"), "coding", "agentwrap-smoke", "runs", spikeRunID+"-i0015-acp-transport-http-sse-spike")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("create log dir: %v", err)
	}
	log.SetOutput(os.Stderr)
	log.Printf("[spike] log dir: %s", logDir)

	eventsLog, err := os.Create(filepath.Join(logDir, "events.jsonl"))
	if err != nil {
		log.Fatalf("create events log: %v", err)
	}
	defer eventsLog.Close()

	sessionLog, err := os.Create(filepath.Join(logDir, "session.json"))
	if err != nil {
		log.Fatalf("create session log: %v", err)
	}
	defer sessionLog.Close()

	// Step 1: Start opencode serve
	var serverCmd *exec.Cmd
	baseURL, serverCmd, err = startServer(eventsLog)
	if err != nil {
		log.Printf("[spike] FAILED to start server: %v", err)
		recordFailure(err, eventsLog)
		return
	}
	log.Printf("[spike] server started at %s (PID %d)", baseURL, serverCmd.Process.Pid)
	saveLog(eventsLog, "server_started", map[string]string{"base_url": baseURL, "pid": fmt.Sprintf("%d", serverCmd.Process.Pid)})

	// Cleanup server on exit
	defer func() {
		serverCmd.Process.Kill()
		serverCmd.Wait()
	}()

	// Give server a moment to be ready
	time.Sleep(2 * time.Second)

	// Step 2: Create client
	client := opencode.NewClient(
		option.WithBaseURL(baseURL),
	)

	// Step 3: Create a session
	ctx := context.Background()
	session, err := client.Session.New(ctx, opencode.SessionNewParams{})
	if err != nil {
		log.Printf("[spike] FAILED to create session: %v", err)
		recordFailure(err, eventsLog)
		return
	}
	log.Printf("[spike] session created: %s", session.ID)
	sessionJSON, _ := json.MarshalIndent(session, "", "  ")
	io.WriteString(sessionLog, string(sessionJSON)+"\n")

	// Step 4: Start SSE event stream listener
	var wg sync.WaitGroup
	sseEvents := make(chan string, 200)
	wg.Add(1)
	go streamEvents(ctx, client, session.ID, eventsLog, sseEvents, &wg)

	// Step 5: Send a simple prompt using Parts
	log.Printf("[spike] sending prompt...")
	saveLog(eventsLog, "sending_prompt", map[string]string{"session_id": session.ID})

	// Build prompt parts using SDK types
	promptParts := []opencode.SessionPromptParamsPartUnion{
		opencode.SessionPromptParamsPart{
			Type: opencode.F(opencode.SessionPromptParamsPartsTypeText),
			Text: opencode.F("Say hello in exactly 5 words."),
		},
	}

	resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
		Parts: opencode.F(promptParts),
	})
	if err != nil {
		log.Printf("[spike] prompt error: %v", err)
		saveLog(eventsLog, "prompt_error", map[string]string{"error": err.Error()})
	} else {
		log.Printf("[spike] prompt response: %+v", resp)
		respJSON, _ := json.MarshalIndent(resp, "", "  ")
		saveLog(eventsLog, "prompt_response", map[string]any{"response": string(respJSON)})
	}

	// Step 6: Wait for session idle or timeout
	idleSeen := false
	idleTimeout := time.After(60 * time.Second)
	for !idleSeen {
		select {
		case evt, ok := <-sseEvents:
			if !ok {
				log.Printf("[spike] SSE channel closed")
				break
			}
			if strings.Contains(evt, `"type":"session.idle"`) || strings.Contains(evt, `"session.idle"`) {
				idleSeen = true
				log.Printf("[spike] *** SESSION IDLE ***")
			}
		case <-idleTimeout:
			log.Printf("[spike] timeout waiting for idle")
			break
		}
	}

	// Step 7: Try abort on a new long-running session
	log.Printf("[spike] testing abort...")
	abortSession, err := client.Session.New(ctx, opencode.SessionNewParams{})
	if err != nil {
		log.Printf("[spike] could not create abort test session: %v", err)
	} else {
		log.Printf("[spike] abort test session: %s", abortSession.ID)
		saveLog(eventsLog, "abort_test_session", map[string]string{"session_id": abortSession.ID})

		// Start SSE for abort session
		abortSSEEvents := make(chan string, 200)
		var abortWG sync.WaitGroup
		abortWG.Add(1)
		go streamEvents(ctx, client, abortSession.ID, eventsLog, abortSSEEvents, &abortWG)

		// Send a long prompt
		longPromptParts := []opencode.SessionPromptParamsPartUnion{
			opencode.SessionPromptParamsPart{
				Type: opencode.F(opencode.SessionPromptParamsPartsTypeText),
				Text: opencode.F("Write a comprehensive essay about the history of computing from 1940 to 2025, covering all major milestones, architectures, and influential figures."),
			},
		}

		_, err = client.Session.Prompt(ctx, abortSession.ID, opencode.SessionPromptParams{
			Parts: opencode.F(longPromptParts),
		})
		if err != nil {
			log.Printf("[spike] abort test prompt error: %v", err)
		}

		// Brief pause then abort
		time.Sleep(3 * time.Second)
		_, err = client.Session.Abort(ctx, abortSession.ID, opencode.SessionAbortParams{})
		if err != nil {
			log.Printf("[spike] abort error: %v", err)
			saveLog(eventsLog, "abort_error", map[string]string{"error": err.Error()})
		} else {
			log.Printf("[spike] abort sent successfully")
			saveLog(eventsLog, "abort_sent", map[string]string{"session_id": abortSession.ID})
		}

		// Wait for abort SSE events
		time.Sleep(5 * time.Second)
		close(abortSSEEvents)
		abortWG.Wait()
	}

	// Close SSE stream
	close(sseEvents)
	wg.Wait()

	log.Printf("[spike] spike complete")
	saveLog(eventsLog, "spike_complete", nil)
}

func startServer(eventsLog *os.File) (string, *exec.Cmd, error) {
	cmd := exec.Command("opencode", "serve", "--port", "0", "--print-logs")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil, fmt.Errorf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("start opencode serve: %v", err)
	}

	// Read stdout to get the actual URL
	buf := make([]byte, 4096)
	var addr string
	for i := 0; i < 20; i++ {
		n, readErr := stdout.Read(buf)
		if readErr != nil && readErr != io.EOF {
			break
		}
		if n > 0 {
			line := string(buf[:n])
			os.Stderr.WriteString(line)
			// Look for "opencode server listening on http://"
			if idx := strings.Index(line, "http://"); idx >= 0 {
				endIdx := strings.Index(line[idx:], "\n")
				if endIdx < 0 {
					endIdx = len(line)
				} else {
					endIdx += idx
				}
				addr = strings.TrimSpace(line[idx:endIdx])
				break
			}
		}
		if readErr == io.EOF {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Fallback: try scanning ports
	if addr == "" {
		for i := 0; i < 30; i++ {
			time.Sleep(500 * time.Millisecond)
			// Try wide range of ports
			ports := []string{
				"8080", "8081", "8082", "8083", "8084", "8085", "8086", "8087", "8088", "8089",
				"39721", "39722", "39723", "39724", "39725", "39726", "39727", "39728", "39729",
				"4096", "4097", "4098", "4099", "4100",
				"41037", "41038", "41039", "41040", "41041",
				"43171", "43172", "43173", "43174", "43175",
			}
			for _, port := range ports {
				testURL := fmt.Sprintf("http://127.0.0.1:%s", port)
				resp, err := http.Get(testURL)
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode < 500 {
						addr = testURL
						goto found
					}
				}
			}
		}
	}
found:
	if addr == "" {
		cmd.Process.Kill()
		return "", nil, fmt.Errorf("could not find server port after 15s")
	}

	return addr, cmd, nil
}

func streamEvents(ctx context.Context, client *opencode.Client, sessionID string, eventsLog *os.File, sseEvents chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	params := opencode.EventListParams{
		Directory: opencode.F(string(os.Getenv("PWD"))),
	}

	stream := client.Event.ListStreaming(ctx, params)
	defer stream.Close()

	lineNum := 0
	for stream.Next() {
		evt := stream.Current()
		lineNum++

		// Record the full event for inspection
		evtBytes, _ := json.Marshal(evt)
		eventsLog.Write(evtBytes)
		eventsLog.WriteString("\n")

		// Check union type and log details
		switch u := evt.AsUnion().(type) {
		case opencode.EventListResponseEventMessageUpdated:
			log.Printf("[spike] SSE #%d: message.updated — role=%s, msgID=%s", lineNum, u.Properties.Info.Role, u.Properties.Info.ID)
			// Log message parts
			if parts := extractParts(u.Properties.Info); len(parts) > 0 {
				for i, p := range parts {
					log.Printf("[spike]   part[%d]: type=%s, text=%q, tool=%s, reason=%s",
						i, p.Type, truncate(p.Text, 50), p.Tool, p.Reason)
				}
			}
		case opencode.EventListResponseEventMessagePartUpdated:
			delta := u.Properties.Delta
			log.Printf("[spike] SSE #%d: message.part.updated (delta len=%d, text=%q)",
				lineNum, len(delta), truncate(delta, 40))
		case opencode.EventListResponseEventSessionIdle:
			log.Printf("[spike] SSE #%d: *** session.idle (session: %s) ***", lineNum, u.Properties.SessionID)
		case opencode.EventListResponseEventSessionCreated:
			log.Printf("[spike] SSE #%d: session.created — sessionID=%s", lineNum, u.Properties.Info.ID)
		case opencode.EventListResponseEventSessionUpdated:
			log.Printf("[spike] SSE #%d: session.updated — sessionID=%s", lineNum, u.Properties.Info.ID)
		case opencode.EventListResponseEventSessionError:
			errName := ""
			if u.Properties.Error.Name != "" {
				errName = string(u.Properties.Error.Name)
			}
			log.Printf("[spike] SSE #%d: *** session.error: name=%s ***", lineNum, errName)
		case opencode.EventListResponseEventPermissionUpdated:
			log.Printf("[spike] SSE #%d: permission.updated", lineNum)
		case opencode.EventListResponseEventFileEdited:
			log.Printf("[spike] SSE #%d: file.edited: %s", lineNum, u.Properties.File)
		case opencode.EventListResponseEventMessageRemoved:
			log.Printf("[spike] SSE #%d: message.removed: msgID=%s", lineNum, u.Properties.MessageID)
		default:
			log.Printf("[spike] SSE #%d: %s", lineNum, string(evt.Type))
		}

		select {
		case sseEvents <- string(evtBytes):
		default:
		}
	}

	if err := stream.Err(); err != nil {
		log.Printf("[spike] SSE stream error: %v", err)
		saveLog(eventsLog, "sse_error", map[string]string{"error": err.Error()})
	}

	log.Printf("[spike] SSE stream ended (total events: %d)", lineNum)
}

// partInfo holds extracted part data
type partInfo struct {
	Type   string
	Text   string
	Tool   string
	Reason string
}

// extractParts extracts parts from a Message by marshaling and looking for "parts" field
func extractParts(msg opencode.Message) []partInfo {
	// Use JSON to extract parts from the union
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil
	}

	var raw struct {
		Parts []struct {
			Type   string `json:"type"`
			Text   string `json:"text"`
			Tool   string `json:"tool"`
			Reason string `json:"reason"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(msgBytes, &raw); err != nil {
		return nil
	}

	result := make([]partInfo, len(raw.Parts))
	for i, p := range raw.Parts {
		result[i] = partInfo{Type: p.Type, Text: p.Text, Tool: p.Tool, Reason: p.Reason}
	}
	return result
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func saveLog(eventsLog *os.File, event string, data any) {
	m := map[string]any{"spike_event": event, "time": time.Now().Format(time.RFC3339Nano)}
	if data != nil {
		m["data"] = data
	}
	b, _ := json.Marshal(m)
	eventsLog.Write(b)
	eventsLog.WriteString("\n")
}

func recordFailure(err error, eventsLog *os.File) {
	m := map[string]any{
		"spike_event": "failure",
		"time":        time.Now().Format(time.RFC3339Nano),
		"error":       err.Error(),
	}
	b, _ := json.Marshal(m)
	eventsLog.Write(b)
	eventsLog.WriteString("\n")
}
