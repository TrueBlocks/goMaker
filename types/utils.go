package types

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/file"
	"github.com/TrueBlocks/trueblocks-chifra/v6/pkg/logger"
)

var ErrNoTemplateFolder = errors.New("could not find the templates directory")

// TemplateMetadata holds metadata extracted from template files
type TemplateMetadata struct {
	Output string `yaml:"output"`
	Scope  string `yaml:"scope"`
}

func shouldProcess(source, subPath, tag string) (bool, error) {
	_ = subPath
	single := os.Getenv("TB_MAKER_SINGLE")
	if single != "" && !strings.Contains(source, single) {
		// logger.Warn("skipping ", source, " because of ", single)
		return false, nil
	}

	if tag == "" {
		return false, nil
	}

	isSdk := strings.Contains(source, "sdk_")
	isExample := strings.Contains(source, "examples_")
	isPython := strings.Contains(source, "python")
	isTypeScript := strings.Contains(source, "typescript")
	isFuzzer := strings.Contains(source, "sdkFuzzer")
	switch tag {
	case "daemon":
		if isSdk || isFuzzer || isExample {
			return false, nil
		}
	case "scrape":
		if isFuzzer || isExample {
			return false, nil
		}
		fallthrough
	case "explore":
		if isSdk && (isPython || isTypeScript || isFuzzer) || isExample {
			return false, nil
		}
	}

	// source should already be the complete path to the file, no need to modify it
	if !file.FileExists(source) {
		return false, fmt.Errorf("file does not exist %s", source)
	}

	return true, nil
}

// getGeneratorContentsAndDest processes a template file and returns both cleaned content and destination path
func getGeneratorContentsAndDest(fullPath, subPath, group, reason, routeTag, typeTag, groupTag string) (string, string) {
	_ = subPath
	// fullPath should already include the complete path to the generator file
	gPath := fullPath
	if !file.FileExists(gPath) {
		logger.Fatal("Could not find generator file: ", gPath)
	}

	tmpl := file.AsciiFileToString(gPath)
	if err := ValidateTemplate(tmpl, gPath); err != nil {
		logger.Fatal(err)
	}

	// Extract metadata first, before stripping it
	var dest string
	if metadata := parseMetadataBlock(tmpl, reason); metadata != nil {
		dest = processMetadataPath(metadata.Output, routeTag, typeTag, groupTag, reason)
	} else {
		logger.ShouldNotHappen("Old style templates should be gone.")
	}

	// Now strip metadata from template content
	tmpl = stripMetadata(tmpl)

	tmpl = strings.ReplaceAll(tmpl, "[{GROUP}]", group)
	tmpl = strings.ReplaceAll(tmpl, "[{REASON}]", reason)

	return tmpl, dest
}

// stripMetadata removes metadata block from template content and trims whitespace
func stripMetadata(content string) string {
	if !strings.HasPrefix(content, "/*\n") {
		return content
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return content
	}

	// Find the end of the metadata block
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "*/" {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return content // No proper metadata block found
	}

	// Return content after metadata block, trimmed
	remaining := strings.Join(lines[endIndex+1:], "\n")
	return strings.TrimSpace(remaining) + "\n"
}

// parseMetadataBlock parses metadata from a comment block at the start of a template
func parseMetadataBlock(content, reason string) *TemplateMetadata {
	switch reason {
	case "readme":
		content = strings.ReplaceAll(content, "[[reason]]_", "chifra/")
	case "model":
		content = strings.ReplaceAll(content, "[[reason]]_", "data-model/")
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 3 || lines[0] != "/*" {
		return nil
	}

	metadata := &TemplateMetadata{}
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "*/" {
			break
		}

		if strings.HasPrefix(line, "output:") {
			metadata.Output = strings.TrimSpace(strings.TrimPrefix(line, "output:"))
		} else if strings.HasPrefix(line, "scope:") {
			metadata.Scope = strings.TrimSpace(strings.TrimPrefix(line, "scope:"))
		}
	}

	if metadata.Output == "" {
		return nil
	}
	return metadata
}

