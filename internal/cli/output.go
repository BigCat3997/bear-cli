package cli

type StdOutFormat string

const (
	JSON          StdOutFormat = "JSON"
	TABLE         StdOutFormat = "TABLE"
	LINUX_ENV_VAR StdOutFormat = "LINUX_ENV_VAR"
	NONE          StdOutFormat = "NONE"
)

func ParseStdOutFormat(s string) StdOutFormat {
	switch s {
	case "json":
		return JSON
	case "table":
		return TABLE
	case "env":
		return LINUX_ENV_VAR
	case "none":
		return NONE
	default:
		return LINUX_ENV_VAR
	}
}
