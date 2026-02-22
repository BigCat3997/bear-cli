package models

type Command string

const (
	Ps                 Command = "ps"
	PsCreateCredential Command = "create-cred"
	PsGetCredential    Command = "get-cred"
	PsInitCredential   Command = "init-cred"
	PsLoginByCred      Command = "login-by-cred"
)

var CommandDescriptions = map[Command]string{
	Ps:                 "Interact with PluralSight resources",
	PsCreateCredential: "Creates credential",
	PsGetCredential:    "Get credential",
	PsInitCredential:   "Initialize credential in every consumed environment (e.g. Terraform's variables.tf, etc.)",
	PsLoginByCred:      "Log in to appropriate cloud by credential",
}
