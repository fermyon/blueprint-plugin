package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseSpinTOML(t *testing.T) {
	tests := []struct {
		name         string
		tomlFilePath string
		want         SpinTOML
	}{{
		name:         "test_toml_parsing",
		tomlFilePath: "../test_data/spin.toml",
		want: SpinTOML{
			Variables: map[string]Variable{
				"test_var":             {Default: "test"},
				"secret_var":           {Secret: true},
				"override_default_var": {Default: "some_val"},
				"missing_default_var":  {},
				"missing_required_var": {Required: true},
				"test_default_var":     {Default: "another_val"},
			},
			Application: Application{
				Name:        "Test Spin TOML",
				Version:     "0.1.0",
				Authors:     []string{"Fermyon Engineering Team <engineering@fermyon.com>"},
				Description: "This is a test spin.toml file used for testing the blueprint plugin.",
				Trigger: ApplicationTrigger{
					HTTP: ApplicationTriggerHTTP{
						Base: "/blueprint",
					},
					Redis: ApplicationTriggerRedis{
						Address: "redis://localhost:6379",
					},
				},
			},
			Trigger: Trigger{
				HTTP: []HTTPTrigger{
					{
						Route: Route{
							String: "/route-one/...",
						},
						Component: "number-one",
						Executor: Executor{
							Type: "spin",
						},
					},
					{
						Route: Route{
							Struct: &struct {
								Private bool
							}{
								Private: true,
							},
						},
						Component: "number-two",
						Executor: Executor{
							Type: "non-Spin executor",
						},
					},
				},
				Redis: []RedisTrigger{
					{
						Address:   "redis://anotherhost.io:6379",
						Channel:   "test-channel",
						Component: "number-two",
					},
					{
						Channel:   "root-channel",
						Component: "number-three",
					},
				},

				Other: []OtherTrigger{
					{TriggerType: "random", Component: "number-three"},
				},
			},
			Component: map[string]Component{
				"number-one": {
					Description: "This is a description for component 1.",
					Variables: map[string]string{
						"parsed_test_var":     "This is the test_var: {{ test_var }}",
						"parsed_secret_var":   "This is the secret_var: {{ secret_var }}",
						"parsed_optional_var": "This is the test_default_var: {{ test_default_var }}",
					},
					Source: Source{
						String: "component-one/main.wasm",
					},
					AllowedOutboundHosts: []string{"https://localhost:3000", "postgres://localhost:5432"},
					KeyValueStores:       []string{"redis://localhost:6379"},
					AIModels:             []string{"gpt4_wrapper"},
					SQLiteDatabases:      []string{"default"},
				},
				"number-two": {
					Source: Source{
						Struct: &struct {
							URL    string "toml:\"url\""
							Digest string "toml:\"digest\""
						}{
							URL:    "https://ghcr.io/fermyon/component-number-two",
							Digest: "thisisatestdigeststring",
						},
					},
				},
				"number-three": {
					Source: Source{
						String: "component-three/main.wasm",
					},
				},
			},
		},
	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSpinToml(tt.tomlFilePath)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// cmp.Diff ensures that the pointer structs (i.e. Route and Source) are not compared, as they will have different memory locations
			if diff := cmp.Diff(tt.want, *got, cmpopts.IgnoreUnexported(Route{}, Source{})); diff != "" {
				t.Errorf("parseSpinToml() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
