package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"windsurf-gateway/internal/patcher"
)

type restoreRequest struct {
	ConfigDir  string `json:"config_dir,omitempty"`
	InstallDir string `json:"install_dir,omitempty"`
}

func main() {
	var (
		listenAddr  = flag.String("listen", "127.0.0.1:0", "listen address for the local UI server")
		noBrowser   = flag.Bool("no-browser", false, "do not auto-open the local UI in a browser")
		applyOnce   = flag.Bool("apply", false, "apply a patch once in CLI mode")
		detectOnly  = flag.Bool("detect", false, "print detected patch state as JSON")
		restoreOnly = flag.Bool("restore", false, "restore the latest backup and print JSON")
		gatewayURL  = flag.String("gateway", "", "gateway endpoint, required with -apply")
		registerURL = flag.String("register-gateway", "", "optional register gateway endpoint")
		authToken   = flag.String("auth-token", "", "optional gateway user token starting with ws-")
		mode        = flag.String("mode", string(patcher.ModeAll), "patch mode: config, extension, all")
		configDir   = flag.String("config-dir", "", "custom Windsurf config directory")
		installDir  = flag.String("install-dir", "", "custom Windsurf install directory")
	)
	flag.Parse()

	if *detectOnly {
		result, err := patcher.Detect(*configDir, *installDir)
		exitWithJSON(result, err)
		return
	}
	if *restoreOnly {
		result, err := patcher.Restore(*configDir, *installDir)
		exitWithJSON(result, err)
		return
	}
	if *applyOnce {
		result, err := patcher.Apply(patcher.ApplyOptions{
			ConfigDir:          *configDir,
			InstallDir:         *installDir,
			GatewayURL:         *gatewayURL,
			RegisterGatewayURL: *registerURL,
			AuthToken:          *authToken,
			Mode:               patcher.Mode(*mode),
		})
		exitWithJSON(result, err)
		return
	}

	server := newUIServer()
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	defer listener.Close()

	url := "http://" + listener.Addr().String()
	log.Printf("Windsurf patcher UI listening on %s", url)
	if !*noBrowser {
		go func() {
			time.Sleep(200 * time.Millisecond)
			if err := openBrowser(url); err != nil {
				log.Printf("open browser failed: %v", err)
			}
		}()
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func exitWithJSON(value any, err error) {
	payload := map[string]any{"ok": err == nil}
	if err != nil {
		payload["error"] = err.Error()
	} else {
		payload["result"] = value
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
	if err != nil {
		os.Exit(1)
	}
}

func newUIServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = indexTemplate.Execute(w, map[string]any{"DefaultMode": patcher.ModeAll})
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		result, err := patcher.Detect(r.URL.Query().Get("config_dir"), r.URL.Query().Get("install_dir"))
		writeJSON(w, result, err)
	})
	mux.HandleFunc("/api/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request patcher.ApplyOptions
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, nil, fmt.Errorf("invalid request: %w", err))
			return
		}
		result, err := patcher.Apply(request)
		writeJSON(w, result, err)
	})
	mux.HandleFunc("/api/restore", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request restoreRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, nil, fmt.Errorf("invalid request: %w", err))
			return
		}
		result, err := patcher.Restore(request.ConfigDir, request.InstallDir)
		writeJSON(w, result, err)
	})
	return &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}

func writeJSON(w http.ResponseWriter, result any, err error) {
	status := http.StatusOK
	if err != nil {
		status = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	payload := map[string]any{"ok": err == nil}
	if err != nil {
		payload["error"] = err.Error()
	} else {
		payload["result"] = result
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

var indexTemplate = template.Must(template.New("index").Parse(indexHTML))
