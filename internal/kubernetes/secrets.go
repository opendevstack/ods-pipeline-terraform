package kubernetes

import (
	"context"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

func GetSecret(clientset k8s.Interface, namespace string, secretName string) (*corev1.Secret, error) {

	log.Printf("Get secret %s in namespace %s", secretName, namespace)

	secret, err := clientset.CoreV1().
		Secrets(namespace).
		Get(context.TODO(), secretName, metav1.GetOptions{})

	return secret, err
}

func GetSecrets(clientset k8s.Interface, namespace string, secretName string) (map[string]string, error) {
	secret, err := GetSecret(clientset, namespace, secretName)
	if err != nil {
		return nil, err
	}
	if secret.Type != "Opaque" {
		return nil, fmt.Errorf("secret type is not Opaque")
	}
	secrets := make(map[string]string)
	for key, value := range secret.Data {
		secrets[key] = string(value)
	}
	return secrets, nil
}

func GetSecretKey(clientset k8s.Interface, namespace, secretName, key string) (string, error) {

	log.Printf("Get secret %s", secretName)

	secret, err := clientset.CoreV1().
		Secrets(namespace).
		Get(context.TODO(), secretName, metav1.GetOptions{})

	if err != nil {
		return "", err
	}

	v, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found", key)
	}

	return string(v), err
}