// processMetadataPath processes Go template variables in a metadata output path
func processMetadataPath(outputPath, routeTag, typeTag, groupTag, reason string) string {
	dest := outputPath

	dest = strings.ReplaceAll(dest, "[[Route]]", Proper(routeTag))
	dest = strings.ReplaceAll(dest, "[[Type]]", Proper(typeTag))
	dest = strings.ReplaceAll(dest, "[[Group]]", Proper(groupTag))
	dest = strings.ReplaceAll(dest, "[[Reason]]", Proper(reason))

	dest = strings.ReplaceAll(dest, "[[route]]", Lower(routeTag))
	dest = strings.ReplaceAll(dest, "[[type]]", Lower(typeTag))
	dest = strings.ReplaceAll(dest, "[[group]]", Lower(groupTag))
	dest = strings.ReplaceAll(dest, "[[reason]]", Lower(reason))

	return dest
}

var rootFolder = "dev-tools/goMaker/"
var cachedTemplatesPath string
var templatesPathError error
var templatesPathOnce sync.Once

func getRootFolder() string {
	return filepath.Join(rootFolder)
}

func setRootFolder(folder string) {
	rootFolder = folder
}

func getTemplatePath() (string, error) {
	templatesPathOnce.Do(func() {
		if envPath := os.Getenv("TB_TEMPLATES_PATH"); envPath != "" {
			if !strings.HasSuffix(envPath, "/") {
				envPath += "/"
			}
			if !strings.HasSuffix(envPath, "templates/") {
				templatesPathError = fmt.Errorf("TB_TEMPLATES_PATH must end with 'templates/', got: %s", envPath)
				return
			}
			if file.FolderExists(envPath) {
				classDefPath := filepath.Join(envPath, "classDefinitions")
				if !file.FolderExists(classDefPath) {
					templatesPathError = fmt.Errorf("TB_TEMPLATES_PATH points to %s but classDefinitions subfolder does not exist", envPath)
					return
				}
				rootPath := strings.TrimSuffix(envPath, "/templates/")
				setRootFolder(rootPath)
				cachedTemplatesPath = envPath
				return
			} else {
				templatesPathError = fmt.Errorf("TB_TEMPLATES_PATH environment variable points to non-existent directory: %s", envPath)
				return
			}
		}

		paths := []string{
			"./code_gen/templates",
			"../dev-tools/goMaker/templates",
			"./dev-tools/goMaker/templates",
		}

		for _, thePath := range paths {
			classDefPath := filepath.Join(thePath, "classDefinitions")
			if file.FolderExists(classDefPath) {
				if strings.HasSuffix(thePath, "/templates") {
					rootPath := strings.TrimSuffix(thePath, "/templates")
					setRootFolder(rootPath)
				} else if strings.HasSuffix(thePath, "templates") {
					rootPath := strings.TrimSuffix(thePath, "templates")
					setRootFolder(rootPath)
				}
				cachedTemplatesPath = thePath
				return
			}
		}

		templatesPathError = fmt.Errorf("could not find templates directory with classDefinitions subfolder in any of: %s", strings.Join(paths, ", "))
	})

	return cachedTemplatesPath, templatesPathError
}

// getTemplatePathNoErr returns the templates path without error handling for backward compatibility
func getTemplatePathNoErr() string {
	thePath, _ := getTemplatePath()
	return thePath
}

