package types

import (
	"path/filepath"
)

// ValidateTemplatesFolder checks if the templates folder exists and isn't empty
func ValidateTemplatesFolder() error {
	thePath, err := getTemplatePath()
	if err != nil {
		return err
	}

	// Check if templates folder exists but is empty
	isEmpty := true
	files, err := filepath.Glob(filepath.Join(thePath, "*"))
	if err == nil && len(files) > 0 {
		isEmpty = false
	}

	if isEmpty {
		return ErrEmptyTemplatesFolder
	}

	return nil
}
