//go:build !dev

package bindata

import "embed"

// Fs is the embedded filesystem containing the frontend static assets.
//
//go:embed static/*
var Fs embed.FS
