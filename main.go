package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	External       bool
	Namespace      string
	Kubeconfig     string
	ObjectKind     string
	EventType      string
	Interval       uint32
	Handlers       []string
	LabelSelectors string
	StatusMap      string
	AgentAPIURL    string
}

type eventStatusMap map[string]uint32

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-kubernetes-events",
			Short:    "Sensu Kubernetes events check",
			Keyspace: "sensu.io/plugins/sensu-kubernetes-events/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		{
			Path:      "namespace",
			Env:       "KUBERNETES_NAMESPACE",
			Argument:  "namespace",
			Shorthand: "n",
			Default:   "",
			Usage:     "Namespace to which to limit this check",
			Value:     &plugin.Namespace,
		},
		{
			Path:      "external",
			Env:       "",
			Argument:  "external",
			Shorthand: "e",
			Default:   false,
			Usage:     "Connect to cluster externally (using kubeconfig)",
			Value:     &plugin.External,
		},
		{
			Path:      "kubeconfig",
			Env:       "KUBERNETES_CONFIG",
			Argument:  "kubeconfig",
			Shorthand: "c",
			Default:   "",
			Usage:     "Path to the kubeconfig file (default $HOME/.kube/config)",
			Value:     &plugin.Kubeconfig,
		},
		{
			Path:      "object-kind",
			Env:       "KUBERNETES_OBJECT_KIND",
			Argument:  "object-kind",
			Shorthand: "k",
			Default:   "",
			Usage:     "Object kind to limit query to (Pod, Cluster, etc.)",
			Value:     &plugin.ObjectKind,
		},
		{
			Path:      "event-type",
			Env:       "KUBERNETES_EVENT_TYPE",
			Argument:  "event-type",
			Shorthand: "t",
			Default:   "!=Normal",
			Usage:     "Query for fieldSelector type (supports = and !=)",
			Value:     &plugin.EventType,
		},
		{
			Path:      "label-selectors",
			Env:       "KUBERNETES_LABEL_SELECTORS",
			Argument:  "label-selectors",
			Shorthand: "l",
			Default:   "",
			Usage:     "Query for labelSelectors (e.g. release=stable,environment=qa)",
			Value:     &plugin.LabelSelectors,
		},
		{
			Path:      "status-map",
			Env:       "KUBERNETES_STATUS_MAP",
			Argument:  "status-map",
			Shorthand: "s",
			Default:   `{"normal": 0, "warning": 1, "default": 3}`,
			Usage:     "Map Kubernetes event type to Sensu event status",
			Value:     &plugin.StatusMap,
		},
		{
			Path:      "agent-api-url",
			Env:       "KUBERNETES_AGENT_API_URL",
			Argument:  "agent-api-url",
			Shorthand: "a",
			Default:   "http://127.0.0.1:3031/events",
			Usage:     "The URL for the Agent API used to send events",
			Value:     &plugin.AgentAPIURL,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, true)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	if plugin.External {
		if len(plugin.Kubeconfig) == 0 {
			if home := homeDir(); home != "" {
				plugin.Kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}
	}

	// check to make sure plugin.EventType starts with = or !=, if not, prepend =
	if len(plugin.EventType) > 0 && !strings.HasPrefix(plugin.EventType, "!=") && !strings.HasPrefix(plugin.EventType, "=") {
		plugin.EventType = fmt.Sprintf("=%s", plugin.EventType)
	}

	// Pick these up from the STDIN event
	plugin.Interval = event.Check.Interval
	plugin.Handlers = event.Check.Handlers

	if len(plugin.Namespace) == 0 {
		plugin.Namespace = event.Check.Namespace
	} else if plugin.Namespace == "all" {
		plugin.Namespace = ""
	}

	if len(plugin.AgentAPIURL) == 0 {
		return sensu.CheckStateCritical, fmt.Errorf("--agent-api-url or env var KUBERNETES_AGENT_API_URL required")
	}

	return sensu.CheckStateOK, nil
}

