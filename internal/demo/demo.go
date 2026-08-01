package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/internal/task"
)

const (
	KindEmail    = "email"
	KindMedia    = "media"
	KindReports  = "reports"
	KindWebhooks = "webhooks"
	KindCleanup  = "cleanup"
)

type Field struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	Required    bool   `json:"required"`
}

type Info struct {
	Kind        string  `json:"kind"`
	Title       string  `json:"title"`
	Blurb       string  `json:"blurb"`
	Path        string  `json:"path"`
	ExampleName string  `json:"example_name"`
	Fields      []Field `json:"fields"`
}

var catalog = []Info{
	{
		Kind:        KindEmail,
		Title:       "Email & notifications",
		Blurb:       "Queue welcome emails, password resets, and push alerts so the API responds instantly while workers deliver in the background.",
		Path:        "/demos/email",
		ExampleName: "send-welcome-email",
		Fields: []Field{
			{Key: "to", Label: "Recipient", Placeholder: "user@example.com", Required: true},
			{Key: "template", Label: "Template", Placeholder: "welcome", Required: true},
			{Key: "subject", Label: "Subject", Placeholder: "Welcome to Calebsons", Required: false},
		},
	},
	{
		Kind:        KindMedia,
		Title:       "Image & video processing",
		Blurb:       "After an upload, workers resize, thumbnail, and transcode media without blocking the upload request.",
		Path:        "/demos/media",
		ExampleName: "resize-product-photo",
		Fields: []Field{
			{Key: "asset_id", Label: "Asset ID", Placeholder: "img_1842", Required: true},
			{Key: "op", Label: "Operation", Placeholder: "thumbnail", Required: true},
			{Key: "width", Label: "Width (px)", Placeholder: "640", Required: false},
		},
	},
	{
		Kind:        KindReports,
		Title:       "Invoices & reports",
		Blurb:       "Build PDFs and CSVs on demand or on a schedule, then email the artifact when the job finishes.",
		Path:        "/demos/reports",
		ExampleName: "generate-invoice-pdf",
		Fields: []Field{
			{Key: "customer", Label: "Customer", Placeholder: "acme-corp", Required: true},
			{Key: "format", Label: "Format", Placeholder: "pdf", Required: true},
			{Key: "period", Label: "Period", Placeholder: "2026-Q2", Required: false},
		},
	},
	{
		Kind:        KindWebhooks,
		Title:       "Webhook & third-party sync",
		Blurb:       "Push order updates to Shopify, Stripe, or Slack. Flaky remotes retry with backoff, then land in dead-letter.",
		Path:        "/demos/webhooks",
		ExampleName: "sync-order-paid",
		Fields: []Field{
			{Key: "target", Label: "Target", Placeholder: "shopify", Required: true},
			{Key: "event", Label: "Event", Placeholder: "order.paid", Required: true},
			{Key: "order_id", Label: "Order ID", Placeholder: "ORD-1001", Required: true},
		},
	},
	{
		Kind:        KindCleanup,
		Title:       "Data cleanup & maintenance",
		Blurb:       "Purge expired sessions, reindex search, or archive old records across workers without loading the app servers.",
		Path:        "/demos/cleanup",
		ExampleName: "purge-expired-sessions",
		Fields: []Field{
			{Key: "job", Label: "Job", Placeholder: "purge-sessions", Required: true},
			{Key: "older_than_days", Label: "Older than (days)", Placeholder: "30", Required: true},
			{Key: "dry_run", Label: "Dry run", Placeholder: "false", Required: false},
		},
	},
}

func Catalog() []Info {
	out := make([]Info, len(catalog))
	copy(out, catalog)
	return out
}

func Get(kind string) (Info, bool) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	for _, d := range catalog {
		if d.Kind == kind {
			return d, true
		}
	}
	return Info{}, false
}

func ValidKind(kind string) bool {
	_, ok := Get(kind)
	return ok
}

func NormalizeKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if ValidKind(kind) {
		return kind
	}
	return ""
}

