package kvs

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	logging "rpc-gateway/pkg/core/log"

	"github.com/ghodss/yaml"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type K8sClient struct {
	Clientset *kubernetes.Clientset
	Namespace string
}

//decode Service
func (c *K8sClient) UnmarshalService(bytes []byte) corev1.Service {
	var spec corev1.Service
	err := yaml.Unmarshal(bytes, &spec)
	if err != nil {
		panic(err.Error())
	}
	return spec
}

//deploy service
func (c *K8sClient) DeployService(svc corev1.Service) error {
	_, err := c.Clientset.CoreV1().Services(c.Namespace).Create(context.TODO(), &svc, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			existSvc := c.GetServices(svc.Name)
			resourceVersion := existSvc[0].ObjectMeta.ResourceVersion
			clusterIP := existSvc[0].Spec.ClusterIP
			svc.ObjectMeta.ResourceVersion = resourceVersion
			svc.Spec.ClusterIP = clusterIP
			_, errUpdate := c.Clientset.CoreV1().Services(c.Namespace).Update(context.TODO(), &svc, metav1.UpdateOptions{})
			if errUpdate != nil {
				logging.Log.Error("failed services updated error!", errUpdate)
			} else {
				logging.Log.Info("success services        ", "\""+svc.Name+"\"")
				return nil
			}
		}
		return err
	} else {
		logging.Log.Info("success services        ", "\""+svc.Name+"\"")
		return nil
	}
}

//get services
func (c *K8sClient) GetServices(apps ...string) []*corev1.Service {
	var svcs []*corev1.Service
	if len(apps) > 0 {
		for _, app := range apps {
			svc, _ := c.Clientset.CoreV1().Services(c.Namespace).Get(context.TODO(), app, metav1.GetOptions{})
			if svc.Name == "" {
				logging.Log.Info("service not exists!")
			} else {
				svcs = append(svcs, svc)
			}
		}
	} else {
		svcList, _ := c.Clientset.CoreV1().Services(c.Namespace).List(context.TODO(), metav1.ListOptions{})
		logging.Log.Infof("there are %d svc in the cluster", len(svcList.Items))
		for k, _ := range svcList.Items {
			svcs = append(svcs, &svcList.Items[k])
		}
	}
	return svcs
}

//get services by label
func (c *K8sClient) GetServicesByLabel(data map[string]string) *corev1.ServiceList {
	labelSelector := metav1.LabelSelector{MatchLabels: data}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		Limit:         100,
	}
	svcList, _ := c.Clientset.CoreV1().Services(c.Namespace).List(context.TODO(), listOptions)
	// logging.Log.Infof("there are %d svc in the cluster", len(svcList.Items))
	return svcList
}

//decode Deployment
func (c *K8sClient) UnmarshalDeployment(bytes []byte) appsv1.Deployment {
	var spec appsv1.Deployment
	err := yaml.Unmarshal(bytes, &spec)
	if err != nil {
		panic(err.Error())
	}
	return spec
}

//deploy deployment
func (c *K8sClient) DeployDeployment(deploy appsv1.Deployment) error {
	_, err := c.Clientset.AppsV1().Deployments(c.Namespace).Create(context.TODO(), &deploy, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			existDeploys := c.GetDeployments(deploy.Name)
			if !reflect.DeepEqual(existDeploys[0], deploy) {
				deploymentUpdate, errUpdate := c.Clientset.AppsV1().Deployments(c.Namespace).Update(context.TODO(), &deploy, metav1.UpdateOptions{})
				if errUpdate != nil {
					logging.Log.Info("failed deployments updated error!")
					return errUpdate
				} else {
					if deploymentUpdate.Status.Replicas > 0 {
						logging.Log.Info("success deployments     ", "\""+deploy.Name+"\"")
						// log.Println("~~~:replicas:", deploymentUpdate.Status.Replicas)
						// for _, st := range deploymentUpdate.Status.Conditions {
						// 	log.Println("~~~:conditions:", "->", st.Type, ":", st.Status)
						// }
						return nil
					}
				}
			} else {
				logging.Log.Info("no need update")
				return errors.New("no need update")
			}
		}
		return err
	} else {
		logging.Log.Info("success deployments     ", "\""+deploy.Name+"\"")
		return nil
	}
}

//get deploys
func (c *K8sClient) GetDeployments(apps ...string) []*appsv1.Deployment {
	var deploys []*appsv1.Deployment
	if len(apps) > 0 {
		for _, app := range apps {
			deploy, _ := c.Clientset.AppsV1().Deployments(c.Namespace).Get(context.TODO(), app, metav1.GetOptions{})
			if deploy.Status.Replicas == 0 {
				logging.Log.Info("resource not found!")
			} else {
				deploys = append(deploys, deploy)
			}
		}
	} else {
		deployList, _ := c.Clientset.AppsV1().Deployments(c.Namespace).List(context.TODO(), metav1.ListOptions{})
		logging.Log.Infof("there are %d deployment in the cluster", len(deployList.Items))
		for _, deploy := range deployList.Items {
			deploys = append(deploys, &deploy)
		}
	}
	return deploys
}

