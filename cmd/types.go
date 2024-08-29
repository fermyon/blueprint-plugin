package cmd

import (
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
)

type SpinTOML struct {
	// Applies to all components
	Application Application `toml:"application"`

	// The individual component triggers
	Trigger Trigger `toml:"trigger"`

	// The individual component details
	Component map[string]Component `toml:"component"`

	// The top-level variables resolved via environment variables
	Variables map[string]Variable `toml:"variables"`
}

type Variable struct {
	Default  string `toml:"default"`
	Secret   bool   `toml:"secret"`
	Required bool   `toml:"required"`
}

type Application struct {
	Name        string             `toml:"name"`
	Version     string             `toml:"version"`
	Authors     []string           `toml:"authors"`
	Description string             `toml:"description"`
	Trigger     ApplicationTrigger `toml:"trigger"`
}

type ApplicationTrigger struct {
	HTTP  ApplicationTriggerHTTP  `toml:"http"`
	Redis ApplicationTriggerRedis `toml:"redis"`
}

type ApplicationTriggerHTTP struct {
	Base string `toml:"base"`
}

type ApplicationTriggerRedis struct {
	Address string `toml:"address"`
}

type Trigger struct {
	HTTP  []HTTPTrigger
	Redis []RedisTrigger
	Other []OtherTrigger
}

type RedisTrigger struct {
	Address   string `toml:"address"`
	Channel   string `toml:"channel"`
	Component string `toml:"component"`
}

type HTTPTrigger struct {
	Route     Route    `toml:"route"`
	Component string   `toml:"component"`
	Executor  Executor `toml:"executor"`
}

type OtherTrigger struct {
	// The assumption is that all triggers must have a component defined
	Component string `toml:"component"`
	// This is a hacky solution for storing the type of the trigger
	// This is done in the UnmarshalTOML function for the Trigger type.
	TriggerType string
}

// UnmarshalTOML (for *Trigger) is a function that the `toml` package will call when
// it encounters a trigger data structure. It must be named UnmarshalTOML, regardless of the type
func (t *Trigger) UnmarshalTOML(rawData any) error {
	data, ok := rawData.(map[string]any)
	if !ok {
		return fmt.Errorf("malformed data, expected map")
	}
	for key, value := range data {
		switch key {
		case "http":
			var httpTriggers []HTTPTrigger
			decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				DecodeHook: routeDecodeHook,
				Result:     &httpTriggers,
			})
			if err != nil {
				return err
			}

			decoder.Decode(value)
			t.HTTP = append(t.HTTP, httpTriggers...)
		case "redis":
			var redisTriggers []RedisTrigger
			err := mapstructure.Decode(value, &redisTriggers)
			if err != nil {
				return fmt.Errorf("failed to map trigger %q: %w", key, err)
			}
			t.Redis = append(t.Redis, redisTriggers...)
		default:
			var otherTriggers []OtherTrigger
			err := mapstructure.Decode(value, &otherTriggers)
			if err != nil {
				return fmt.Errorf("failed to map trigger %q: %w", key, err)
			}

			// Adding the trigger type (which is the key) to the struct
			for i := range otherTriggers {
				otherTriggers[i].TriggerType = key
			}

			t.Other = append(t.Other, otherTriggers...)
		}
	}

	return nil
}

// This was a ChatGPT generated function, and helped fix errors with the route struct.
// Not totally sure why this works while a regular unmarshal is insufficient
func routeDecodeHook(from reflect.Type, to reflect.Type, data any) (any, error) {
	if to != reflect.TypeOf(Route{}) {
		return data, nil
	}

	switch from.Kind() {
	case reflect.String:
		return Route{String: data.(string)}, nil
	case reflect.Map:
		mapData := data.(map[string]any)
		routeStruct := &struct {
			Private bool
		}{}
		if err := mapstructure.Decode(mapData, routeStruct); err != nil {
			return nil, fmt.Errorf("error decoding route map: %w", err)
		}
		return Route{Struct: routeStruct}, nil
	default:
		return nil, fmt.Errorf("invalid type for Route: %s", from)
	}
}

// This needs to be a pointer; otherwise, the private field will not be read properly
type Route struct {
	String string
	Struct *struct {
		Private bool
	}
}

// UnmarshalTOML (for *Route) is a function that the `toml` package will call when
// it encounters a Route data structure. It must be named UnmarshalTOML, regardless of the type
func (r *Route) UnmarshalTOML(data any) error {
	switch v := data.(type) {
	case string:
		r.String = v
	case map[string]any:
		private, ok := v["private"].(bool)
		if !ok {
			return fmt.Errorf("expected 'private' to be a bool")
		}
		r.Struct = &struct {
			Private bool
		}{
			Private: private,
		}
	default:
		return fmt.Errorf("invalid type for Route: %T", v)
	}
	return nil
}

type Executor struct {
	Type string `toml:"type"`
}

type Component struct {
	Description          string            `toml:"description"`
	Source               Source            `toml:"source"`
	Variables            map[string]string `toml:"variables"`
	AllowedOutboundHosts []string          `toml:"allowed_outbound_hosts"`
	KeyValueStores       []string          `toml:"key_value_stores"`
	AIModels             []string          `toml:"ai_models"`
	SQLiteDatabases      []string          `toml:"sqlite_databases"`
}

type Source struct {
	String string
	Struct *struct {
		URL    string `toml:"url"`
		Digest string `toml:"digest"`
	}
}

// UnmarshalTOML (for *Source) is a function that the `toml` package will call when
// it encounters a Source data structure. It must be named UnmarshalTOML, regardless of the type
func (s *Source) UnmarshalTOML(data any) error {
	switch v := data.(type) {
	case string:
		s.String = v
	case map[string]any:
		url, ok := v["url"].(string)
		if !ok {
			return fmt.Errorf("expected URL to be a string")
		}
		digest, ok := v["digest"].(string)
		if !ok {
			return fmt.Errorf("expected Digest to be a string")
		}
		s.Struct = &struct {
			URL    string `toml:"url"`
			Digest string `toml:"digest"`
		}{
			URL:    url,
			Digest: digest,
		}
	default:
		return fmt.Errorf("invalid type for Source: %T", v)
	}
	return nil
}
