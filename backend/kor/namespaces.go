package main

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

func getListNamespacesErrorIfExists(clientset kubernetes.Interface, w http.ResponseWriter) error {
	_, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	return err
}
