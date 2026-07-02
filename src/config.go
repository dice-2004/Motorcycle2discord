package main

import (
	"encoding/json"
	"os"
)

// Activity は、Discord RPCで設定するステータスの詳細情報と、対応するApplication Client IDを保持します。
type Activity struct {
	Name     string `json:"name"`      // アクティビティの表示名（例: "運転中"）
	Details  string `json:"details"`   // 詳細ステータス（例: "バイクを運転しています"）
	State    string `json:"state"`     // 状態ステータス（例: "インカム接続中 🏍️"）
	ClientID string `json:"client_id"` // Discord Developer PortalのApplication ID
}

// Config は、設定ファイル全体を表す構造体です。
type Config struct {
	Activities map[string]Activity `json:"activities"`
}

// LoadConfig は、指定されたパスからJSON設定ファイルを読み込みます。
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