//decode ConfigMap
func (c *K8sClient) UnmarshalConfigMap(bytes []byte) corev1.ConfigMap {
	var spec corev1.ConfigMap
	err := yaml.Unmarshal(bytes, &spec)
	if err != nil {
		panic(err.Error())
	}
	return spec
}

//deploy Configmap
func (c *K8sClient) DeployConfigMap(cm corev1.ConfigMap) error {
	_, err := c.Clientset.CoreV1().ConfigMaps(c.Namespace).Create(context.TODO(), &cm, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			_, errUpdate := c.Clientset.CoreV1().ConfigMaps(c.Namespace).Update(context.TODO(), &cm, metav1.UpdateOptions{})
			if errUpdate != nil {
				logging.Log.Error("failed configmaps updated error!")
				return errUpdate
			} else {
				logging.Log.Info("success configmaps      ", "\""+cm.Name+"\"")
				return nil
			}
		}
		return err
	} else {
		logging.Log.Info("success configmaps      ", "\""+cm.Name+"\"")
		return nil
	}
}

//get Configmaps
func (c *K8sClient) GetConfigMaps(cms ...string) []*corev1.ConfigMap {
	var configmaps []*corev1.ConfigMap
	if len(cms) > 0 {
		for _, cm := range cms {
			configmap, _ := c.Clientset.CoreV1().ConfigMaps(c.Namespace).Get(context.TODO(), cm, metav1.GetOptions{})
			if configmap.Name == "" {
				logging.Log.Info("configmap not exists!")
			} else {
				configmaps = append(configmaps, configmap)
			}
		}
	} else {
		cms, _ := c.Clientset.CoreV1().ConfigMaps(c.Namespace).List(context.TODO(), metav1.ListOptions{})
		logging.Log.Info("there are %d cm in the cluster", len(cms.Items))
		for _, cm := range cms.Items {
			configmaps = append(configmaps, &cm)
		}
	}
	return configmaps
}

//get pods by label
func (c *K8sClient) GetPodsByLabel(data map[string]string) *corev1.PodList {
	labelSelector := metav1.LabelSelector{MatchLabels: data}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		Limit:         100,
	}
	pods, _ := c.Clientset.CoreV1().Pods(c.Namespace).List(context.TODO(), listOptions)
	return pods
}

//pod exec command
func (c *K8sClient) PodExecCommand(namespace, podName, command, containerName string) (string, string, error) {
	kubecfgpath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	//kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubecfgpath)
	if err != nil {
		kubecfgpath = filepath.Join("./run/", "kubeconfig")
		config, err = clientcmd.BuildConfigFromFlags("", kubecfgpath)
		if err != nil {
			panic(err)
		}
	}

	k8sCli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", "", err
	}

	//command
	cmd := []string{
		"sh",
		"-c",
		command,
	}
	const tty = false
	req := k8sCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).SubResource("exec").Param("container", containerName)
	req.VersionedParams(
		&corev1.PodExecOptions{
			Command: cmd,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     tty,
		},
		scheme.ParameterCodec,
	)

	var stdout, stderr bytes.Buffer
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", "", err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

//resource delete
func (c *K8sClient) ResDelete(resourceType string, resName string) error {
	switch resourceType {
	case "Service":
		name := resName
		err := c.Clientset.CoreV1().Services(c.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			logging.Log.Error("delete service error!", err)
			return err
		}
		logging.Log.Info("delete success services      ", "\""+name+"\"")

	case "Deployment":
		name := resName
		err := c.Clientset.AppsV1().Deployments(c.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			logging.Log.Error("delete deployment error!", err)
			return err
		}
		logging.Log.Info("delete success deployment    ", "\""+name+"\"")

	case "ConfigMap":
		name := resName
		err := c.Clientset.CoreV1().ConfigMaps(c.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			logging.Log.Error("delete configmap error!", err)
			return err
		}
		logging.Log.Info("delete success configmap     ", "\""+name+"\"")

	case "StatefulSet":
		break
	}
	return nil
}

// get endpoints by name
func (c *K8sClient) GetEndpointsByName(app string) (*corev1.Endpoints, error) {
	getOptions := metav1.GetOptions{}
	endPoints, err := c.Clientset.CoreV1().Endpoints(c.Namespace).Get(context.TODO(), app, getOptions)
	if err != nil {
		return nil, err
	}
	return endPoints, nil
}
