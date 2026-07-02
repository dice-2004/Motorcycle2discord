#!/bin/bash
set -e

echo "=== Motorcycle2Discord Container Startup ==="

# 1. 最新のDiscordのインストール/アップデート
# 起動時に常に最新版を入れることで、Discord公式の「アップデート強制による起動不可」を回避します。
echo "Checking and downloading latest Discord deb package..."
wget -q -O /tmp/discord.deb "https://discord.com/api/download?platform=linux&format=deb"

echo "Installing Discord..."
dpkg -i /tmp/discord.deb || { apt-get update && apt-get install -f -y; }
rm -f /tmp/discord.deb

# 2. D-Bus システムデーモンの起動
# Electron/Discordの動作に必要なメッセージバスサービスを起動します。
service dbus start || true

# 3. 仮想ディスプレイ Xvfb の常時起動 (デフォルト :99)
# Xvfb自体はメモリをほとんど消費しないため、常時稼働させることでVNC接続の安定化とDiscordの起動ラグ短縮を図ります。
echo "Starting Xvfb on display $DISPLAY..."
Xvfb "$DISPLAY" -screen 0 1024x768x16 &
sleep 1 # 起動完了を少し待つ

# 4. VNCサーバーの常時起動 (ENABLE_VNC=true の場合)
# 初回のログイン操作や、ログイン切れの時のトラブルシュート用にVNCサーバーを裏で動かします。
if [ "$ENABLE_VNC" = "true" ]; then
    echo "Starting VNC Server on port 5900..."
    # -forever: 接続が切れてもVNCサーバーを終了しない
    # -shared: 複数接続を許可
    # -nopw: パスワードなし (LAN内での利用を想定、不要なら終了後にENABLE_VNC=falseに設定)
    x11vnc -display "$DISPLAY" -forever -shared -nopw -rfbport 5900 &

    echo "Starting noVNC (Web VNC) on port 6080..."
    # websockifyを使用してVNC(5900)をWebSockets(6080)に中継し、ブラウザで閲覧可能にします
    websockify --web /usr/share/novnc 6080 localhost:5900 &
fi

# 5. Go APIサーバーのフォアグラウンド起動
echo "Starting Go API Server..."
exec /app/server
