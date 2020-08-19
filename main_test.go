package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMain(t *testing.T) {
}

func TestCheckArgs(t *testing.T) {
	assert := assert.New(t)
	plugin.External = true
	plugin.EventType = "Normal"
	plugin.AgentAPIURL = "http://127.0.0.1:3031/events"
	event := corev2.FixtureEvent("entity1", "check1")
	status, err := checkArgs(event)
	assert.NoError(err)
	assert.Equal(sensu.CheckStateOK, status)
	assert.Equal("=Normal", plugin.EventType)
	assert.Equal("default", plugin.Namespace)
	plugin.Namespace = "all"
	status, err = checkArgs(event)
	assert.NoError(err)
	assert.Equal(sensu.CheckStateOK, status)
	assert.Equal(0, len(plugin.Namespace))
}

func TestCreateSensuEvent(t *testing.T) {
	const (
		k8sObjName    = "k8s-a0b1c2d3e4-event.a0b1c2d3e4f5a6b7"
		k8sInvObjName = "k8s-0a1b2c3d4e-object.0a1b2c3d4e5f6a7b"
	)

	testcases := []struct {
		k8sInvObjKind      string
		k8sInvObjFieldPath string
		k8sType            string
		k8sReason          string
		k8sMessage         string
		evStatus           uint32
		evEntityName       string
		evCheckName        string
	}{
		{
			"Pod",
			"",
			"Warning",
			"Failed",
			"Error: ImagePullBackOff",
			1,
			k8sInvObjName,
			"pod-failed",
		},
		{
			"Pod",
			"spec.containers{myservice}",
			"Warning",
			"Failed",
			"Error: ImagePullBackOff",
			1,
			k8sInvObjName,
			"container-myservice-imagepullbackoff",
		},
		{
			"Pod",
			"spec.containers{myservice}",
			"Normal",
			"Pulling",
			"Pulling image \"wrongimage:latest\"",
			0,
			k8sInvObjName,
			"container-myservice-pulling",
		},
		{
			"Node",
			"spec.containers{myservice}",
			"Warning",
			"Failed",
			"Error: ImagePullBackOff",
			1,
			k8sInvObjName,
			"failed",
		},
		{
			"Cluster",
			"spec.containers{myservice}",
			"Warning",
			"Failed",
			"Error: BackOff",
			1,
			k8sInvObjName + "-cluster",
			"BackOff",
		},
		{
			"ReplicaSet",
			"",
			"Normal",
			"SuccessfulDelete",
			"Deleted pod: myservice-bbd465f66-nbrpw",
			0,
			"myservice-bbd465f66-nbrpw",
			"pod-deleted",
		},
	}

	// plugin constants
	plugin.StatusMap = `{"normal": 0, "warning": 1, "default": 3}`
	plugin.Interval = 60
	plugin.Handlers = []string{"slack"}
	plugin.PluginConfig.Name = "kubernetes-event=check"

	for _, tc := range testcases {
		assert := assert.New(t)
		k8sev := k8scorev1.Event{}
		k8sev.InvolvedObject = k8scorev1.ObjectReference{}
		k8sev.ObjectMeta = metav1.ObjectMeta{}

		// k8s event constants
		k8sev.ObjectMeta.Namespace = "namespace"
		k8sev.Count = 1
		k8sev.ObjectMeta.Name = k8sObjName
		k8sev.InvolvedObject.Name = k8sInvObjName

		// test cases
		k8sev.InvolvedObject.Kind = tc.k8sInvObjKind
		k8sev.InvolvedObject.FieldPath = tc.k8sInvObjFieldPath
		k8sev.Type = tc.k8sType
		k8sev.Reason = tc.k8sReason
		k8sev.Message = tc.k8sMessage

		ev, err := createSensuEvent(k8sev)
		assert.NoError(err)
		assert.Equal(tc.evStatus, ev.Check.Status)
		assert.Equal(tc.evEntityName, ev.Check.ProxyEntityName)
		assert.Equal(tc.evCheckName, ev.Check.ObjectMeta.Name)
	}
}

func TestSubmitEventAgentAPI(t *testing.T) {
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")
	var test = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(err)
		eV := &corev2.Event{}
		err = json.Unmarshal(body, eV)
		require.NoError(t, err)
	}))
	_, err := url.ParseRequestURI(test.URL)
	require.NoError(t, err)
	plugin.AgentAPIURL = test.URL
	assert.NoError(submitEventAgentAPI(event))
}

func TestGetSensuEventStatus(t *testing.T) {
	testcases := []struct {
		statusMap    string
		k8sEventType string
		status       uint32
	}{
		{`{"normal": 0, "warning": 1, "default": 3}`, "Normal", 0},
		{`{"normal": 0, "warning": 1, "default": 3}`, "Warning", 1},
		{`{"normal": 0, "warning": 1, "default": 3}`, "NoMatch", 3},
		{`{"Normal": 0, "Warning": 1, "Default": 3}`, "normal", 0},
		{`{"Normal": 0, "Warning": 1, "Default": 3}`, "warning", 1},
		{`{"Normal": 0, "Warning": 1, "Default": 3}`, "nomatch", 3},
		{`{"normal": 0, "warning": 2, "default": 3}`, "Warning", 2},
		{`{"warning": 1, "default": 3}`, "Normal", 3},
		{`{"normal": 0, "warning": 1}`, "NoMatch", 255},
	}
	for _, tc := range testcases {
		assert := assert.New(t)
		plugin.StatusMap = tc.statusMap
		st, err := getSensuEventStatus(tc.k8sEventType)
		assert.NoError(err)
		assert.Equal(tc.status, st)
	}
}
