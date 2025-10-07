package config

import (
	"strings"
)

type Environment int32

const (
	UNDEFINED_ENV Environment = iota
	LOCAL_ENV
	DEV_ENV
	UAT_ENV
	PROD_ENV
)

func StringToEnvironment(s string) Environment {
	switch strings.ToLower(s) {
	case "local":
		return LOCAL_ENV
	case "dev":
		return DEV_ENV
	case "uat":
		return UAT_ENV
	case "prod":
		return PROD_ENV
	default:
		return UNDEFINED_ENV
	}
}

func EnvironmentToString(e Environment) string {
	switch e {
	case LOCAL_ENV:
		return "local"
	case DEV_ENV:
		return "dev"
	case UAT_ENV:
		return "uat"
	case PROD_ENV:
		return "prod"
	default:
		return "UNDEFINED"
	}
}
