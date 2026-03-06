package webassets

import "embed"

// FS stores the bundled frontend templates and static assets.
//
//go:embed templates static
var FS embed.FS
