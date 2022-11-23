package bootstrap

import (
	"fmt"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

var saRoles = []vaultSaRole{
	{
		name:        "argocd-repo-server",
		saName:      "argocd-repo-server",
		saNamespace: "argocd",
	},
}

func addRole(client *vault.Client, options *vaultSaRole) error {
	path := fmt.Sprintf("auth/kubernetes/role/%s", options.name)
	data := map[string]interface{}{
		"bound_service_account_names":      options.saName,
		"bound_service_account_namespaces": options.saNamespace,
		"policies":                         []string{policyName, "default"},
		"ttl":                              "1h",
	}

	_, err = client.Logical().Write(path, data)
	if err != nil {
		return err
	}
	log.Infof("k8s auth role '%s' configured", options.name)
	return nil
}
