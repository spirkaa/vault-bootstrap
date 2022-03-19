package bootstrap

import (
	"fmt"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

const policy = `
path "secret/data/*" {
	capabilities = ["read", "list"]
}
`

func configurePolicy(client *vault.Client, options *vaultSaPolicyOptions) error {
	err := client.Sys().PutPolicy(options.name, policy)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("auth/kubernetes/role/%s", options.name)
	data := map[string]interface{}{
		"bound_service_account_names":      options.saName,
		"bound_service_account_namespaces": options.saNamespace,
		"policies":                         []string{options.name, "default"},
		"ttl":                              "24h",
	}

	_, err = client.Logical().Write(path, data)
	if err != nil {
		return err
	}
	log.Info("k8s auth policy and role configured")
	return nil
}
