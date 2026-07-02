package main

import (
	"log"
	"sync"
	"time"
)

// DiscordController はDiscordアプリの制御や将来的なメッセージ送受信を抽象化するインターフェースです。
// これにより、ヘッドレスDiscord以外の接続方式（Direct API等）への差し替えや、
// メッセージ送受信機能（AI代理返答など）の追加を、既存のHTTPルーティングに影響を与えずに行うことができます。
type DiscordController interface {
	StartActivity(act Activity, typeName string) error
	StopActivity() error
	IsRunning() bool
	IsStarting() bool
	GetCurrentActivity() string
	
	// --- 将来的な拡張（AI代理返答など）のための設計プレースホルダー ---
	// SendMessage(channelID string, content string) error
	// OnMessageReceived(handler func(msg DiscordMessage))
}

// DiscordMessage は将来的なチャット送受信機能のためのメッセージ構造体です
type DiscordMessage struct {
	ChannelID string
	AuthorID  string
	Content   string
	Timestamp time.Time
}

// HeadlessDiscordController は、仮想ディスプレイ(Xvfb)と公式RPCを使用した
// インターフェースの実装です。
type HeadlessDiscordController struct {
	mu              sync.Mutex
	pm              *ProcessManager
	rm              *RPCManager
	currentActivity string
	isStarting      bool
}

// NewHeadlessDiscordController はHeadlessDiscordControllerのインスタンスを作成します。
func NewHeadlessDiscordController(display string) *HeadlessDiscordController {
	return &HeadlessDiscordController{
		pm: NewProcessManager(display),
		rm: NewRPCManager(),
	}
}

// IsRunning はDiscordプロセスが稼働しているかを判定します。
func (c *HeadlessDiscordController) IsRunning() bool {
	return c.pm.IsDiscordRunning()
}

// IsStarting は起動処理のシーケンス中であるかを判定します。
func (c *HeadlessDiscordController) IsStarting() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isStarting
}

// GetCurrentActivity は現在設定されているアクティビティ名を取得します。
func (c *HeadlessDiscordController) GetCurrentActivity() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentActivity
}

// StartActivity は非同期でDiscordを起動し、RPCアクティビティを設定します。
func (c *HeadlessDiscordController) StartActivity(act Activity, typeName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 既に起動処理中の場合は重複処理を防止
	if c.isStarting {
		log.Println("Discord is already in startup sequence. Request skipped.")
		return nil
	}

	// すでに同一アクティビティが稼働中の場合
	if c.currentActivity == typeName && c.pm.IsDiscordRunning() {
		log.Printf("Activity '%s' is already active.\n", typeName)
		return nil
	}

	// すでにDiscordは起動しているが、別のアクティビティに変更する場合（即座に変更可能）
	if c.pm.IsDiscordRunning() {
		log.Printf("Discord is active. Switching activity to: %s\n", typeName)
		if err := c.rm.SetActivity(act); err != nil {
			return err
		}
		c.currentActivity = typeName
		return nil
	}

	// 新規でDiscordを起動してステータスを設定する場合（非同期で実行）
	c.isStarting = true
	go c.startupSequence(act, typeName)

	return nil
}

// startupSequence は非同期でプロセス起動・ソケット待機・RPC送信を行うシーケンスです。
func (c *HeadlessDiscordController) startupSequence(act Activity, typeName string) {
	log.Printf("Initiating headless Discord startup sequence for: %s\n", typeName)

	// 1. プロセス起動
	if err := c.pm.StartDiscord(); err != nil {
		log.Printf("Failed to start Discord process: %v\n", err)
		c.resetState()
		return
	}

	// 2. ソケットの出現待ち（最大30秒）
	if err := c.pm.WaitForIPCSocket(30 * time.Second); err != nil {
		log.Printf("Discord IPC socket not found: %v\n", err)
		_ = c.pm.StopDiscord()
		c.resetState()
		return
	}

	// DiscordのUI立ち上がりと自動ログインのためのバッファ時間（5秒）
	time.Sleep(5 * time.Second)

	// 3. RPCステータスの設定
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.rm.SetActivity(act); err != nil {
		log.Printf("Failed to set RPC activity: %v\n", err)
		// 失敗した場合はクリーンアップ
		c.rm.Disconnect()
		_ = c.pm.StopDiscord()
		c.currentActivity = ""
	} else {
		c.currentActivity = typeName
	}
	c.isStarting = false
}

// StopActivity はDiscordを終了し、ステータスをクリアします。
func (c *HeadlessDiscordController) StopActivity() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	log.Println("Stopping current activity and terminating Discord...")
	c.rm.Disconnect()
	
	if err := c.pm.StopDiscord(); err != nil {
		return err
	}

	c.currentActivity = ""
	c.isStarting = false
	return nil
}

func (c *HeadlessDiscordController) resetState() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isStarting = false
	c.currentActivity = ""
}

// --- 将来的な拡張機能のモック実装 ---

// SendMessage は指定したチャネルにメッセージを送信するメソッドのプレースホルダーです。
// 将来、AIエージェントがDiscordを通じて自動返信（代理返答）をする際に、
// DiscordのHTTP API（またはUser Client）を利用して実装を追加します。
func (c *HeadlessDiscordController) SendMessage(channelID string, content string) error {
	log.Printf("[Placeholder] Sending message to channel %s: %s\n", channelID, content)
	// TODO: Discord User API (Post Message) の実装を追加する
	return nil
}
