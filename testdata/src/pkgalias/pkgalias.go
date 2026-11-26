package pkgalias

import (
	bytesv1 "bytes"
	_ "embed"
	pkgFmt "fmt"     // want `import alias pkgFmt: check for an import cycle or name clash first; drop the alias, or use \{module\}v1 for a real clash`
	strpkg "strings" // want `import alias strpkg: check for an import cycle or name clash first; drop the alias, or use \{module\}v1 for a real clash`
)

var (
	_ = strpkg.ToLower
	_ = pkgFmt.Sprint
	_ = bytesv1.Equal
)
