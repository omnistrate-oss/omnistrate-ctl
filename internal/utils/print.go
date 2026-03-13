package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
)

var (
	LastPrintedString string
)

func PrintError(err error) {
	errorMsg := color.New(color.FgRed, color.Bold).SprintFunc()
	msg := fmt.Sprintf("%s %s", errorMsg("Error: "), err.Error())
	fmt.Fprintln(os.Stderr, msg)
	if !config.IsDryRun() {
		os.Exit(1)
	}
}

func PrintSuccess(msg string) {
	successMsg := color.New(color.FgGreen, color.Bold).SprintFunc()
	formatted := successMsg(msg)
	fmt.Println(formatted)
}

func PrintInfo(msg string) {
	infoMsg := color.New(color.FgCyan).SprintFunc()
	formatted := infoMsg(msg)
	fmt.Println(formatted)
}

func PrintWarning(msg string) {
	warningMsg := color.New(color.FgYellow).SprintFunc()
	formatted := warningMsg(msg)
	fmt.Println(formatted)
}

func PrintURL(label, url string) {
	urlMsg := color.New(color.FgCyan).SprintFunc()
	formatted := fmt.Sprintf("%s: %s", label, urlMsg(url))
	fmt.Println(formatted)
}

func PrintJSON(res any) {
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		PrintError(err)
		return
	}
	formatted := string(data)
	fmt.Println(formatted)
}

func PrintTextTableJsonArrayOutput[T any](output string, objects []T) error {
	switch output {
	case "text":
		dataArray := make([]string, 0)
		for _, obj := range objects {
			data, err := json.MarshalIndent(obj, "", "    ")
			if err != nil {
				return err
			}
			dataArray = append(dataArray, string(data))
		}
		err := PrintText(dataArray)
		if err == nil {
			LastPrintedString = fmt.Sprintf("%v", dataArray)
		}
		return err
	case "table":
		dataArray := make([]string, 0)
		for _, obj := range objects {
			data, err := json.MarshalIndent(obj, "", "    ")
			if err != nil {
				return err
			}
			dataArray = append(dataArray, string(data))
		}
		err := PrintTable(dataArray)
		if err == nil {
			LastPrintedString = fmt.Sprintf("%v", dataArray)
		}
		return err
	case "json":
		data, err := json.MarshalIndent(objects, "", "    ")
		if err != nil {
			return err
		}
		if len(objects) == 0 {
			formatted := "[]"
			fmt.Println(formatted)
			LastPrintedString = formatted
		} else {
			formatted := string(data)
			fmt.Printf("%s\n", formatted)
			LastPrintedString = formatted
		}
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
	return nil
}

func PrintTextTableJsonOutput[T any](output string, object T) error {
	data, err := json.MarshalIndent(object, "", "    ")
	if err != nil {
		return err
	}

	switch output {
	case "text":
		err := PrintText([]string{string(data)})
		if err == nil {
			LastPrintedString = string(data)
		}
		return err
	case "table":
		err := PrintTable([]string{string(data)})
		if err == nil {
			LastPrintedString = string(data)
		}
		return err
	case "json":
		formatted := string(data)
		fmt.Printf("%s\n", formatted)
		LastPrintedString = formatted
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
	return nil
}

func PrintTextTableYamlOutput[T any](object T) error {
	data, err := yaml.Marshal(object)
	if err != nil {
		return err
	}

	formatted := string(data)
	fmt.Print(formatted)
	LastPrintedString = formatted
	return nil
}

func WriteYAMLToFile[T any](filePath string, object T) error {
	data, err := yaml.Marshal(object)
	if err != nil {
		return fmt.Errorf("failed to marshal object to YAML: %w", err)
	}

	err = os.WriteFile(filePath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write YAML to file %s: %w", filePath, err)
	}

	PrintSuccess(fmt.Sprintf("Template saved to %s", filePath))
	return nil
}

// ConfirmAction displays a Yes/No confirmation prompt and returns true if the user selects Yes.
func ConfirmAction(message string) (bool, error) {
	var confirmed bool
	err := huh.NewConfirm().
		Title(message).
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed).
		Run()
	if err != nil {
		return false, err
	}
	return confirmed, nil
}

// PromptSelect displays a selection prompt and returns the chosen option.
func PromptSelect(title string, options []string) (string, error) {
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	var choice string
	err := huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&choice).
		Run()
	if err != nil {
		return "", err
	}
	return choice, nil
}

// PromptInput displays a text input prompt with optional validation.
func PromptInput(title, placeholder string, validate func(string) error) (string, error) {
	var value string
	field := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Value(&value)
	if validate != nil {
		field = field.Validate(validate)
	}
	err := field.Run()
	if err != nil {
		return "", err
	}
	return value, nil
}

// PromptPassword displays a masked password input prompt.
func PromptPassword(title, placeholder string) (string, error) {
	var value string
	err := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		EchoMode(huh.EchoModePassword).
		Value(&value).
		Run()
	if err != nil {
		return "", err
	}
	return value, nil
}

// ValidateEmail returns a validation function that checks for a valid email address.
func ValidateEmail(input string) error {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	if emailRegex.MatchString(input) {
		return nil
	}
	return fmt.Errorf("invalid email address")
}
