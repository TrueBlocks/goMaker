package types

import (
	"os"

	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/logger"
	"github.com/joho/godotenv"
)

var verbose bool = false

func init() {
	_ = godotenv.Load()
	verbose = os.Getenv("TB_VERBOSE") == "true"
}

func SetVerbose(v bool) {
	verbose = v
}

func IsVerbose() bool {
	return verbose
}

func VerboseLog(args ...interface{}) {
	if verbose {
		logger.Info(args...)
	}
}
