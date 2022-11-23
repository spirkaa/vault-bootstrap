package bootstrap

import (
	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

const policyName = "read-all"
const policyDef = `
path "secret/data/*" {
	capabilities = ["read", "list"]
}
`

func addPolicy(client *vault.Client) error {
	err := client.Sys().PutPolicy(policyName, policyDef)
	if err != nil {
		return err
	}
	log.Infof("k8s auth policy '%s' configured", policyName)
	return nil
}
