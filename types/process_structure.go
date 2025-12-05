package types

import (
	"os"
	"path/filepath"
	"strings"
)

// ProcessFile processes a single file, applying the template to it and
// writing the result to the destination.
func (item *Structure) ProcessFile(sourceIn, group, reason string) error {
	VerboseLog("Processing structure file:", sourceIn, "for type:", item.Name())

	cwd, _ := os.Getwd()
	subPath := "types"
	fullPath := filepath.Join(cwd, getGeneratorsPath(), subPath, sourceIn)
	if ok, err := shouldProcess(fullPath, subPath, item.Class); err != nil {
		return err
	} else if !ok {
		VerboseLog("  Skipping", sourceIn, "as it should not be processed")
		return nil
	}

	VerboseLog("  Reading template from:", fullPath)
	// For types in pkg/types/<route>/<type>.go format, pass the route
	route := item.Route
	if route == "" {
		route = strings.ToLower(item.Class)
	}
	tmpl, dest := getGeneratorContentsAndDest(fullPath, subPath, group, reason, route, item.Name(), group)
	if strings.Contains(dest, "/-facet-") {
		for _, facet := range item.Facets {
			name := Lower("/" + facet.Name)
			if name == "/index" {
				name = "/indexdata"
			}
			dd := strings.ReplaceAll(dest, "/-facet-", name)
			dd = strings.ReplaceAll(dd, "/-Facet-", "/"+facet.Name)
			VerboseLog("  Generating file:", dd)
			tmplName := fullPath + group + reason + facet.Name
			result := facet.executeTemplate(tmplName, tmpl)
			result = strings.ReplaceAll(result, "--Facet--", facet.Name)
			if strings.Contains(dest, "Panel") && strings.Contains(tmpl, "onFinal") && !strings.Contains(name, "openapprovals") {
				result = strings.Replace(result, "onFinal", "_onFinal", -1)
			}
			_, err := WriteCode(dd, result)
			if err != nil {
				return err
			}
		}
	} else {
		VerboseLog("  Generating file:", dest)
		tmplName := fullPath + group + reason
		result := item.executeTemplate(tmplName, tmpl)
		_, err := WriteCode(dest, result)

		return err
	}

	return nil
}

func (f *Facet) executeTemplate(name, tmplCode string) string {
	return executeTemplate(f, "structure", name, tmplCode)
}
