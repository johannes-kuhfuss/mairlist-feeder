package app

import "embed"

//go:embed static/vendor/bootstrap/bootstrap.min.css static/vendor/bootstrap/bootstrap.bundle.min.js
var staticFiles embed.FS
