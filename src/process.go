package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ProcessManager はDiscordアプリのプロセス起動・停止・状態監視を管理します。
// 仮想ディスプレイ(Xvfb)はentrypoint.sh側で常時起動している前提で動作します。
type ProcessManager struct {
	discordCmd *exec.Cmd
	mu         sync.Mutex
	display    string
}

// NewProcessManager は新しいProcessManagerインスタンスを作成します。
func NewProcessManager(display string) *ProcessManager {
	return &ProcessManager{
		display: display,
	}
}

// IsDiscordRunning はDiscordプロセスが現在動作中かどうかを判定します。
func (pm *ProcessManager) IsDiscordRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.discordCmd != nil && pm.discordCmd.Process != nil {
		// プロセスが存在するかシグナル0を送って確認
		err := pm.discordCmd.Process.Signal(syscallSignalZero())
		return err == nil
	}
	return false
}

// StartDiscord はDiscordアプリをバックグラウンドで起動します。
func (pm *ProcessManager) StartDiscord() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 既に起動している場合は何もしない
	if pm.discordCmd != nil && pm.discordCmd.Process != nil {
		err := pm.discordCmd.Process.Signal(syscallSignalZero())
		if err == nil {
			log.Println("Discord is already running.")
			return nil
		}
	}

	log.Println("Cleaning up any existing Discord processes...")
	// 以前のセッションの残骸プロセスがあれば掃除
	_ = exec.Command("pkill", "-9", "-f", "discord").Run()
	_ = os.Remove("/tmp/discord-ipc-0") // 古いソケットを削除

	log.Println("Starting Discord in headless mode on display", pm.display)
	// --multi-instance: 複数インスタンスの起動を許可
	// --no-sandbox: Dockerコンテナ内（特にrootユーザー）でChromiumベースのDiscordを動かすために必要
	// --disable-gpu, --disable-dev-shm-usage: 仮想ディスプレイ上での描画ハングや共有メモリ不足でのクラッシュを防止
	pm.discordCmd = exec.Command("discord", "--multi-instance", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage")
	
	// 環境変数の設定
	// DISPLAY: Xvfbのディスプレイ番号を指定
	// XDG_RUNTIME_DIR: DiscordがIPCソケットを作成する親ディレクトリ。/tmpに指定することで /tmp/discord-ipc-0 にソケットを作らせる
	pm.discordCmd.Env = append(os.Environ(), 
		"DISPLAY="+pm.display, 
		"XDG_RUNTIME_DIR=/tmp",
	)
	pm.discordCmd.Stdout = os.Stdout
	pm.discordCmd.Stderr = os.Stderr
	if err := pm.discordCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Discord: %w", err)
	}

	log.Println("Discord process started successfully.")
	return nil
}

// StopDiscord はDiscordのプロセスを確実にキルします。
func (pm *ProcessManager) StopDiscord() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	log.Println("Stopping Discord process...")
	
	// pkillを使用して関連プロセスを確実に木っ端微塵にする
	_ = exec.Command("pkill", "-9", "-f", "discord").Run()
	_ = os.Remove("/tmp/discord-ipc-0") // ソケットも削除

	pm.discordCmd = nil

	log.Println("Discord process stopped.")
	return nil
}

// WaitForIPCSocket はDiscordが起動してIPCソケットファイルが生成されるまで待機します。
func (pm *ProcessManager) WaitForIPCSocket(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	socketPath := "/tmp/discord-ipc-0"

	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			log.Println("Discord IPC socket found.")
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for Discord IPC socket at %s", socketPath)
}

// syscallSignalZero はシグナル0を表すヘルパー関数です。
func syscallSignalZero() syscall.Signal {
	return syscall.Signal(0) // シグナル0はプロセスの存在確認用
}
