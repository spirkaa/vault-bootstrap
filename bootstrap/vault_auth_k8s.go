package bootstrap

import (
	"fmt"
	"os"
	"time"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

func checkVaultUp(client *vault.Client) bool {
	for i := 0; i < 15; i++ {
		hr, err := client.Sys().Health()
		if err != nil {
			log.Warn(err.Error(), "k8s auth: Retrying...")
			time.Sleep(1 * time.Second)
			continue
		}
		if !hr.Initialized || hr.Sealed {
			log.Warn("k8s auth: Vault not Initialized/Unsealed. Retrying...")
			time.Sleep(1 * time.Second)
			continue
		}
		return true
	}
	return false
}

func checkK8sAuth(client *vault.Client) (bool, error) {
	auths, err := client.Logical().Read("sys/auth")
	if err != nil {
		return false, err
	}
	if k8sAuth := auths.Data["kubernetes/"]; k8sAuth != nil {
		log.Info("k8s auth already enabled")
		return true, nil
	}
	return false, nil
}

func configureK8sAuth(client *vault.Client, clientsetK8s *kubernetes.Clientset) error {
	err := client.Sys().EnableAuthWithOptions("kubernetes/", &vault.EnableAuthOptions{
		Type: "kubernetes",
	})

	if err != nil {
		return err
	}

	// Get k8s API URL
	const (
		EnvK8sSvc  = "KUBERNETES_SERVICE_HOST"
		EnvK8sPort = "KUBERNETES_SERVICE_PORT"
	)
	k8sSvc, ok := os.LookupEnv(EnvK8sSvc)
	if !ok {
		return fmt.Errorf("k8s auth: lookup of %s failed", EnvK8sSvc)
	}
	k8sPort, ok := os.LookupEnv(EnvK8sPort)
	if !ok {
		return fmt.Errorf("k8s auth: lookup of %s failed", EnvK8sPort)
	}
	k8sHost := fmt.Sprintf("https://%s:%s", k8sSvc, k8sPort)

	// Configure k8s authentication
	_, err = client.Logical().Write("auth/kubernetes/config", map[string]interface{}{
		"kubernetes_host": k8sHost,
	})
	if err != nil {
		return err
	}
	log.Info("k8s auth: Successfully enabled")
	return nil
}
