# Vault Bootstrap

Alternative: [kubevault/unsealer](https://github.com/kubevault/unsealer)

## About

This tool can be used to initialize
[Hashicorp Vault](https://github.com/hashicorp/vault)
on [Kubernetes](https://kubernetes.io/releases/).
It can perform the following steps:

* Vault initilazation
* Save Root Token and Unseal Keys to K8s Secret
* Vault unseal
* Enable Kubernetes authentication

## Disclaimer

In this version, the Vault token and unseal Keys can only be saved to a Kubernetes secret.
This is insecure and this deployment is *ONLY SUITED FOR DEVELOPMENT ENVIRONMENTS*.
However, this tool can be extended to save Vault token and unseal Keys to a different secret engine
(Azure Key Vault, AWS KMS, another Vault instance).

## Usage

Docker Image:
[ghcr.io/spirkaa/vault-bootstrap](https://github.com/spirkaa/vault-bootstrap/pkgs/container/vault-bootstrap)*

### Scenario 1 - Job

To install Vault Bootstrap to OpenShift or Kubernetes, deploy the following Job:

```yaml
cat <<EOF | oc apply -f
apiVersion: batch/v1
kind: Job
metadata:
  name: vault
  namespace: vault
spec:
  backoffLimit: 3
  template:
    spec:
      serviceAccount: vault
      restartPolicy: Never
      containers:
      - name: vault-init
        image: ghcr.io/spirkaa/vault-bootstrap:latest
        env:
        - name: VAULT_ADDR
          value: "https://vault.vault:8200"
        - name: VAULT_CLUSTER_MEMBERS
          value: >-
            https://vault-0.vault-internal:8200,https://vault-1.vault-internal:8200,https://vault-2.vault-internal:8200
        - name: VAULT_KEY_SHARES
          value: "5"
        - name: VAULT_KEY_THRESHOLD
          value: "3"
        - name: VAULT_ENABLE_INIT
          value: "true"
        - name: VAULT_ENABLE_K8SSECRET
          value: "true"
        - name: VAULT_ENABLE_UNSEAL
          value: "true"
        - name: VAULT_ENABLE_K8SAUTH
          value: "true"
        - name: VAULT_K8SAUTH_SERVICE_ACCOUNT
          value: vault-external-secrets
        - name: NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        imagePullPolicy: Always
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: vault
  name: vault
rules:
- apiGroups:
  - ""
  resources:
  - "pods"
  verbs:
  - "get"
  - "list"
  - "watch"
- apiGroups:
  - ""
  resources:
  - "secrets"
  verbs:
  - "get"
  - "list"
  - "watch"
  - "create"
- apiGroups:
  - ""
  resources:
  - "serviceaccounts"
  verbs:
  - "get"
  - "list"
  - "watch"
- apiGroups:
  - "batch"
  resources:
  - "jobs"
  verbs:
  - "get"
  - "list"
  - "watch"
  - "create"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: vault
  namespace: vault
subjects:
- kind: ServiceAccount
  name: vault
  namespace: vault
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: vault
EOF
```

If you are choosing to save the root token and unseal keys into a Kubernetes secret,
you can re-run the init job for unsealing any of the pods (in case in gets rescheduled).
To do this, run the following command

```shell
oc get job vault-init -o json | jq 'del(.spec.selector)' | jq 'del(.spec.template.metadata.labels)' | oc replace --force -f -
```

### Scenario 2 - Init Container

This tool can be run in `init-container` mode, which can be used if we want to perform auto-unsealing from K8s secret.
In this mode, the initContainer will spawn up a `vault-bootstrap` job
configured to perform only unsealing only for the pods attached to.
To perform this scenario, add the following definition to the Vault StatefulSet definition

```yaml
initContainers:
- name: vault-bootstrap
  image: ghcr.io/spirkaa/vault-bootstrap:latest
  command:
    - /vault-bootstrap
  args:
    - --mode
    - init-container
  env:
    - name: VAULT_ADDR
      value: "http://vault.vault:8200"
    - name: VAULT_CLUSTER_MEMBERS
      value: >-
        http://vault-0.vault-internal.vault:8200
    - name: VAULT_KEY_SHARES
      value: "5"
    - name: VAULT_KEY_THRESHOLD
      value: "3"
    - name: VAULT_ENABLE_INIT
      value: "true"
    - name: VAULT_ENABLE_K8SSECRET
      value: "true"
    - name: VAULT_ENABLE_UNSEAL
      value: "true"
    - name: VAULT_ENABLE_K8SAUTH
      value: "true"
    - name: VAULT_K8SAUTH_SERVICE_ACCOUNT
      value: vault-external-secrets
    - name: VAULT_K8S_POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
```

## Configuration

The configurations are specified as Environment variables. Below the supported ones.

| Environment Variable    | Default value      | Info          |
|-------------------------|--------------------|---------------|
| VAULT_ADDR                    | https://vault:8200 | Vault address |
| VAULT_CLUSTER_MEMBERS         | https://vault:8200 | Vault cluster members as URLs specified in a comma separated list |
| VAULT_KEY_SHARES              | 1                  | Key Shares generated by initialization |
| VAULT_KEY_THRESHOLD           | 1                  | Key Threshold generated by initialization |
| VAULT_ENABLE_INIT             | true               | Enable Vault initialization |
| VAULT_ENABLE_K8SSSECRET       | true               | Enable saving Vault root token and share keys into a K8s secrets |
| VAULT_ENABLE_UNSEAL           | true               | Enable Vault unseal |
| VAULT_ENABLE_K8SAUTH          | true               | Enable Kubernetes authentication for Vault |
| VAULT_SERVICE_ACCOUNT         | vault              | Service account for job pod |
| VAULT_K8SAUTH_SERVICE_ACCOUNT | vault              | Service account for K8s authentication |
| VAULT_K8S_POD_NAME            | N/A                | Relevant only for `init-container` mode. |
