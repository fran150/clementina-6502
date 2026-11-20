package clementina

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// ListenAutomation opens the loopback TCP port the automation API is served
// on. Port zero chooses an available port.
func ListenAutomation(port uint) (net.Listener, error) {
	if port > 65535 {
		return nil, fmt.Errorf("invalid port")
	}
	return net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
}

// AutomationURL is the endpoint clients post requests to.
func AutomationURL(listener net.Listener) string {
	return fmt.Sprintf("http://%s/v1", listener.Addr())
}

// ServeAutomation serves protocol v1 for machine on listener until the
// listener closes. Only same-host requests to the listener's own address are
// accepted, and browsers (requests with an Origin) are refused.
func ServeAutomation(listener net.Listener, machine *Automation) error {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "POST" || r.URL.Path != "/v1" || r.Header.Get("Origin") != "" || r.Host != listener.Addr().String() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var request AutomationRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&request)
		if err == nil {
			var extra any
			if decoder.Decode(&extra) != io.EOF {
				err = fmt.Errorf("expected one JSON request")
			}
		}
		var result any
		if err == nil {
			result, err = machine.Execute(request)
		}
		if err != nil {
			writeAutomationJSON(w, map[string]any{"version": 1, "ok": false, "error": err.Error()})
			return
		}
		writeAutomationJSON(w, map[string]any{"version": 1, "ok": true, "result": result})
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	return server.Serve(listener)
}

// writeAutomationJSON sends one response object. A failed write means the
// client has gone away, so the error is only reported on stderr.
func writeAutomationJSON(w io.Writer, value any) {
	if err := json.NewEncoder(w).Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, "write response:", err)
	}
}
