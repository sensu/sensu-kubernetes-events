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
	event := corev2.FixtureEvent("entity1", "check1")
	status, err := checkArgs(event)
	assert.NoError(err)
	assert.Equal(status, sensu.CheckStateOK)
	assert.Equal(plugin.EventType, "=Normal")
}

func TestCreateSensuEvent(t *testing.T) {
	assert := assert.New(t)
	k8sev := k8scorev1.Event{}
	k8sev.InvolvedObject = k8scorev1.ObjectReference{}
	k8sev.ObjectMeta = metav1.ObjectMeta{}
	k8sev.ObjectMeta.Namespace = "namespace"
	k8sev.ObjectMeta.Name = "sensu-a0b1c2d3e4-test.a0b1c2d3e4f5a6b7"
	k8sev.InvolvedObject.Kind = "Pod"
	k8sev.InvolvedObject.Name = "test-0a1b2c3d4e-sensu.0a1b2c3d4e5f6a7b"
	k8sev.Type = "Warning"
	k8sev.Reason = "Because"
	k8sev.Message = "All your base belong to us"
	k8sev.Count = 1
	plugin.StatusMap = `{"normal": 0, "warning": 1, "default": 3}`
	plugin.Interval = 60
	plugin.Handlers = []string{"slack"}
	plugin.PluginConfig.Name = "kubernetes-event=check"
	ev, err := createSensuEvent(k8sev)
	assert.NoError(err)
	assert.Equal(ev.Check.Status, uint32(1))
	assert.Equal(ev.Check.ProxyEntityName, "test-0a1b2c3d4e-sensu.0a1b2c3d4e5f6a7b")
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
	plugin.EventAPI = test.URL
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
		assert.Equal(st, tc.status)
	}
}

/* func TestCheckArgs(t *testing.T) {
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")
	event.Metrics = corev2.FixtureMetrics()
	assert.Error(checkArgs(event))
	plugin.Index = "sensu_events"
	assert.Error(checkArgs(event))
	plugin.URL = []string{"http://localhost:9200"}
	assert.NoError(checkArgs(event))
	plugin.Username = "user1"
	assert.Error(checkArgs(event))
	plugin.Password = "password1"
	assert.NoError(checkArgs(event))
}

func TestParseEvent(t *testing.T) {
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")
	event.Metrics = corev2.FixtureMetrics()
	plugin.Index = "sensu_events"
	_, err := parseEvent(event)
	assert.NoError(err)
}

func TestExecuteHandler(t *testing.T) {
	testcases := []struct {
		datedIndex   bool
		indexURLPath string
	}{
		{false, "/sensu_events/_doc"},
		{true, fmt.Sprintf("/sensu_events-%s/_doc", time.Now().Format("2006.01.02"))},
	}

	for _, tc := range testcases {
		assert := assert.New(t)
		event := corev2.FixtureEvent("entity1", "check1")
		event.Metrics = corev2.FixtureMetrics()
		plugin.Index = "sensu_events"
		plugin.DatedIndex = tc.datedIndex

		var test = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(html.EscapeString(r.URL.Path), tc.indexURLPath)
			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(err)
			eV := &EventValue{}
			err = json.Unmarshal(body, eV)
			assert.Equal(eV.Measurements["answer"], float64(42))
			assert.Equal(eV.Metadata.Labels["answer.foo"], "bar")
			require.NoError(t, err)
		}))
		_, err := url.ParseRequestURI(test.URL)
		require.NoError(t, err)
		plugin.URL = []string{test.URL}
		assert.NoError(executeHandler(event))
	}
}
*/
