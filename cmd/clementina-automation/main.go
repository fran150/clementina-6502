// Command clementina-automation serves one exclusively owned headless machine.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fran150/clementina-6502/pkg/computers/clementina"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	port := flag.Uint("port", 0, "loopback TCP port; zero chooses an available port")
	sdFolder := flag.String("sd", "", "host folder mounted as the emulated MIA SD card")
	flag.Parse()
	if *port > 65535 {
		fmt.Fprintln(os.Stderr, "invalid port")
		os.Exit(1)
	}
	if *sdFolder != "" {
		info, err := os.Stat(*sdFolder)
		if err != nil || !info.IsDir() {
			fmt.Fprintln(os.Stderr, "sd path must be an existing directory")
			os.Exit(1)
		}
	}
	machine, err := clementina.NewAutomationWithSDFolder(*sdFolder)
	if err != nil {
		panic(err)
	}
	defer machine.Close()
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		panic(err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "POST" || r.URL.Path != "/v1" || r.Header.Get("Origin") != "" || r.Host != listener.Addr().String() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var request clementina.AutomationRequest
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
			writeJSON(w, map[string]any{"version": 1, "ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"version": 1, "ok": true, "result": result})
	})
	fmt.Printf("http://%s/v1\n", listener.Addr())
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	if err := server.Serve(listener); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// writeJSON sends one response object. A failed write means the client has
// gone away, so the error is only reported on stderr.
func writeJSON(w io.Writer, value any) {
	if err := json.NewEncoder(w).Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, "write response:", err)
	}
}
