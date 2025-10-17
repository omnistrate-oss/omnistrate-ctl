package build

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
)

func renderFile(fileData []byte, rootDir string, file string, sm ysmrr.SpinnerManager, spinner *ysmrr.Spinner) (
	newFileData []byte, err error) {
	newFileData = fileData

	newFileData, err = renderFileReferences(newFileData, file, sm, spinner)
	if err != nil {
		return
	}

	if strings.Contains(string(newFileData), "env_file:") {
		newFileData, err = renderEnvFileAndInterpolateVariables(newFileData, rootDir, file, sm, spinner)
		if err != nil {
			return
		}
	}
	return
}

func renderEnvFileAndInterpolateVariables(
	fileData []byte, rootDir string, file string, sm ysmrr.SpinnerManager, spinner *ysmrr.Spinner) (
	newFileData []byte, err error) {
	// Replace `$` with `$$` to avoid interpolation. Do not replace for `${...}` since it's used to specify variable interpolations
	fileData = []byte(strings.ReplaceAll(string(fileData), "$", "$$"))                // Escape $ to $$
	fileData = []byte(strings.ReplaceAll(string(fileData), "$${", "${"))              // Unescape $${ to ${ for variable interpolation
	fileData = []byte(strings.ReplaceAll(string(fileData), githubPAT, "$"+githubPAT)) // Escape GitHub PAT placeholder

	// Write the compose spec to a temporary file
	tempFile := filepath.Join(rootDir, filepath.Base(file)+".tmp")
	err = os.WriteFile(tempFile, fileData, 0600)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return
	}

	// Render the compose file using docker compose config
	renderCmd := exec.Command("docker", "compose", "-f", tempFile, "config")
	cmdOut := &bytes.Buffer{}
	cmdErr := &bytes.Buffer{}
	renderCmd.Stdout = cmdOut
	renderCmd.Stderr = cmdErr

	err = renderCmd.Run()
	if err != nil {
		if spinner != nil {
			spinner.Error()
			sm.Stop()
		}
		_, _ = fmt.Fprintf(os.Stderr, "%s", cmdErr.String())
		utils.HandleSpinnerError(spinner, sm, err)

		return
	}
	newFileData = cmdOut.Bytes()

	// Remove the temporary file
	err = os.Remove(tempFile)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return
	}

	// Docker compose config command escapes the $ character by adding a $ in front of it, so we need to unescape it
	newFileData = []byte(strings.ReplaceAll(string(newFileData), "$$", "$"))

	// Quote numeric cpus values in deploy.resources
	// Match: cpus: <number> where the number is NOT quoted
	re := regexp.MustCompile(`(?m)(^\s*cpus:\s*)([0-9.]+)\s*$`)
	newFileData = []byte(re.ReplaceAllString(string(newFileData), `$1"$2"`))

	return
}

func renderFileReferences(
	fileData []byte, file string, sm ysmrr.SpinnerManager, spinner *ysmrr.Spinner) (
	newFileData []byte, err error) {
	re := regexp.MustCompile(`(?m)^(?P<indent>[ \t]+)?(?P<key>[\S\t ]+)?{{[ \t]*\$file:(?P<filepath>[^\s}]+)[ \t]*}}`)
	var filePathIndex, indentIndex, keyIndex int
	groupNames := re.SubexpNames()
	for i, name := range groupNames {
		if name == "indent" {
			indentIndex = i
		}
		if name == "key" {
			keyIndex = i
		}
		if name == "filepath" {
			filePathIndex = i
		}
	}

	var renderingErr error
	newFileDataStr := re.ReplaceAllStringFunc(string(fileData), func(match string) (replacement string) {
		replacement = match

		submatches := re.FindStringSubmatch(match)
		addedIndentation := submatches[indentIndex]
		key := submatches[keyIndex]
		if strings.HasPrefix(key, ignoreKeyForFileEmbedding) {
			key = ""
		}

		filePath := submatches[filePathIndex]
		if len(filePath) == 0 {
			renderingErr = fmt.Errorf("no file path found in file reference '%s'", match)
			return
		}

		// Read file content
		cleanedFilePath := filepath.Clean(filePath)
		fileDir := filepath.Dir(file)
		isRelative := !filepath.IsAbs(cleanedFilePath)
		if isRelative {
			cleanedFilePath = filepath.Join(fileDir, cleanedFilePath)
		}

		if _, fileErr := os.Stat(cleanedFilePath); os.IsNotExist(fileErr) {
			renderingErr = fmt.Errorf("file '%s' does not exist", filePath)
			return
		}

		fileContent, readErr := os.ReadFile(cleanedFilePath)
		if readErr != nil {
			renderingErr = fmt.Errorf("file '%s' could not be read", filePath)
			return
		}

		// Render the file (in case it uses nested file references)
		renderedFileContentBytes, nestedRenderErr := renderFileReferences(fileContent, cleanedFilePath, sm, spinner)
		if nestedRenderErr != nil {
			renderingErr = errors.Wrapf(nestedRenderErr,
				"failed to replace file references for file '%s'", filePath)
			return
		}

		// Add indentation of parent context
		replacement = string(renderedFileContentBytes)
		if len(addedIndentation) > 0 {
			// Add indentation to each line
			lines := strings.Split(replacement, "\n")
			for i, line := range lines {
				if i == 0 {
					lines[i] = addedIndentation + key + line
				} else if len(line) > 0 {
					lines[i] = addedIndentation + line
				}
			}
			replacement = strings.Join(lines, "\n")
		} else {
			replacement = key + replacement
		}

		return
	})

	// Handle error
	if renderingErr != nil {
		err = errors.Wrapf(renderingErr, "error rendering file '%s'", file)
		if spinner != nil {
			spinner.Error()
			sm.Stop()
		}
		_, _ = fmt.Fprintf(os.Stderr, "%s", err.Error())
		utils.HandleSpinnerError(spinner, sm, err)

		return
	}

	newFileData = []byte(newFileDataStr)
	return
}
