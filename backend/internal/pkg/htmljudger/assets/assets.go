// Package assets embeds the frozen v6 judge assets so a binary needs no
// runtime files. Importing it adds about 10 MB to the binary; services that
// prefer to mount the files can use judge.LoadDir instead.
package assets

import "embed"

// FS holds model.json, references.json and prompts.json.
//
//go:embed model.json references.json prompts.json
var FS embed.FS
