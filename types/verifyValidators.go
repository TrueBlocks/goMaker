package types

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/file"
	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/logger"
)

func (cb *CodeBase) verifyValidators() {
	cwd, _ := os.Getwd()
	for _, cmd := range cb.Commands {
		path := filepath.Join(cwd, "chifra/internal/", cmd.Route, "validate.go")
		if file.FileExists(path) {
			for _, opts := range cmd.Options {
				if ok, wanted := ValidateEnums(path, opts.Enums); !ok {
					logger.Fatal(fmt.Sprintf("Missing enum validator (%s) for %s", wanted, path))
				}
			}
		}
	}
}
