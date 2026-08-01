# Walkthrough — Setup, Run, and Test

End-to-end guide for running the Calebsons distributed task queue locally,
including the shared dashboard shell (overview + five real-world demo pages).
Follow the sections in order the first time through.

## 1. Prerequisites

| Tool | Version | Check |
|------|---------|-------|
| Go | `1.22+` | `go version` |
| Git | any recent | `git --version` |
| `curl` | any | `curl --version` |
| Browser | any modern | — |

On macOS: `brew install go`.

## 2. Get the code

```bash
git clone <your-remote> calebsons
cd calebsons/calebsons_inc/go
```

All commands below assume this directory as the working root unless noted
otherwise.

## 3. Run the server

```bash
go run ./cmd/server
```

Expected output:

```
Calebsons queue dashboard → http://localhost:8080
API health                 → http://localhost:8080/health
```

Leave this terminal open. By default the server seeds **one sample task per
demo** and starts **3** workers. The UI is one dashboard: left rail for
**Overview** and each demo, main panel for the active page.

### Useful env overrides

```bash
PORT=9090 WORKERS=5 SEED_DEMO=false go run ./cmd/server
```

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8080` | Listen port |
| `WORKERS` | `3` | Concurrent workers |
| `SEED_DEMO` | `true` | Auto-enqueue one job per demo kind |

## 4. Open the dashboard

### Overview — http://localhost:8080

Same dashboard chrome as every demo page. You should see:

1. Left rail: **Overview** (active) plus links to all five demos.
2. **Calebsons** title, **live** badge, and global meters.
3. Five demo cards in the main panel.
4. An **All recent tasks** stream with kind chips on each row.

### Demo pages

| Demo | URL |
|------|-----|
| Email & notifications | http://localhost:8080/demos/email |
| Image & video processing | http://localhost:8080/demos/media |
| Invoices & reports | http://localhost:8080/demos/reports |
| Webhook & third-party sync | http://localhost:8080/demos/webhooks |
| Data cleanup & maintenance | http://localhost:8080/demos/cleanup |

Each demo page keeps the same rail and swaps the main panel for:

- Meters scoped to that demo only
- A form with fields for that workflow
- A task list filtered to that `kind`

### Manual UI checks

1. From home, open **Email & notifications**, enqueue a welcome email — it
   should complete with a delivery `result` string.
2. On **Image & video processing**, enqueue a resize — result should mention
   an output asset path.
3. On **Webhooks**, check **Simulate flaky remote**, enqueue — watch it fail,
   retry, then complete (or land in **Dead** if max tries are too low).
4. On **Cleanup**, set dry run to `true` — result should say it *would*
   archive rows.
5. Cancel a `queued` or `running` task from any demo page.
6. Confirm the home stream shows tasks from all kinds with kind chips.

## 5. Smoke-test the API with curl

### Health & demo catalog

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/api/demos
```

### Stats (global or per kind)

```bash
curl -s http://localhost:8080/api/stats
curl -s "http://localhost:8080/api/stats?kind=email"
```

### Enqueue a demo job

```bash
curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"kind":"email","name":"send-reset","payload":"{\"to\":\"ada@example.com\",\"template\":\"password-reset\"}","max_attempts":3}'
```

```bash
curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"kind":"webhooks","name":"flaky-sync","payload":"{\"target\":\"shopify\",\"event\":\"order.paid\",\"order_id\":\"ORD-9\",\"flaky\":true}","max_attempts":3}'
```

### List by kind / status

```bash
curl -s "http://localhost:8080/api/tasks?kind=media&limit=5"
curl -s "http://localhost:8080/api/tasks?kind=webhooks&status=completed"
```

### Cancel

```bash
curl -s -X POST http://localhost:8080/api/tasks/<task-id>/cancel
```

## 6. Troubleshooting

| Symptom | Fix |
|---------|-----|
| `go: command not found` | Install Go and put it on `PATH`. |
| `Address already in use` | `PORT=9090 go run ./cmd/server`. |
| Dashboard shows **offline** | Server terminal still running? Hard-refresh after restart. |
| `kind is required` on enqueue | Include `"kind":"email"` (or another valid kind). |
| Stale UI after code changes | Restart `go run` (assets are embedded at build time) and hard-refresh. |
| Want an empty queue | Restart with `SEED_DEMO=false`. |

## 7. Next steps

- Point producers at `POST /api/tasks` with a `kind` from your domain.
- Swap the in-memory broker for Redis/NATS (see `_schedule/go.md`).
- Add delayed jobs, gRPC, and Kubernetes manifests as the queue hardens.
