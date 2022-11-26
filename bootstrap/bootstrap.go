package bootstrap

import (
	"context"
	"net/url"
	"os"
	"strings"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func getPodName(p *apiv1.Pod) string {
	if p.ObjectMeta.Name != "" {
		return p.ObjectMeta.Name
	}
	return strings.TrimSuffix(p.ObjectMeta.GenerateName, "-")
}

// Run Vault bootstrap
func Run() {
	// Create clientSet for k8s client-go
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	clientsetK8s, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	podsList, err := clientsetK8s.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	var pdList []string
	for _, pd := range podsList.Items {
		pdList = append(pdList, getPodName(&pd))
	}
	log.Debugf("Pods list: %s", strings.Join(pdList, ";"))

	// Define Vault client for Vault LB
	clientConfigLB := vault.DefaultConfig()
	// Skip TLS verification for initialization
	insecureTLS := &vault.TLSConfig{
		Insecure: true,
	}
	clientConfigLB.ConfigureTLS(insecureTLS)

	clientLB, err := vault.NewClient(clientConfigLB)
	if err != nil {
		os.Exit(1)
	}

	// Slice of maps containing vault pods details
	var vaultPods []vaultPod
	vaultMembersUrls := strings.Split(vaultClusterMembers, ",")
	// Generate the slice from Env variable
	for _, member := range vaultMembersUrls {
		var pod vaultPod
		podFqdn, _ := url.Parse(member)
		pod.fqdn = member
		pod.name = strings.Split(podFqdn.Hostname(), ".")[0]
		clientConfig := &vault.Config{
			Address: pod.fqdn,
		}
		clientConfig.ConfigureTLS(insecureTLS)

		client, err := vault.NewClient(clientConfig)
		if err != nil {
			os.Exit(1)
		}
		pod.client = client
		vaultPods = append(vaultPods, pod)
	}
	// Define main client (vault-0) which will be used for initialization
	// When using integrated RAFT storage, the vault cluster member that is initialized
	// needs to be first one which is unsealed
	// In the unseal part we'll always start with the first member
	vaultFirstPod := vaultPods[0]
	preflight(vaultPods)

	var rootToken *string
	var unsealKeys *[]string

	pVaultSecretRoot := &vaultSecretRoot
	pVaultSecretUnseal := &vaultSecretUnseal

	if vaultInit {
		init, err := checkInit(vaultFirstPod)
		if err != nil {
			log.Errorf(err.Error())
			os.Exit(1)
		}
		if !init {
			rootToken, unsealKeys, err = operatorInit(vaultFirstPod)
			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
			// If flag for creating k8s secrets is set
			if vaultK8sSecret {
				// Check if vault secret root exists
				_, err = getValuesFromK8sSecret(clientsetK8s, pVaultSecretRoot)
				if err != nil {
					// if it fails because secret is not found, create the secret
					if errors.IsNotFound(err) {
						if errI := createK8sSecret(clientsetK8s, &vaultSecretRoot, rootToken); errI != nil {
							log.Error(errI.Error())
							os.Exit(1)
						}
					} else {
						log.Error(err.Error())
						os.Exit(1)
					}
				}
				// Check if vault secret unseal exists
				_, err = getValuesFromK8sSecret(clientsetK8s, pVaultSecretUnseal)
				if err != nil {
					// if it fails because secret is not found, create the secret
					if errors.IsNotFound(err) {
						unsealKeysString := strings.Join(*unsealKeys, ";")
						if errI := createK8sSecret(clientsetK8s, &vaultSecretUnseal, &unsealKeysString); errI != nil {
							log.Error(errI.Error())
							os.Exit(1)
						}
					} else {
						log.Error(err.Error())
						os.Exit(1)
					}
				}
			} else {
				logTokens(rootToken, unsealKeys)
			}
		} else {
			log.Info("Vault already initialized")
		}
	}

	if vaultUnseal {
		// Check if unseal keys in memory and if not load them
		if unsealKeys == nil {
			unsealKeysString, err := getValuesFromK8sSecret(clientsetK8s, pVaultSecretUnseal)
			if err != nil {
				panic("Cannot load Unseal Keys")
			}
			npUnsealKeys := strings.Split(*unsealKeysString, ";")
			unsealKeys = &npUnsealKeys
			log.Debug("Unseal Keys loaded successfully")
		}
		// Unseal first member first
		unsealMember(vaultFirstPod, *unsealKeys)
		for _, vaultPod := range vaultPods[1:] {
			if err := operatorRaftJoin(vaultPod, vaultFirstPod); err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
			unsealMember(vaultPod, *unsealKeys)
		}
	}

	if vaultK8sAuth {
		// Check if root token in memory and if not load it
		if rootToken == nil {
			pRootToken, err := getValuesFromK8sSecret(clientsetK8s, pVaultSecretRoot)
			if err != nil {
				panic("Cannot load Root Token")
			}
			rootTokenVal := *pRootToken
			rootToken = &rootTokenVal
			log.Debug("Root Token loaded successfully")
		}

		up := checkVaultUp(clientLB)
		if !up {
			panic("k8s auth: Vault not ready. Cannot proceed")
		}

		// enable k8s auth
		clientLB.SetToken(*rootToken)
		k8sAuth, err := checkK8sAuth(clientLB)
		if err != nil {
			log.Errorf(err.Error())
			os.Exit(1)
		}
		if !k8sAuth {
			if err := configureK8sAuth(clientLB, clientsetK8s); err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
		}

		// add policy
		if err := addPolicy(clientLB); err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}

		// add roles
		// roleFromVars := vaultSaRole{
		// 	vaultK8sAuthServiceAccount,
		// 	vaultK8sAuthServiceAccount,
		// 	namespace,
		// }
		// saRoles = append(saRoles, roleFromVars)
		// log.Infof("%s", saRoles)
		for _, role := range saRoles {
			if err := addRole(clientLB, &role); err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
		}

		// enable secret engine
		secret, err := checkSecretEngine(clientLB)
		if err != nil {
			log.Errorf(err.Error())
			os.Exit(1)
		}
		if !secret {
			if err := enableSecretEngine(clientLB); err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
		}

		return
	}
}
