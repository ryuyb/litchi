//go:build !embed

package static

import (
	"fmt"
	"net/http"
)

func getFileSystem() (http.FileSystem, error) {
	return nil, fmt.Errorf("embed build tag not set - frontend not embedded")
}