// getGeneratorsPath returns the path to the generators folder, checking for TB_GENERATORS_PATH override
func getGeneratorsPath() string {
	if envPath := os.Getenv("TB_GENERATORS_PATH"); envPath != "" {
		if !strings.HasSuffix(envPath, "/") {
			envPath += "/"
		}
		if !file.FolderExists(envPath) {
			logger.Fatal("TB_GENERATORS_PATH env variable points to non-existent folder: %s", envPath)
		}
		if !strings.HasSuffix(envPath, "generators/") {
			logger.Fatal("TB_GENERATORS_PATH must end with 'generators', got: %s", envPath)
		}
		return envPath
	}

	paths := []string{
		"./code_gen/templates/generators",
		"../dev-tools/goMaker/templates/generators",
		"./dev-tools/goMaker/templates/generators",
	}

	for _, genPath := range paths {
		if file.FolderExists(genPath) {
			return genPath
		}
	}

	logger.Fatal("could not find generators directory in any of: %s", strings.Join(paths, ", "))
	return "" // unreachable but needed for compilation
}

func getTemplateContents(fnIn string) string {
	fn := filepath.Join(getTemplatePathNoErr(), fnIn+".md")
	content := file.AsciiFileToString(fn)
	if err := ValidateTemplate(content, fn); err != nil {
		logger.Fatal(err)
	}
	return content
}

// ValidateTemplate validates that EXISTING_CODE markers are properly formatted
func ValidateTemplate(content, templatePath string) error {
	lines := strings.Split(content, "\n")
	existingCodeCount := 0

	for _, line := range lines {
		if strings.Contains(line, "// EXISTING_CODE") {
			// Check rule 1: Line can only contain whitespace and // EXISTING_CODE
			// trimmed := strings.TrimSpace(line)
			// if trimmed != "// EXISTING_CODE" {
			// 	return fmt.Errorf("line %d in template %s contains '// EXISTING_CODE' but has other non-whitespace characters: %s",
			// 		i+1, templatePath, line)
			// }
			existingCodeCount++
		}
	}

	// Check rule 2: Must have even number of EXISTING_CODE markers (including zero)
	if existingCodeCount%2 != 0 {
		return fmt.Errorf("template %s has %d '// EXISTING_CODE' markers, but must have an even number",
			templatePath, existingCodeCount)
	}

	return nil
}

func GetGeneratedPath() string {
	return filepath.Join(getRootFolder(), "generated/")
}

func getGeneratedContents(fnIn string) string {
	fn := filepath.Join(GetGeneratedPath(), fnIn+".md")
	return file.AsciiFileToString(fn)
}

func LowerNoSpaces(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", ""))
}

func GoName(s string) string {
	return FirstUpper(CamelCase(s))
}

func CamelCase(s string) string {
	if len(s) < 2 {
		return s
	}

	result := ""
	toUpper := false
	for _, c := range s {
		if c == '_' {
			toUpper = true
			continue
		}
		if toUpper {
			result += strings.ToUpper(string(c))
			toUpper = false
		} else {
			result += string(c)
		}
	}
	return strings.ReplaceAll(strings.ToLower(result[0:1])+result[1:], " ", "")
}

func Pad(s string, width int) string {
	return s + strings.Repeat(" ", width-len(s))
}

func FirstUpper(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

func FirstLower(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[0:1]) + s[1:]
}

func Plural(s string) string {
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "ed") || strings.HasSuffix(s, "ing") {
		return s
	} else if strings.HasSuffix(s, "x") {
		return s + "es"
	} else if strings.HasSuffix(s, "y") {
		return s + "ies"
	} else if s == "config" || s == "session" || s == "publish" {
		return s
	}
	return s + "s"
}

func Proper(s string) string {
	titleCaser := cases.Title(language.English)
	return titleCaser.String(s)
}

func Singular(s string) string {
	sLower := strings.ToLower(s)
	if sLower == "addresses" {
		return s[:len(s)-2]
	}

	exclusions := []string{"baddress", "status", "stats", "series", "dalledress"}
	if !contains(exclusions, sLower) && strings.HasSuffix(sLower, "s") {
		return s[:len(s)-1]
	}

	return s
}

func Lower(s string) string {
	return strings.ToLower(s)
}

func Upper(s string) string {
	return strings.ToUpper(s)
}

// ws white space
var ws = "\n\r\t"

// wss white space with space
var wss = ws + " "
