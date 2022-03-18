package bootstrap

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func InitContainer() {
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

	podName, ok := os.LookupEnv("VAULT_K8S_POD_NAME")
	if !ok {
		panic("Cannot extract Pod name from environment variables")
	}

	pod, err := clientsetK8s.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error(err.Error())
		panic("Cannot extract Pod information from Kubernetes API")
	}

	randomString := strings.Replace(uuid.New().String(), "-", "", -1)
	jobName := podName + "-bootstrap-" + randomString[0:4]
	JobImage := pod.Status.InitContainerStatuses[0].Image

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      "Never",
					ServiceAccountName: vaultServiceAccount,
					Containers: []corev1.Container{
						{
							Name:  jobName,
							Image: JobImage,
							Env: []corev1.EnvVar{
								{
									Name:  "VAULT_ADDR",
									Value: vaultAddr,
								},
								{
									Name:  "VAULT_CLUSTER_MEMBERS",
									Value: vaultClusterMembers,
								},
								{
									Name:  "VAULT_KEY_SHARES",
									Value: strconv.Itoa(vaultKeyShares),
								},
								{
									Name:  "VAULT_KEY_THRESHOLD",
									Value: strconv.Itoa(vaultKeyThreshold),
								},
								{
									Name:  "VAULT_ENABLE_INIT",
									Value: strconv.FormatBool(vaultInit),
								},
								{
									Name:  "VAULT_ENABLE_K8SSECRET",
									Value: strconv.FormatBool(vaultK8sSecret),
								},
								{
									Name:  "VAULT_ENABLE_UNSEAL",
									Value: strconv.FormatBool(vaultUnseal),
								},
								{
									Name:  "VAULT_ENABLE_K8SAUTH",
									Value: strconv.FormatBool(vaultK8sAuth),
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := clientsetK8s.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		log.Error(err.Error())
		panic("Failed to create job")
	}
	log.Info("Created job ", result.GetObjectMeta().GetName())
}
