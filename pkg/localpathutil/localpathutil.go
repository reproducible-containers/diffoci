// From https://github.com/lima-vm/lima/blob/v0.17.2/pkg/localpathutil/localpathutil.go .
/*
   Copyright The Lima Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package localpathutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Expand is from https://github.com/lima-vm/lima/blob/v0.17.2/pkg/localpathutil/localpathutil.go#L11-L34 .
//
// Expand expands a path like "~", "~/", "~/foo".
// Paths like "~foo/bar" are unsupported.
//
// FIXME: is there an existing library for this?
func Expand(orig string) (string, error) {
	s := orig
	if s == "" {
		return "", errors.New("empty path")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(s, "~") {
		if s == "~" || strings.HasPrefix(s, "~/") {
			s = strings.Replace(s, "~", homeDir, 1)
		} else {
			// Paths like "~foo/bar" are unsupported.
			return "", fmt.Errorf("unexpandable path %q", orig)
		}
	}
	return filepath.Abs(s)
}