// Handle simulates realistic work for each demo kind and writes task.Result.
func Handle(ctx context.Context, t *task.Task) error {
	kind := NormalizeKind(t.Kind)
	if kind == "" {
		kind = KindEmail
	}

	delay := workDelay(kind, t)
	if err := sleep(ctx, delay); err != nil {
		return err
	}

	payload := map[string]any{}
	_ = json.Unmarshal([]byte(t.Payload), &payload)

	// Flaky webhook demos fail until the last attempt.
	if kind == KindWebhooks {
		if flaky, _ := payload["flaky"].(bool); flaky && t.Attempts < t.MaxAttempts {
			return fmt.Errorf("remote %v returned 503", payload["target"])
		}
		if strings.Contains(strings.ToLower(t.Name), "flaky") && t.Attempts < t.MaxAttempts {
			return fmt.Errorf("remote endpoint temporarily unavailable")
		}
	}
	if forceFail, _ := payload["fail"].(bool); forceFail {
		return fmt.Errorf("handler rejected payload")
	}

	t.Result = resultFor(kind, t, payload)
	return nil
}

func SeedRequests() []task.CreateRequest {
	return []task.CreateRequest{
		{Kind: KindEmail, Name: "send-welcome-email", Payload: `{"to":"ada@example.com","template":"welcome","subject":"Welcome"}`, MaxAttempts: 3},
		{Kind: KindMedia, Name: "resize-product-photo", Payload: `{"asset_id":"img_1842","op":"thumbnail","width":"640"}`, MaxAttempts: 3},
		{Kind: KindReports, Name: "generate-invoice-pdf", Payload: `{"customer":"acme-corp","format":"pdf","period":"2026-Q2"}`, MaxAttempts: 2},
		{Kind: KindWebhooks, Name: "flaky-shopify-sync", Payload: `{"target":"shopify","event":"order.paid","order_id":"ORD-1001","flaky":true}`, MaxAttempts: 3},
		{Kind: KindCleanup, Name: "purge-expired-sessions", Payload: `{"job":"purge-sessions","older_than_days":"30","dry_run":"false"}`, MaxAttempts: 2},
	}
}

func workDelay(kind string, t *task.Task) time.Duration {
	base := map[string]time.Duration{
		KindEmail:    700 * time.Millisecond,
		KindMedia:    1400 * time.Millisecond,
		KindReports:  1600 * time.Millisecond,
		KindWebhooks: 900 * time.Millisecond,
		KindCleanup:  1100 * time.Millisecond,
	}[kind]
	return base + time.Duration(len(t.Name)%4)*200*time.Millisecond
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func resultFor(kind string, t *task.Task, payload map[string]any) string {
	switch kind {
	case KindEmail:
		return fmt.Sprintf("Delivered %v to %v via provider (message_id=msg_%s)",
			str(payload, "template", "notification"),
			str(payload, "to", "unknown"),
			t.ID[len(t.ID)-6:])
	case KindMedia:
		return fmt.Sprintf("Processed %v with op=%v width=%v → s3://media/%v/out.webp",
			str(payload, "asset_id", "asset"),
			str(payload, "op", "transcode"),
			str(payload, "width", "auto"),
			str(payload, "asset_id", "asset"))
	case KindReports:
		return fmt.Sprintf("Wrote %v report for %v (%v) → reports/%s.%v",
			str(payload, "format", "pdf"),
			str(payload, "customer", "customer"),
			str(payload, "period", "current"),
			t.ID,
			str(payload, "format", "pdf"))
	case KindWebhooks:
		return fmt.Sprintf("Synced %v to %v for %v (http 200)",
			str(payload, "event", "event"),
			str(payload, "target", "target"),
			str(payload, "order_id", "n/a"))
	case KindCleanup:
		dry := str(payload, "dry_run", "false")
		n := 120 + len(t.Name)*3
		if dry == "true" {
			return fmt.Sprintf("Dry run: would %v %d rows older than %v days",
				str(payload, "job", "cleanup"), n, str(payload, "older_than_days", "30"))
		}
		return fmt.Sprintf("Completed %v — archived %d rows older than %v days",
			str(payload, "job", "cleanup"), n, str(payload, "older_than_days", "30"))
	default:
		return "ok"
	}
}

func str(m map[string]any, key, fallback string) string {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			if t != "" {
				return t
			}
		case float64:
			return fmt.Sprintf("%.0f", t)
		case bool:
			return fmt.Sprintf("%t", t)
		}
	}
	return fallback
}
