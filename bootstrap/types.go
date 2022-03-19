package bootstrap

import vault "github.com/hashicorp/vault/api"

type vaultPod struct {
	name   string
	fqdn   string
	client *vault.Client
}

type vaultSaPolicyOptions struct {
	name        string
	saName      string
	saNamespace string
}
