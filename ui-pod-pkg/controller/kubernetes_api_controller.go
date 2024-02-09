package controller

import (
	"fmt"
	"hub/pkg/model"
	"hub/pkg/utils"
	"io"
	"log"
	"net/http"
	"time"

	//kubernates imports
	"context"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	// "k8s.io/client-go/1.5/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	//get IPs import
)

func CreateMultiplePods() error {
	imageName := "sahilbrane/test6:latest" // hardcode this for testing in local, original image:sahilbrane/cypress-server:latest

	nodePort := 30001
	containerPort := 8081
	for i := 0; i < 2; i++ {
		podName := utils.GetRandomNames()
		containerName := podName
		err := CreateCypressPods(imageName, podName, containerName, nodePort+i, containerPort)
		if err != nil {
			log.Printf("error creating pod: [%v]", podName)
			return err
		}
	}
	return nil
}

func CreateCypressPods(imageName, podName, containerName string, nodePort, containerPort int) error {
	// Load the Kubernetes configuration from a file (e.g., ~/.kube/config) or the default location.
	homeDir, _ := os.UserHomeDir()
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homeDir, ".kube", "config"))
	if err != nil {
		return err
	}

	// Create a Kubernetes clientset using the configuration.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// Define a Namespace for the deployment and service.
	namespace := "pll-env"

	// Create a Deployment.
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": podName},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": podName}},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            containerName,
							Image:           imageName,
							ImagePullPolicy: v1.PullIfNotPresent,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: int32(containerPort),
								},
							},
							Env: []v1.EnvVar{
								{Name: "MY_NAME", Value: podName},
							},
						},
					},
				},
			},
		},
	}

	_, err = clientset.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Create a Service with NodePort.
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName + "-service",
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"app": podName},
			Ports: []v1.ServicePort{
				{
					Port:       int32(containerPort),
					TargetPort: intstr.FromInt(containerPort),
					NodePort:   int32(nodePort),
				},
			},
			Type: v1.ServiceTypeNodePort,
		},
	}

	_, err = clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func int32Ptr(i int32) *int32 { return &i }

// To do - get pods
func GetCypressPodsIP() (model.CypressPodInfoList, error) {

	namespace := "pll-env"

	// Load the Kubernetes configuration from a file (e.g., ~/.kube/config) or the default location.
	homeDir, _ := os.UserHomeDir()
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homeDir, ".kube", "config"))
	if err != nil {
		return model.CypressPodInfoList{}, err
	}

	// Create a Kubernetes clientset using the configuration.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return model.CypressPodInfoList{}, err
	}

	// Create a list to store the pod information.
	var podInfoList model.CypressPodInfoList

	// TODO: sleeping rn, have to change it later
	time.Sleep(60 * time.Second)

	// List all running pods in the specified namespace.
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return model.CypressPodInfoList{}, err
	}

	// waits until all the pods are in running state
	// for {
	// 	allPodsRunning := true
	// 	for _, pod := range pods.Items {
	// 		if pod.Status.Phase != v1.PodRunning {
	// 			allPodsRunning = false
	// 			break
	// 		}
	// 	}

	// 	if allPodsRunning {
	// 		break
	// 	}

	// 	// Wait for a short duration before checking again
	// 	time.Sleep(time.Second * 3)
	// }

	if len(pods.Items) == 0 {
		fmt.Printf("0 Pods found\n")
		return model.CypressPodInfoList{}, err
	}

	// Get the Node IP from the first pod (assuming all pods in the namespace are on the same node).
	podInfoList.NodeIP = pods.Items[0].Status.HostIP

	services, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return model.CypressPodInfoList{}, err
	}

	// Create a slice to store the service information.
	var podServices []model.CypressPodsInfo

	// Iterate through the services and add NodePort services to the slice.
	for _, service := range services.Items {
		if service.Spec.Type == v1.ServiceTypeNodePort {
			changedName := service.Name[:len(service.Name)-8]
			log.Printf("serviceName: %v", service.Name)

			serviceInfo := model.CypressPodsInfo{
				PodName:  changedName,
				NodePort: int(service.Spec.Ports[0].NodePort),
			}
			podServices = append(podServices, serviceInfo)
		}
	}

	// Add the podServices slice to the CypressPodInfoList.
	podInfoList.PodsInfo = podServices

	log.Printf("Node IP: %s\n", podInfoList.NodeIP) //working
	for _, podInfo := range podInfoList.PodsInfo {
		log.Printf("Pod: %s NodePort: %d\n", podInfo.PodName, podInfo.NodePort)
	}

	return podInfoList, nil
}

func DeleteRedundantPods() error {
	resp, err := http.Get(utils.KUBERNETES_API_IP + utils.DELETE_CYPRESS_POD_ROUTE)
	if err != nil {
		log.Println("Error deleting pods:", err)
		return err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.Status == "200 OK" {
		log.Println("Pod deletion successfull")

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
			return err
		}
		log.Println("pod creation successful with body: ", string(body))
		return nil
	}
	log.Println("error deleting pods:", err)
	return err
}
