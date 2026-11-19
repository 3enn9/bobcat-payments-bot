# Private MAX Codex assistant

Owner: MAX user 242575328. Existing commands are retained. Plain text in the owner's
private dialog goes to Codex; /new resets the session in queue order. No group access.

The app stores jobs, final responses and delivery progress in codex_jobs and the active
session in codex_dialog. Tables are created idempotently when enabled. One worker,
10 pending jobs, 5-minute timeout. Completed jobs expire after 30 days. Delivery is
at-least-once: a crash after MAX accepts a chunk but before its checkpoint can repeat
that chunk. A crash during an agent run may repeat read-only analysis.

Run with BOTH compose files on every deployment:
`docker compose -f docker-compose.yml -f docker-compose.codex.yml up -d --build app codex-agent`

Ignored .env.codex: CODEX_ENABLED=true, CODEX_MAX_OWNER_ID=242575328,
CODEX_AGENT_URL=http://codex-agent:8090, CODEX_SERVICE_TOKEN, MAX_WEBHOOK_SECRET.
Ignored .env.codex-agent: CODEX_SERVICE_TOKEN, AGENT_DB_USER, AGENT_DB_PASS,
AGENT_DB_ADDR (host:3306), AGENT_DB_NAME. Generate independent random secrets.
Do not give the agent production app or MAX credentials.

Codex CLI pinned to 0.169.0. Login:
`docker compose -f docker-compose.yml -f docker-compose.codex.yml exec codex-agent codex login --device-auth`
Complete browser login yourself; auth is stored in codex_state. Do not copy personal
local credentials. The model follows the authenticated account's CLI default; use
model in ops/codex.config.toml to explicitly choose an available model.
Shell disabled; sandbox read-only; only the four read tools are enabled through MCP.
No public agent ports or Docker socket. SQL is SELECT-only and uses a separate MySQL
account with SELECT grants only on business tables, no FILE/EXECUTE/admin privileges.
The schema tool shows tables visible to that account. Broker limits queries to 5 seconds,
500 rows. Long results are truncated. Source reads exclude env, keys, .git and dependencies.
Logs come from a fixed host collector; no agent-selected Docker commands.

Install paymentsbot-codex-logs.service and .timer from ops into /etc/systemd/system;
enable the timer. It atomically snapshots app logs (last day / 10000 lines / 4MB) every
30s, tool returns at most 500 matching lines and redacts obvious secret assignments.
Only app logs; not historical archives or other VPS services. Never log full credentials
or webhook payloads in the application itself.

Enable MAX secret only after updating the existing webhook subscription with the same
secret (X-Max-Bot-Api-Secret). Rollback: CODEX_ENABLED=false, rebuild/recreate app with
the overlay, stop codex-agent. Keep webhook secret consistent with the subscription.
Keep snapshots, auth and SQL credentials out of Git. Disable feature on auth expiration,
subscription limit, timeout or resource pressure; ordinary bot handlers remain available.

Validate: go test ./...; smoke-test MCP schema/query/source/logs with the read-only
account, verify INSERT denied, then test authenticated execution, session continuation,
/new and actual owner's MAX dialog. Do not send fake MAX updates to production.
