package bootstrap

import (
	"fmt"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

func checkInit(pod vaultPod) (bool, error) {
	init, err := pod.client.Sys().InitStatus()
	if err != nil {
		return false, err
	}
	return init, nil
}

func operatorInit(pod vaultPod) (*string, *[]string, error) {
	initReq := &vault.InitRequest{
		SecretShares:    vaultKeyShares,
		SecretThreshold: vaultKeyThreshold,
	}
	initResp, err := pod.client.Sys().Init(initReq)
	if err != nil {
		return nil, nil, err
	}

	for i := 0; i < 15; i++ {
		init, err := checkInit(pod)
		if err != nil {
			log.Errorf(err.Error())
			time.Sleep(1 * time.Second)
			continue
		}
		if !init {
			time.Sleep(1 * time.Second)
			continue
		}
		log.Infof("%s: vault successfully initialized", pod.name)
		return &initResp.RootToken, &initResp.Keys, nil
	}
	panic("Cannot proceed. Vault not initialized")
}

func operatorRaftJoin(pod vaultPod, leader vaultPod) error {
	log.Debugf("%s: raft join", pod.name)
	joinReq := &vault.RaftJoinRequest{
		LeaderAPIAddr: leader.fqdn,
	}
	joinResp, err := pod.client.Sys().RaftJoin(joinReq)
	if err != nil {
		return err
	}
	if joinResp == nil || !joinResp.Joined {
		return fmt.Errorf("nil or negative response from raft join request: %v", joinResp)
	}

	log.Infof("%s: node successfully joined raft", pod.name)
	return nil
}

// log tokens to K8s log if you don't want to save it in a secret
func logTokens(rootToken *string, unsealKeys *[]string) {
	tokenLog := fmt.Sprintf("Root Token: %s", *rootToken)
	unsealKeysLog := fmt.Sprintf("Unseal Key(s): %s", strings.Join(*unsealKeys, ";"))
	log.Info(tokenLog)
	log.Info(unsealKeysLog)
}
