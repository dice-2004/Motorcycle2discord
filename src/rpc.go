package main

import (
	"log"
	"time"

	"github.com/hugolgst/rich-go/client"
)

// RPCManager は、Discord RPC (Rich Presence) との接続とステータス送信を管理します。
type RPCManager struct {
	currentClientID string
	isConnected     bool
}

// NewRPCManager は新しいRPCManagerインスタンスを作成します。
func NewRPCManager() *RPCManager {
	return &RPCManager{}
}

// SetActivity は、指定されたアクティビティ設定に基づいてDiscord RPCに接続し、ステータスを更新します。
func (rm *RPCManager) SetActivity(act Activity) error {
	// クライアントIDが異なる場合（別のアクティビティが開始された場合）は一度ログアウトする
	if rm.isConnected && rm.currentClientID != act.ClientID {
		log.Printf("Client ID changed from %s to %s. Logging out of previous RPC session...\n", rm.currentClientID, act.ClientID)
		client.Logout()
		rm.isConnected = false
	}

	// RPCサーバー（Discordアプリ）にログイン
	if !rm.isConnected {
		log.Printf("Connecting to Discord RPC using Client ID: %s...\n", act.ClientID)
		
		var loginErr error
		for i := 1; i <= 5; i++ {
			loginErr = client.Login(act.ClientID)
			if loginErr == nil {
				break
			}
			log.Printf("Failed to connect to Discord RPC (attempt %d/5): %v. Retrying in 1s...\n", i, loginErr)
			time.Sleep(1 * time.Second)
		}
		
		if loginErr != nil {
			return loginErr
		}
		rm.isConnected = true
		rm.currentClientID = act.ClientID
		log.Println("Successfully connected to Discord RPC.")
	}

	// アクティビティの開始時刻をセット（「経過時間」としてDiscord上でカウントアップ表示されます）
	startTime := time.Now()

	log.Printf("Setting RPC Activity - Name: %s, Details: %s, State: %s\n", act.Name, act.Details, act.State)
	err := client.SetActivity(client.Activity{
		Details: act.Details,
		State:   act.State,
		Timestamps: &client.Timestamps{
			Start: &startTime,
		},
	})
	if err != nil {
		return err
	}

	log.Println("Discord status updated successfully.")
	return nil
}

// Disconnect はDiscord RPC接続をクローズします。
func (rm *RPCManager) Disconnect() {
	if rm.isConnected {
		log.Println("Disconnecting from Discord RPC...")
		client.Logout()
		rm.isConnected = false
		rm.currentClientID = ""
		log.Println("Disconnected from Discord RPC.")
	}
}
