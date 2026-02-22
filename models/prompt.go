package models

type StdOutFormat string

const (
	JSON          StdOutFormat = "JSON"
	TABLE         StdOutFormat = "TABLE"
	LINUX_ENV_VAR StdOutFormat = "LINUX_ENV_VAR"
)

func ParseStdOutFormat(s string) StdOutFormat {
	switch s {
	case "json":
		return JSON
	case "table":
		return TABLE
	case "env":
		return LINUX_ENV_VAR
	default:
		return LINUX_ENV_VAR
	}
}