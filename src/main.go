package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
)

// ServerState はAPIサーバーの状態をスレッドセーフに管理し、
// Discordの制御を抽象化されたインターフェースを介して行います。
type ServerState struct {
	mu     sync.Mutex
	dc     DiscordController // 抽象化されたコントローラー
	config *Config
}

// StartRequest は /activities/start APIのパラメータを定義します。
type StartRequest struct {
	Type string `json:"type"`
}

// APIResponse は標準的なAPIレスポンスのフォーマットです。
type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// StatusResponse は /status APIのレスポンスです。
type StatusResponse struct {
	DiscordRunning  bool   `json:"discord_running"`
	CurrentActivity string `json:"current_activity"`
	IsStarting      bool   `json:"is_starting"`
}

func main() {
	log.Println("Starting Motorcycle2Discord API Server...")

	configPath := "config/config.json"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}
	
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config.json: %v", err)
	}
	log.Printf("Successfully loaded configurations for %d activities.\n", len(config.Activities))

	// 2. マネージャーの初期化
	display := ":99"
	if envDisplay := os.Getenv("DISPLAY"); envDisplay != "" {
		display = envDisplay
	}

	// 将来的にAIエージェントの代理返答などの機能で、
	// Discordアプリを起動しない「Direct API」接続等を実装した場合は、
	// ここで別の実装クラス（例: NewDirectDiscordController）を差し替えるだけで済みます。
	dc := NewHeadlessDiscordController(display)

	state := &ServerState{
		dc:     dc,
		config: config,
	}

	// コンテナ起動時のクリーンアップ（異常終了したプロセスの大掃除）
	_ = state.dc.StopActivity()

	// 3. ルーティング定義
	http.HandleFunc("/activities/start", state.handleStart)
	http.HandleFunc("/activities/stop", state.handleStop)
	http.HandleFunc("/status", state.handleStatus)

	// 4. HTTPサーバーの起動
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	log.Printf("Server listening on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

// handleStart はアクティビティを開始するハンドラーです。
func (s *ServerState) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 設定の存在チェック
	act, exists := s.config.Activities[req.Type]
	if !exists {
		s.respondWithError(w, http.StatusBadRequest, "Unknown activity type: "+req.Type)
		return
	}

	// 既に起動処理中の場合は重複処理を防ぐ
	if s.dc.IsStarting() {
		s.respondJSON(w, http.StatusAccepted, APIResponse{
			Status:  "starting",
			Message: "Discord is currently starting up, request queued or ignored",
		})
		return
	}

	// 既に同じアクティビティが動作中の場合
	if s.dc.GetCurrentActivity() == req.Type && s.dc.IsRunning() {
		s.respondJSON(w, http.StatusOK, APIResponse{
			Status:  "running",
			Message: "Activity " + req.Type + " is already running",
		})
		return
	}

	// コントローラーを介してアクティビティを開始 (内部で非同期処理される)
	err := s.dc.StartActivity(act, req.Type)
	if err != nil {
		log.Printf("Error starting activity: %v\n", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to initiate activity")
		return
	}

	s.respondJSON(w, http.StatusAccepted, APIResponse{
		Status:  "starting",
		Message: "Initiated Discord startup sequence for: " + req.Type,
	})
}

// handleStop はアクティビティを停止するハンドラーです。
func (s *ServerState) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	log.Println("Received stop request. Tearing down activity...")
	if err := s.dc.StopActivity(); err != nil {
		log.Printf("Error stopping Discord: %v\n", err)
		s.respondWithError(w, http.StatusInternalServerError, "Error occurred during stop sequence")
		return
	}

	s.respondJSON(w, http.StatusAccepted, APIResponse{
		Status:  "stopped",
		Message: "Discord and Xvfb processes terminated",
	})
}

// handleStatus は現在のサーバー状態を返します。
func (s *ServerState) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondWithError(w, http.StatusMethodNotAllowed, "Only GET method is allowed")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.respondJSON(w, http.StatusOK, StatusResponse{
		DiscordRunning:  s.dc.IsRunning(),
		CurrentActivity: s.dc.GetCurrentActivity(),
		IsStarting:      s.dc.IsStarting(),
	})
}

// respondJSON はJSON形式でレスポンスを返します。
func (s *ServerState) respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(response)
}

// respondWithError はエラーレスポンスをJSONで返します。
func (s *ServerState) respondWithError(w http.ResponseWriter, code int, message string) {
	s.respondJSON(w, code, APIResponse{
		Status:  "error",
		Message: message,
	})
}
