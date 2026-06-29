#!/bin/sh
set -eu

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"
UMASK="${UMASK:-002}"
CONFIG_DIR="${CONFIG_DIR:-/config}"

umask "$UMASK"

if ! getent group app >/dev/null 2>&1; then
  addgroup -g "$PGID" -S app
fi

if ! id app >/dev/null 2>&1; then
  adduser -u "$PUID" -S app -G app
fi

mkdir -p "$CONFIG_DIR"
chown -R "$PUID:$PGID" "$CONFIG_DIR"

exec su-exec "$PUID:$PGID" /server
