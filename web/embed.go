package web

import "embed"

// Static holds the dashboard assets served at / and /static/.
//
//go:embed static/*
var Static embed.FS
