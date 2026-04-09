//go:build embed

package static

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var distFS embed.FS

func getFileSystem() (http.FileSystem, error) {
	fsys, err := fs.Sub(distFS, "dist/client")
	if err != nil {
		return nil, fmt.Errorf("failed to create sub filesystem from embedded dist: %w", err)
	}
	return http.FS(fsys), nil
}