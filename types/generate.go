package types

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/colors"
	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/file"
	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/logger"
	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/walk"
)

type Generator struct {
	Against   string   `json:"against"`
	Templates []string `json:"templates"`
}

// Generate generates the code for the codebase using the given templates.
func (cb *CodeBase) Generate() {
	VerboseLog("Starting code generation process")

	// Validate that the necessary files and folders exist
	if err := cb.isValidSetup(); err != nil {
		logger.Fatal(err)
	}

	// Before we start, we need to verify that the validators are in place
	cb.verifyValidators()

	generatedPath := GetGeneratedPath()
	if !file.FolderExists(generatedPath) {
		logger.Fatal(fmt.Sprintf("generatedPath %s is empty", generatedPath))
	}
	VerboseLog("Creating generated code directory at", generatedPath)
	_ = file.EstablishFolder(generatedPath)

	generators, err := getGenerators()
	if err != nil {
		logger.Fatal(err)
	}

	VerboseLog("Processing generators")
	for _, generator := range generators {
		VerboseLog("Processing", generator.Against, "templates")
		switch generator.Against {
		case "codebase":
			for _, source := range generator.Templates {
				VerboseLog("Processing codebase template:", source)
				if err := cb.ProcessFile(source, "", ""); err != nil {
					logger.Fatal(err)
				}
			}
		case "groups":
			for _, source := range generator.Templates {
				VerboseLog("Processing group template:", source)
				for _, group := range cb.GroupList("") {
					VerboseLog("  - For group:", group.GroupName())
					if err := cb.ProcessGroupFile(source, group.GroupName(), "readme"); err != nil {
						logger.Fatal(err)
					}
				}
			}
			for _, source := range generator.Templates {
				VerboseLog("Processing group model template:", source)
				for _, group := range cb.GroupList("") {
					if err := cb.ProcessGroupFile(source, group.GroupName(), "model"); err != nil {
						logger.Fatal(err)
					}
				}
			}
		case "routes":
			for _, source := range generator.Templates {
				VerboseLog("Processing route template:", source)
				for _, c := range cb.Commands {
					VerboseLog("  - For command:", c.Route)
					if err := c.ProcessFile(source, "", ""); err != nil {
						logger.Fatal(err)
					}
				}
			}
		case "types":
			for _, source := range generator.Templates {
				VerboseLog("Processing type template:", source)
				for _, s := range cb.Structures {
					sort.Slice(s.Members, func(i, j int) bool {
						return s.Members[i].SortName() < s.Members[j].SortName()
					})
					if !s.DisableGo {
						VerboseLog("  - For type:", s.Name())
						if err := s.ProcessFile(source, "", ""); err != nil {
							logger.Fatal(err)
						}
					}
				}
			}
		default:
			logger.Fatal("unknown against value: ", generator.Against)
		}
	}
	logger.Info(colors.Green + "Done..." + strings.Repeat(" ", 120) + colors.Off + "\033[K")
}

// getGenerators returns the generators we will be using
func getGenerators() ([]Generator, error) {
	generatorsPath := getGeneratorsPath() + "/"
	if !file.FolderExists(generatorsPath) {
		return []Generator{}, fmt.Errorf("generatorsPath (%s) not found", generatorsPath)
	}
	VerboseLog("Looking for templates in:", generatorsPath)

	theMap := make(map[string][]string)
	vFunc := func(file string, vP any) (bool, error) {
		_ = vP
		isPartial := strings.HasSuffix(file, ".partial.tmpl")
		isTemplate := strings.HasSuffix(file, ".tmpl") && !isPartial
		filter := os.Getenv("TB_GENERATOR_FILTER")
		if len(filter) > 0 && !strings.Contains(file, filter) {
			return true, nil
		}
		if isTemplate {
			VerboseLog("  Found template:", file)
			// Strip the generators path prefix to get relative path from generators/
			relPath := file

			// Find the index of "generators/" in the path
			generatorsIndex := strings.Index(file, "generators/")
			if generatorsIndex != -1 {
				// Extract path after "generators/"
				relPath = file[generatorsIndex+len("generators/"):]
			}

			if strings.Contains(relPath, string(os.PathSeparator)) {
				parts := strings.Split(relPath, string(os.PathSeparator))
				category := parts[0] // This should be codebase, routes, types, or groups
				theMap[category] = append(theMap[category], file)
			}
		}
		return true, nil
	}

	_ = walk.ForEveryFileInFolder(generatorsPath, vFunc, nil)

	ret := []Generator{}
	for against, templates := range theMap {
		VerboseLog("  Creating generator for:", against, "with templates:", templates)
		g := Generator{
			Against: against,
		}
		sort.Strings(templates)
		for _, template := range templates {
			// Extract just the filename from the full path
			// Find the index of "generators/" in the path and get path after it
			generatorsIndex := strings.Index(template, "generators/")
			if generatorsIndex != -1 {
				// Get path after "generators/"
				relPath := template[generatorsIndex+len("generators/"):]
				// Remove the category part (against/) to get just the template filename
				if strings.HasPrefix(relPath, against+"/") {
					template = strings.TrimPrefix(relPath, against+"/")
				}
			}
			g.Templates = append(g.Templates, template)
		}
		ret = append(ret, g)
	}
	sort.Slice(ret, func(i, j int) bool {
		sort.Strings(ret[i].Templates)
		return ret[i].Against < ret[j].Against
	})

	return ret, nil
}
