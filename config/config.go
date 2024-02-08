package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

func Fatalf(msg string, params ...any) {
	log.Printf(msg, params...)
	panic("error")
}

var (
	variables = map[string]*environmentVariable{
		// special case environment as we need it when setting all other config
		"ENVIRONMENT": {
			Key:          "ENVIRONMENT",
			StringValue:  setEnvironment(),
			VariableType: variableTypeString,
			IsSecret:     false,
		},
	}

	variableTypeString = "string"
	variableTypeBool   = "bool"
	variableTypeInt    = "int"
)

type AppType string
type EnvironmentType string

const (
	AppTypeApi    AppType = "api"
	AppTypeWorker AppType = "worker"

	Production      EnvironmentType = "production"
	ProductionShort EnvironmentType = "prod"
	Staging         EnvironmentType = "staging"
	StagingShort    EnvironmentType = "stage"
)

type EnvironmentVariableConfig struct {
	Key                  string
	DefaultValue         any
	IsSecret             bool
	IsRequiredInLiveEnv  bool
	IsRequiredInLocalEnv bool
	OnlyInType           AppType
}

type environmentVariable struct {
	Key          string
	VariableType string
	StringValue  string
	BoolValue    bool
	IntValue     int
	IsSecret     bool
}

func IsRegistered(key string) bool {
	return variables[key] != nil
}

func registerString(config *EnvironmentVariableConfig) {
	value, ok := os.LookupEnv(config.Key)
	if !ok {
		if IsLiveEnvironment() && config.IsRequiredInLiveEnv {
			Fatalf("No value found for required environment variable %s", config.Key)
		}
		if IsLocalEnvironment() && config.IsRequiredInLocalEnv {
			Fatalf("No value found for required environment variable %s", config.Key)
		}
		value, ok = config.DefaultValue.(string)
		if !ok {
			Fatalf("Invalid type for default variable %s, expected string got %T", config.Key, config.DefaultValue)
		}
	}

	variables[config.Key] = &environmentVariable{
		Key:          config.Key,
		VariableType: variableTypeString,
		StringValue:  value,
		IsSecret:     config.IsSecret,
	}
}

func registerBoolean(config *EnvironmentVariableConfig) {
	var boolValue bool
	value, ok := os.LookupEnv(config.Key)
	if !ok {
		if IsLiveEnvironment() && config.IsRequiredInLiveEnv {
			Fatalf("No value found for required environment variable %s", config.Key)
		}
		if IsLocalEnvironment() && config.IsRequiredInLocalEnv {
			Fatalf("No value found for required environment variable %s", config.Key)
		}
		boolValue, ok = config.DefaultValue.(bool)
		if !ok {
			Fatalf("Invalid type for default variable %s, expected bool got %T", config.Key, config.DefaultValue)
		}
	} else {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			Fatalf("Invalid value found for required environment variable %s, could not parse %s as boolean", config.Key, value)
		}
		boolValue = parsed
	}

	variables[config.Key] = &environmentVariable{
		Key:          config.Key,
		VariableType: variableTypeBool,
		BoolValue:    boolValue,
		IsSecret:     config.IsSecret,
	}
}

func registerInt(config *EnvironmentVariableConfig) {
	value, ok := os.LookupEnv(config.Key)
	var intValue int
	if !ok {
		if IsLiveEnvironment() && config.IsRequiredInLiveEnv {
			Fatalf("No value found for required environment variable %s", config.Key)
		}
		if IsLocalEnvironment() && config.IsRequiredInLocalEnv {
			Fatalf("No value found for required environment variable %s", config.Key)
		}
		intValue, ok = config.DefaultValue.(int)
		if !ok {
			Fatalf("Invalid type for default variable %s, expected int got %T", config.Key, config.DefaultValue)
		}
	} else {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			Fatalf("Invalid value found for required environment variable %s, could not parse %s as int", config.Key, value)
		}
		intValue = parsed
	}

	variables[config.Key] = &environmentVariable{
		Key:          config.Key,
		VariableType: variableTypeInt,
		IntValue:     intValue,
		IsSecret:     config.IsSecret,
	}
}

func GetString(key string) string {
	value := variables[key]
	if value == nil {
		Fatalf("Environment variable %s hasn't been registered. Did you call RegisterEnvironmentVariable?", key)
	}
	if value.VariableType != variableTypeString {
		Fatalf("Environment variable %s has type %s, cannot access as string", key, value.VariableType)
	}
	return value.StringValue
}

func GetBool(key string) bool {
	value := variables[key]
	if value == nil {
		Fatalf("Environment variable %s hasn't been registered. Did you call RegisterEnvironmentVariable?", key)
	}
	if value.VariableType != variableTypeBool {
		Fatalf("Environment variable %s has type %s, cannot access as bool", key, value.VariableType)
	}
	return value.BoolValue
}

func GetInt(key string) int {
	value := variables[key]
	if value == nil {
		Fatalf("Environment variable %s hasn't been registered. Did you call RegisterEnvironmentVariable?", key)
	}
	if value.VariableType != variableTypeInt {
		Fatalf("Environment variable %s has type %s, cannot access as int", key, value.VariableType)
	}
	return value.IntValue
}

func Dump(output io.Writer) {
	// sort by key alphabetically to make it easier to read
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fmt.Fprint(output, "\n\n--------- Environment Variables ---------\n")
	for _, key := range keys {
		variable := variables[key]
		switch {
		case variable.VariableType == variableTypeString:
			if variable.IsSecret && variable.StringValue != "" {
				fmt.Fprintf(output, " %s: ********\n", variable.Key)
			} else {
				fmt.Fprintf(output, " %s: %s\n", variable.Key, variable.StringValue)
			}
		case variable.VariableType == variableTypeInt:
			if variable.IsSecret {
				fmt.Fprintf(output, " %s: ********\n", variable.Key)
			} else {
				fmt.Fprintf(output, " %s: %d\n", variable.Key, variable.IntValue)
			}
		case variable.VariableType == variableTypeBool:
			if variable.IsSecret {
				fmt.Fprintf(output, " %s: ********\n", variable.Key)
			} else {
				fmt.Fprintf(output, " %s: %v\n", variable.Key, variable.BoolValue)
			}
		}
	}
	fmt.Fprint(output, "-----------------------------------------\n\n")
}

func IsLocalEnvironment() bool {
	return strings.ToLower(GetString("ENVIRONMENT")) == "local"
}

func IsProductionEnvironment() bool {
	switch {
	case strings.ToLower(GetString("ENVIRONMENT")) == "production":
		return true
	case strings.ToLower(GetString("ENVIRONMENT")) == "prod":
		return true
	default:
		return false
	}
}

func IsEnvironmentType(environments map[string]EnvironmentType) bool {
	if _, ok := environments[strings.ToLower(GetString("ENVIRONMENT"))]; !ok {
		return false
	}
	return true
}

func IsTestEnvironment() bool {
	return strings.ToLower(GetString("ENVIRONMENT")) == "test"
}

func IsLiveEnvironment() bool {
	return !IsLocalEnvironment() && !IsTestEnvironment()
}

func setEnvironment() string {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test") {
			return "test"
		}
	}
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	return "production"
}
