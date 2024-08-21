package cmd

import (
	"reflect"
	"testing"
)

func TestParseEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		EnvFilePath string
		want        map[string]string
	}{{
		name:        "test_env_parsing",
		EnvFilePath: "../test_data/test.env",
		want: map[string]string{
			"test_var":             "test",
			"secret_var":           "secret",
			"override_default_var": "overridden_val",
		},
	}}

	for _, tt := range tests {
		got, err := parseEnvVars(tt.EnvFilePath)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("outputs didn't match:\ngot:%v\nwant:%v", got, tt.want)
		}
	}
}

func TestParseComponentVar(t *testing.T) {
	envVars, err := parseEnvVars("../test_data/test.env")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	tests := []struct {
		name      string
		varString string
		want      string
	}{
		{
			name:      "basic_string",
			varString: "{{ test_var }} {{test_var}}",
			want:      "test test",
		},
		{
			name:      "bad_spacing",
			varString: "{{ test_var}} {{test_var }}",
			want:      "test test",
		},
		{
			name:      "no_brackets",
			varString: "Hello, world!",
			want:      "Hello, world!",
		},
		{
			name:      "single_brackets",
			varString: "{ test_var } {test_var}",
			want:      "{ test_var } {test_var}",
		},
		{
			name:      "a_lot_of_spaces",
			varString: "{{                             test_var}} {{  secret_var                                       }}",
			want:      "test secret",
		},
		{
			name: "mixed_brackets",
			varString: "Hello, world this is a {{ test_var }}, and not a {{secret_var}}. " +
				"With luck we'll also have ourselves an {{override_default_var}}. " +
				"Here are some other examples: {{ secret_var}} {{test_var }} { secret_var } {test_var}",
			want: "Hello, world this is a test, and not a secret. " +
				"With luck we'll also have ourselves an overridden_val. " +
				"Here are some other examples: secret test { secret_var } {test_var}",
		},
	}

	for _, tt := range tests {
		got, err := parseComponentVar(tt.varString, envVars)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if got != tt.want {
			t.Errorf("outputs didn't match:\ngot:%v\nwant:%v", got, tt.want)
		}
	}
}
