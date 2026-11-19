#!/bin/sh
set -eu
mkdir -p /var/lib/paymentsbot-codex-logs
chmod 755 /var/lib/paymentsbot-codex-logs
logdir=/var/lib/paymentsbot-codex-logs
if docker logs --timestamps --since 24h --tail 10000 bobcat-payments-bot-app-1 > "$logdir/app.log.tmp" 2>&1; then
  tail -c 4000000 "$logdir/app.log.tmp" > "$logdir/app.log.next"
  chmod 644 "$logdir/app.log.next"
  mv "$logdir/app.log.next" "$logdir/app.log"
fi
rm -f "$logdir/app.log.tmp"
