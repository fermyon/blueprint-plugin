package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [component]",
	Short: "Display a table of components from the specified spin.toml file",
	Long: `The "show" command reads a spin.toml file and prints a table of components to the terminal.
You can optionally specify a component to display information for a specific component only.
By default, the command looks for a "spin.toml" file in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// The path to a "spin.toml" file
		path, err := cmd.Flags().GetString("file")
		if err != nil {
			return err
		}

		// The path to a ".env" file (parseEnvVars will handle blank paths)
		env, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if path == "" {
			path = "spin.toml"
		}

		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("the path %q does not exist", path)
			}
			return err
		}

		tomlData, err := parseSpinToml(path)
		if err != nil {
			return err
		}

		envVars, err := parseEnvVars(env)
		if err != nil {
			return err
		}

		if len(args) == 0 {
			// This won't throw errors because we are not checking the validity of a "spin.toml" file
			fmt.Print(showAllComponents(tomlData, envVars))

		// Also print info about all components if --all flag is set
		if All {
			for name, _ := range tomlData.Component {
				fmt.Print(showSpecificComponent(tomlData, envVars, name))
			}
		}

		} else {
			terminalOutput, err := showSpecificComponent(tomlData, envVars, args[0])
			if err != nil {
				return err
			}

			fmt.Print(terminalOutput)
		}

		return nil
	},
}

// showAllComponents will display a table with all the components listed in a "spin.toml" file
func showAllComponents(tomlData *SpinTOML, envVars map[string]string) string {
	componentTable := table.NewWriter()
	componentTable.SetTitle("Components")
	componentTable.AppendHeader(table.Row{"name", "source"})

	var annotations []string
	if tomlData.Application.Name != "" {
		annotations = append(annotations, "* Name: "+tomlData.Application.Name)
	}
	if tomlData.Application.Version != "" {
		annotations = append(annotations, "* Version: "+tomlData.Application.Version)
	}
	if tomlData.Application.Description != "" {
		annotations = append(annotations, "* Description: "+tomlData.Application.Description)
	}
	if len(tomlData.Application.Authors) > 0 {
		annotations = append(annotations, "* Authors: "+strings.Join(tomlData.Application.Authors, ", "))
	}

	for name, data := range tomlData.Component {
		var source string

		if data.Source.String != "" {
			source = data.Source.String
		} else if data.Source.Struct != nil && data.Source.Struct.URL != "" {
			source = data.Source.Struct.URL
		}

		componentTable.AppendRow(table.Row{name, source})
	}

	variableTable := table.NewWriter()
	variableTable.SetTitle("Variables")
	variableTable.AppendHeader(table.Row{"env_key", "env_value", "is_required", "is_secret", "is_default"})
	for varKey, varData := range tomlData.Variables {
		var matchExists bool
		for envKey, envValue := range envVars {
			if varKey == envKey {
				matchExists = true
				// In the case where someone passes in a env var value that matches the default value,
				// this will show false because this is using the env var value.
				// ("is_default" == true) only applies to nothing being passed via env vars.
				variableTable.AppendRow(table.Row{envKey, envValue, varData.Required, varData.Secret, "false"})
			}
		}

		if !matchExists {
			if varData.Required {
				variableTable.AppendRow(table.Row{varKey, "ERR: MISSING REQUIRED VALUE", true, "n/a", "n/a"})
			} else if varData.Default == "" {
				variableTable.AppendRow(table.Row{varKey, "ERR: ENV VAR NOT FOUND, DEFAULT NOT DEFINED", false, "n/a", "n/a"})
			} else {
				variableTable.AppendRow(table.Row{varKey, varData.Default, varData.Required, varData.Secret, true})
			}
		}
	}

	// Creating the terminal output
	outputString := "\n" +
		strings.Join(annotations, "\n") +
		"\n\n" +
		componentTable.Render()

	if len(envVars) > 0 {
		outputString += "\n\n" + variableTable.Render()
	}

	return outputString
}

// showSpecificComponent will show several tables with details about a specific component
func showSpecificComponent(tomlData *SpinTOML, envVars map[string]string, componentName string) (string, error) {
	componentData, ok := tomlData.Component[componentName]
	if !ok {
		return "", fmt.Errorf("component %q does not exist", componentName)
	}

	// Redis trigger table
	redisTable := table.NewWriter()
	redisTable.SetTitle("Redis Triggers")
	redisTable.AppendHeader(table.Row{"Address", "Channel"})
	var countRedis int // This is used to ensure tables with no data are not printed
	for _, redisTrigger := range tomlData.Trigger.Redis {
		if redisTrigger.Component == componentName {
			countRedis++
			var address string
			// TODO: Is this assumption correct, or can a component be triggered by redis if they don't subscribe to a channel?
			// If the component Redis trigger address is blank, we assume they are subscribing to the application Redis trigger
			if redisTrigger.Address == "" {
				address = tomlData.Application.Trigger.Redis.Address
			} else {
				address = redisTrigger.Address
			}

			redisTable.AppendRow(table.Row{address, redisTrigger.Channel})
		}
	}

	// HTTP trigger table
	HTTPTable := table.NewWriter()
	HTTPTable.SetTitle("HTTP Triggers")
	HTTPTable.AppendHeader(table.Row{"Route", "Executor"})
	var countHTTP int // This is used to ensure tables with no data are not printed
	for _, HTTPTrigger := range tomlData.Trigger.HTTP {
		if HTTPTrigger.Component == componentName {
			countHTTP++
			var route string
			if HTTPTrigger.Route.String != "" { // If the route is a string, handle accordingly
				// Prepending the application base route (even if blank)
				route = tomlData.Application.Trigger.HTTP.Base + HTTPTrigger.Route.String
			} else if HTTPTrigger.Route.Struct != nil && HTTPTrigger.Route.Struct.Private { // If the route is private, handle accordingly
				route = "Private"
			}

			if HTTPTrigger.Executor.Type == "" {
				HTTPTrigger.Executor.Type = "spin"
			}

			HTTPTable.AppendRow(table.Row{route, HTTPTrigger.Executor.Type})
		}
	}

	// Other trigger table
	otherTable := table.NewWriter()
	otherTable.SetTitle("Other Triggers")
	var countOther int // This is used to ensure tables with no data are not printed
	for _, otherTrigger := range tomlData.Trigger.Other {
		if otherTrigger.Component == componentName {
			countOther++
			// Fixing the formatting of this table by ensuring each
			// row text length is not shorter than the title length
			lenDiff := len(otherTrigger.TriggerType) - len("Other Triggers")
			if lenDiff < 0 {
				for i := lenDiff; i <= 0; i++ {
					otherTrigger.TriggerType += " "
				}
			}
			otherTable.AppendRow(table.Row{otherTrigger.TriggerType})
		}
	}

	// Variables table
	variableTable := table.NewWriter()
	variableTable.SetTitle("Variables")
	variableTable.AppendHeader(table.Row{"var_key", "var_value"})

	// Set any missing default value in the envVars map
	for varKey, varData := range tomlData.Variables {
		var matchExists bool
		for envKey := range envVars {
			if varKey == envKey {
				matchExists = true
			}
		}

		if !matchExists {
			if varData.Default != "" {
				envVars[varKey] = varData.Default
			}
		}
	}

	// Parse the component variable templates
	for compVarKey, compVarValue := range tomlData.Component[componentName].Variables {
		parsedVal, err := parseComponentVar(compVarValue, envVars)
		if err != nil {
			return "", err
		}

		variableTable.AppendRow(table.Row{compVarKey, parsedVal})
	}

	// Outbound resources table
	outboundTable := table.NewWriter()
	outboundTable.SetTitle("Outbound Resources")
	outboundTable.AppendHeader(table.Row{"Type", "Value"})

	for _, obHost := range componentData.AllowedOutboundHosts {
		outboundTable.AppendRow(table.Row{"Outbound Host", obHost})
	}
	for _, kvStore := range componentData.KeyValueStores {
		outboundTable.AppendRow(table.Row{"KV", kvStore})
	}
	for _, sqliteDB := range componentData.SQLiteDatabases {
		outboundTable.AppendRow(table.Row{"SQLite", sqliteDB})
	}
	for _, aiModel := range componentData.AIModels {
		outboundTable.AppendRow(table.Row{"AI", aiModel})
	}

	// Annotations
	var annotations []string
	annotations = append(annotations, "* Name: "+componentName)

	if componentData.Description != "" {
		annotations = append(annotations, "* Description: "+componentData.Description)
	}

	if componentData.Source.String != "" {
		annotations = append(annotations, "* Source: "+componentData.Source.String)
		annotations = append(annotations, "* Source Digest: n/a")
	} else {
		annotations = append(annotations, "* Source : "+componentData.Source.Struct.URL)
		annotations = append(annotations, "* Source Digest: "+componentData.Source.Struct.Digest)
	}

	// Creating the terminal output
	outputString := "\n" +
		strings.Join(annotations, "\n")

	// This is used to ensure components with with no outbound sources don't print this table
	if len(componentData.AIModels) > 0 ||
		len(componentData.SQLiteDatabases) > 0 ||
		len(componentData.AllowedOutboundHosts) > 0 ||
		len(componentData.KeyValueStores) > 0 {
		outputString += "\n\n" + outboundTable.Render()
	}

	if countHTTP > 0 {
		outputString += "\n\n" + HTTPTable.Render()
	}

	if countRedis > 0 {
		outputString += "\n\n" + redisTable.Render()
	}

	if countOther > 0 {
		outputString += "\n\n" + otherTable.Render()
	}

	if len(tomlData.Component[componentName].Variables) > 0 {
		outputString += "\n\n" + variableTable.Render()
	}

	return outputString, nil
}

func parseSpinToml(filePath string) (*SpinTOML, error) {
	var tomlFile *SpinTOML
	if _, err := toml.DecodeFile(filePath, &tomlFile); err != nil {
		return nil, err
	}

	return tomlFile, nil
}

func parseComponentVar(varString string, envVars map[string]string) (string, error) {
	// This finds any substring that begins with "{{" and ends with "}}", regardless of whitespace in between
	re := regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

	// This standardizes the formatting of the strings to {{ .key }} so the template engine can parse the templates to variables
	varString = re.ReplaceAllStringFunc(varString, func(match string) string {
		trimmedVar := strings.TrimSpace(match[2 : len(match)-2]) // extract and trim the variable name
		return fmt.Sprintf("{{ .%s }}", trimmedVar)
	})

	tmpl, err := template.New("tmpl").Parse(varString)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	var result strings.Builder
	// Use the template engine to substitute
	if err := tmpl.Execute(&result, envVars); err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return result.String(), nil
}

func parseEnvVars(filePath string) (map[string]string, error) {
	envVars := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	processEnvVar := func(envVar string) {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[0], "SPIN_VARIABLE_") {
			// removing the "SPIN_VARIABLE_" prefix and setting it to lowercase
			parts[0] = strings.ToLower(strings.Split(parts[0], "SPIN_VARIABLE_")[1])
			mu.Lock()
			envVars[parts[0]] = parts[1]
			mu.Unlock()
		}
	}

	if filePath == "" {
		vars := os.Environ()

		for _, v := range vars {
			wg.Add(1)
			go func(envVar string) {
				defer wg.Done()
				processEnvVar(envVar)
			}(v)
		}
	} else {
		if !strings.HasSuffix(filePath, ".env") {
			return nil, fmt.Errorf("the path provided appears not to be a \".env\" file: %q", filePath)
		}

		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			// Ignore comments and empty lines
			line = strings.TrimSpace(line)
			if len(line) == 0 || strings.HasPrefix(line, "#") {
				continue
			}

			wg.Add(1)
			go func(envVar string) {
				defer wg.Done()
				processEnvVar(envVar)
			}(line)
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	wg.Wait()
	return envVars, nil
}
