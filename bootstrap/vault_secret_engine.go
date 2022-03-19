package bootstrap

import (
	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

func checkSecretEngine(client *vault.Client) (bool, error) {
	mounts, err := client.Logical().Read("sys/mounts")
	if err != nil {
		return false, err
	}
	if secret := mounts.Data["secret/"]; secret != nil {
		log.Info("secret engine already enabled")
		return true, nil
	}
	return false, nil
}

func enableSecretEngine(client *vault.Client) error {
	err := client.Sys().Mount("secret/", &vault.MountInput{
		Type: "kv-v2",
	})
	if err != nil {
		return err
	}
	log.Info("secret engine successfully enabled")
	return nil
}