func executeCheck(event *corev2.Event) (int, error) {

	var config *rest.Config
	var err error

	if plugin.External {
		config, err = clientcmd.BuildConfigFromFlags("", plugin.Kubeconfig)
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("Failed to get kubeconfig: %v", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("Failed to get in InClusterConfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Failed to get clientset: %v", err)
	}

	var fieldSelectors []string

	if len(plugin.EventType) > 0 {
		// The plugin.EventType should include its operator (=/!=) that's why the
		// the string is abutted to the type string below
		fieldSelectors = append(fieldSelectors, fmt.Sprintf("type%s", plugin.EventType))
	}

	if len(plugin.ObjectKind) > 0 {
		fieldSelectors = append(fieldSelectors, fmt.Sprintf("involvedObject.kind=%s", plugin.ObjectKind))
	}

	listOptions := metav1.ListOptions{}

	if len(fieldSelectors) > 0 {
		listOptions.FieldSelector = strings.Join(fieldSelectors, ",")
	}

	if len(plugin.LabelSelectors) > 0 {
		listOptions.LabelSelector = plugin.LabelSelectors
	}

	events, err := clientset.CoreV1().Events(plugin.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Failed to get events: %v", err)
	}

	output := []string{}

	for _, item := range events.Items {
		if time.Since(item.FirstTimestamp.Time).Seconds() <= float64(plugin.Interval) {
			output = append(output, fmt.Sprintf("Event for %s %s in namespace %s, reason: %q, message: %q", item.InvolvedObject.Kind, item.ObjectMeta.Name, item.ObjectMeta.Namespace, item.Reason, item.Message))
			event, err := createSensuEvent(item)
			if err != nil {
				return sensu.CheckStateCritical, err
			}
			err = submitEventAgentAPI(event)
			if err != nil {
				return sensu.CheckStateCritical, err
			}
		}
	}

	fmt.Printf("There are %d event(s) in the cluster that match field %q and label %q\n", len(output), listOptions.FieldSelector, listOptions.LabelSelector)
	for _, out := range output {
		fmt.Println(out)
	}

	return sensu.CheckStateOK, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func createSensuEvent(k8sEvent k8scorev1.Event) (*corev2.Event, error) {
	event := &corev2.Event{}
	event.Check = &corev2.Check{}
	msgFields := strings.Fields(k8sEvent.Message)

	lowerKind := strings.ToLower(k8sEvent.InvolvedObject.Kind)
	lowerName := strings.ToLower(k8sEvent.InvolvedObject.Name)
	lowerFieldPath := strings.ToLower(k8sEvent.InvolvedObject.FieldPath)
	lowerReason := strings.ToLower(k8sEvent.Reason)
	lowerMessage := strings.ToLower(k8sEvent.Message)

	// Default labels
	event.ObjectMeta.Labels = make(map[string]string)
	event.ObjectMeta.Labels["io.kubernetes.event.id"] = k8sEvent.ObjectMeta.Name
	event.ObjectMeta.Labels["io.kubernetes.event.namespace"] = k8sEvent.ObjectMeta.Namespace

	// Sensu Event Name
	switch lowerKind {
	case "pod":
		if strings.HasPrefix(lowerFieldPath, "spec.containers") {
			// This is a Pod/Container event (i.e. an event that is associated with a
			// K8s Pod resource, with reference to a specific container in the pod).
			// Pod/Container event names need to be prefixed with container names to
			// avoid event name collisions (e.g. container-influxdb-backoff vs
			// container-grafana-backoff).
			start := strings.Index(lowerFieldPath, "{") + 1
			end := strings.Index(lowerFieldPath, "}")
			container := lowerFieldPath[start:end]
			if len(msgFields) == 2 && msgFields[0] == "Error:" {
				// Expected outcome: container-<container_name>-<error>
				//
				// Examples:
				// - container-nginx-imagepullbackoff
				event.Check.ObjectMeta.Name = fmt.Sprintf(
					"container-%s-%s",
					strings.ToLower(container),
					strings.ToLower(msgFields[1]),
				)
			} else {
				// Expected outcome: container-<container_name>-<reason>
				//
				// Examples:
				// - container-nginx-started
				event.Check.ObjectMeta.Name = fmt.Sprintf(
					"container-%s-%s",
					strings.ToLower(container),
					lowerReason,
				)
			}
		} else {
			// This is a Pod event.
			//
			// Expected outcome: pod-<reason>
			//
			// Examples:
			// - pod-scheduled
			// - pod-created
			// - pod-deleted
			event.Check.ObjectMeta.Name = fmt.Sprintf(
				"pod-%s",
				lowerReason,
			)
		}
	case "replicaset":
		// Parse replicaset event.message values like "Created pod:
		// nginx-bbd465f66-rwb2d" by splitting the string on "pod:".
		if len(strings.Split(lowerMessage, "pod:")) == 2 {
			// This is a Replicaset/Pod event (i.e. an event that is associated
			// with a K8s Replicaset resource, with reference to a specific Pod
			// that is managed by the Replicaset). Replicaset/Pod event names need
			// are prefixed with "pod" for verbosity. NOTE: Replicaset/Pod events
			// are also associated with the underlying Pod entity; see
			// "switch lowerKind" (below) for more information.
			//
			// Expected outcome: pod-<reason>
			//
			// Examples:
			// - pod-scheduled
			// - pod-created
			// - pod-deleted
			//
			// Many replicaset events have messages like "Created pod:
			// nginx-bbd465f66-rwb2d". We want to capture the first word
			// in this string as the event "verb".
			verb := strings.ToLower(msgFields[0])
			event.Check.ObjectMeta.Name = fmt.Sprintf(
				"pod-%s",
				verb,
			)
		} else {
			// This is a Replicaset event.
			//
			// Expected outcome: <reason>
			//
			// Examples:
			// - replicaset-deleted
			event.Check.ObjectMeta.Name = strings.ToLower(
				fmt.Sprintf(
					"replicaset-%s",
					k8sEvent.Reason,
				),
			)
		}
	case "deployment":
		if len(strings.Split(lowerMessage, "replica set")) == 2 {
			message := strings.Split(lowerMessage, "replica set")
			replicaset := strings.Split(strings.TrimSpace(message[1]), " ")[0]
			event.Check.ObjectMeta.Name = fmt.Sprintf(
				"%s-%s",
				lowerReason,
				strings.ToLower(replicaset),
			)
		} else {
			event.Check.ObjectMeta.Name = fmt.Sprintf(
				"%s-%s",
				lowerReason,
				lowerName,
			)
		}
	case "endpoints":
		event.Check.ObjectMeta.Name = fmt.Sprintf(
			"endpoint-%s-%s",
			lowerName,
			lowerReason,
		)
	case "node":
		if strings.HasPrefix(lowerReason, "deleting node") {
			// Node deletion events "reason" field values are completely inconsistent
			// with most other Node events
			event.Check.ObjectMeta.Name = "deletingnode"
		} else {
			// Most node events have pretty clean "reason" field values
			event.Check.ObjectMeta.Name = lowerReason
		}
	default:
		if len(msgFields) == 2 && msgFields[0] == "Error:" {
			// If we have a definitive single word error message, use that as the check name
			event.Check.ObjectMeta.Name = msgFields[1]
		} else {
			event.Check.ObjectMeta.Name = k8sEvent.ObjectMeta.Name
		}
	}

	// Sensu Entity
	switch lowerKind {
	case "replicaset":
		message := strings.Split(k8sEvent.Message, "pod:")
		if len(message) == 2 {
			// associate this event with the pod
			pod := strings.Split(strings.TrimSpace(message[1]), " ")[0]
			event.Check.ProxyEntityName = strings.ToLower(pod)
		} else {
			// associate this event with the replicaset
			event.Check.ProxyEntityName = lowerName
		}
	case "pod", "deployment", "endpoints", "node":
		event.Check.ProxyEntityName = lowerName
	default:
		event.Check.ProxyEntityName = fmt.Sprintf("%s-%s", lowerName, lowerKind)
	}
	status, err := getSensuEventStatus(k8sEvent.Type)
	if err != nil {
		return &corev2.Event{}, err
	}
	event.Timestamp = k8sEvent.LastTimestamp.Time.Unix()
	event.Check.Status = status
	event.Check.Interval = plugin.Interval
	event.Check.Handlers = plugin.Handlers
	event.Check.Output = fmt.Sprintf("Event for %s %s in namespace %s, reason: %q, message: %q\n", k8sEvent.InvolvedObject.Kind, k8sEvent.ObjectMeta.Name, k8sEvent.ObjectMeta.Namespace, k8sEvent.Reason, k8sEvent.Message)
	return event, nil
}

func submitEventAgentAPI(event *corev2.Event) error {

	encoded, _ := json.Marshal(event)
	resp, err := http.Post(plugin.AgentAPIURL, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return fmt.Errorf("Failed to post event to %s failed: %v", plugin.AgentAPIURL, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("POST of event to %s failed with %v", plugin.AgentAPIURL, resp.Status)
	}

	return nil
}

func getSensuEventStatus(eventType string) (uint32, error) {
	statusMap := eventStatusMap{}
	err := json.Unmarshal([]byte(strings.ToLower(plugin.StatusMap)), &statusMap)
	if err != nil {
		return 255, err
	}
	// attempt to map it to a specified status, if not see if a
	// default status exists, otherwise return 255
	if val, ok := statusMap[strings.ToLower(eventType)]; ok {
		return val, nil
	} else if val, ok = statusMap["default"]; ok {
		return val, nil
	}
	return 255, nil
}
