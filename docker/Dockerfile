# ==============================================================================
# Build Stage (Go API Server Compiler)
# ==============================================================================
FROM golang:1.24-bookworm AS builder

WORKDIR /build

# 依存モジュールのキャッシュ
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピーしてビルド
COPY src/ ./src/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./src

# ==============================================================================
# Runner Stage (Debian Slim with Discord & GUI Runtime)
# ==============================================================================
FROM debian:bookworm-slim

# 非インタラクティブモードの設定
ENV DEBIAN_FRONTEND=noninteractive

# 依存パッケージのインストール
# x11vnc, xvfb: 画面なしGUI実行とVNCアクセス用
# 各種ライブラリ: Discord (Electron) が動作するために必要な最小限のパッケージ
RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb \
    x11vnc \
    novnc \
    websockify \
    wget \
    ca-certificates \
    procps \
    dbus \
    dbus-x11 \
    libasound2 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libc6 \
    libcairo2 \
    libcups2 \
    libdbus-1-3 \
    libexpat1 \
    libfontconfig1 \
    libgcc-s1 \
    libgdk-pixbuf2.0-0 \
    libglib2.0-0 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libpango-1.0-0 \
    libx11-6 \
    libxcomposite1 \
    libxdamage1 \
    libxext6 \
    libxfixes3 \
    libxi6 \
    libxrandr2 \
    libxrender1 \
    libxss1 \
    libxtst6 \
    libsecret-1-0 \
    xdg-utils \
    fonts-liberation \
    libatomic1 \
    libgbm1 \
    libnotify4 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# D-BusのUUID生成 (GUIアプリの起動時に警告・エラーが出るのを防ぐ)
RUN dbus-uuidgen --ensure

# アプリケーションディレクトリの準備
WORKDIR /app

# ビルドしたGoバイナリと起動スクリプトのコピー
COPY --from=builder /build/server /app/server
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# 環境変数のデフォルト値
ENV DISPLAY=:99
ENV PORT=8080
ENV ENABLE_VNC=true

# APIポートとVNCポートを公開
EXPOSE 8080 5900 6080

ENTRYPOINT ["/app/entrypoint.sh"]
