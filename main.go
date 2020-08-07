package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	External   bool
	Namespace  string
	Kubeconfig string
	ObjectKind string
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-kubernetes-checks",
			Short:    "Sensu Kubernetes checks",
			Keyspace: "sensu.io/plugins/sensu-kubernetes-checks/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		&sensu.PluginConfigOption{
			Path:      "namespace",
			Env:       "KUBERNETES_NAMESPACE",
			Argument:  "namespace",
			Shorthand: "n",
			Default:   "",
			Usage:     "Namespace to which to limit this check",
			Value:     &plugin.Namespace,
		},
		&sensu.PluginConfigOption{
			Path:      "external",
			Env:       "",
			Argument:  "external",
			Shorthand: "e",
			Default:   false,
			Usage:     "Connect to cluster externally (using kubeconfig)",
			Value:     &plugin.External,
		},
		&sensu.PluginConfigOption{
			Path:      "kubeconfig",
			Env:       "KUBERNETES_CONFIG",
			Argument:  "kubeconfig",
			Shorthand: "c",
			Default:   "",
			Usage:     "Path to the kubeconfig file (default $HOME/.kube/config)",
			Value:     &plugin.Kubeconfig,
		},
		&sensu.PluginConfigOption{
			Path:      "object-kind",
			Env:       "KUBERNETES_OBJECT_KIND",
			Argument:  "object-kind",
			Shorthand: "k",
			Default:   "",
			Usage:     "Object kind to limit query to",
			Value:     &plugin.ObjectKind,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *types.Event) (int, error) {
	if plugin.External {
		if len(plugin.Kubeconfig) == 0 {
			if home := homeDir(); home != "" {
				plugin.Kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}
	}

	return sensu.CheckStateOK, nil
}

func executeCheck(event *types.Event) (int, error) {

	var config *rest.Config
	var err error

	opts := metav1.ListOptions{}

	if len(plugin.ObjectKind) > 0 {
		opts.FieldSelector = fmt.Sprintf("involvedObject.kind=%s", plugin.ObjectKind)
	}

	if plugin.External {
		config, err = clientcmd.BuildConfigFromFlags("", plugin.Kubeconfig)
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("Failed to get kubeconfig: %v", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("Failed to get in InClusterCOnfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Failed to get clientset: %v", err)
	}

	//events, err := clientset.CoreV1().Events(plugin.Namespace).List(context.TODO(), metav1.ListOptions{})
	events, err := clientset.CoreV1().Events(plugin.Namespace).List(context.TODO(), opts)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Failed to get events: %v", err)
	}
	fmt.Printf("There are %d events in the cluster\n", len(events.Items))
	for _, item := range events.Items {
		fmt.Printf("Namespace: %s Name: %s Count: %d Kind: %s %s-%s\n", item.ObjectMeta.Namespace, item.ObjectMeta.Name, item.Count, item.InvolvedObject.Kind, item.Reason, item.Message)
	}

	return sensu.CheckStateOK, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
