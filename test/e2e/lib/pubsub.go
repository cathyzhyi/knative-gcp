/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lib

import (
	"net/http"
	"os"
	"testing"
	"time"

	reconcilertestingv1 "github.com/google/knative-gcp/pkg/reconciler/testing/v1"
	reconcilertestingv1alpha1 "github.com/google/knative-gcp/pkg/reconciler/testing/v1alpha1"
	reconcilertestingv1beta1 "github.com/google/knative-gcp/pkg/reconciler/testing/v1beta1"

	v1 "k8s.io/api/core/v1"

	schemasv1 "github.com/google/knative-gcp/pkg/schemas/v1"
	"github.com/google/knative-gcp/test/e2e/lib/metrics"
	"github.com/google/knative-gcp/test/e2e/lib/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgmetrics "knative.dev/pkg/metrics"
)

type PubSubConfig struct {
	SinkGVK            metav1.GroupVersionKind
	PubSubName         string
	SinkName           string
	TopicName          string
	ServiceAccountName string
}

func MakePubSubOrDie(client *Client, config PubSubConfig) {
	client.T.Helper()
	so := make([]reconcilertestingv1.CloudPubSubSourceOption, 0)
	so = append(so, reconcilertestingv1.WithCloudPubSubSourceSink(config.SinkGVK, config.SinkName))
	so = append(so, reconcilertestingv1.WithCloudPubSubSourceTopic(config.TopicName))
	so = append(so, reconcilertestingv1.WithCloudPubSubSourceServiceAccount(config.ServiceAccountName))
	eventsPubSub := reconcilertestingv1.NewCloudPubSubSource(config.PubSubName, client.Namespace, so...)
	client.CreatePubSubOrFail(eventsPubSub)

	client.Core.WaitForResourceReadyOrFail(config.PubSubName, CloudPubSubSourceV1TypeMeta)
}

func MakePubSubV1beta1OrDie(client *Client, config PubSubConfig) {
	client.T.Helper()
	so := make([]reconcilertestingv1beta1.CloudPubSubSourceOption, 0)
	so = append(so, reconcilertestingv1beta1.WithCloudPubSubSourceSink(config.SinkGVK, config.SinkName))
	so = append(so, reconcilertestingv1beta1.WithCloudPubSubSourceTopic(config.TopicName))
	so = append(so, reconcilertestingv1beta1.WithCloudPubSubSourceServiceAccount(config.ServiceAccountName))
	eventsPubSub := reconcilertestingv1beta1.NewCloudPubSubSource(config.PubSubName, client.Namespace, so...)
	client.CreatePubSubV1beta1OrFail(eventsPubSub)

	client.Core.WaitForResourceReadyOrFail(config.PubSubName, CloudPubSubSourceV1beta1TypeMeta)
}

func MakePubSubV1alpha1OrDie(client *Client, config PubSubConfig) {
	client.T.Helper()
	so := make([]reconcilertestingv1alpha1.CloudPubSubSourceOption, 0)
	so = append(so, reconcilertestingv1alpha1.WithCloudPubSubSourceSink(config.SinkGVK, config.SinkName))
	so = append(so, reconcilertestingv1alpha1.WithCloudPubSubSourceTopic(config.TopicName))
	so = append(so, reconcilertestingv1alpha1.WithCloudPubSubSourceServiceAccount(config.ServiceAccountName))
	eventsPubSub := reconcilertestingv1alpha1.NewCloudPubSubSource(config.PubSubName, client.Namespace, so...)
	client.CreatePubSubV1alpha1OrFail(eventsPubSub)

	client.Core.WaitForResourceReadyOrFail(config.PubSubName, CloudPubSubSourceV1alpha1TypeMeta)
}

func MakePubSubTargetJobOrDie(client *Client, source, targetName, eventType string) {
	client.T.Helper()
	job := resources.PubSubTargetJob(targetName, []v1.EnvVar{
		{
			Name:  "TYPE",
			Value: eventType,
		},
		{
			Name:  "SOURCE",
			Value: source,
		}, {
			Name:  "TIME",
			Value: "6m",
		}})
	client.CreateJobOrFail(job, WithServiceForJob(targetName))
}

func AssertMetrics(t *testing.T, client *Client, topicName, psName string) {
	t.Helper()
	sleepTime := 1 * time.Minute
	t.Logf("Sleeping %s to make sure metrics were pushed to stackdriver", sleepTime.String())
	time.Sleep(sleepTime)

	// If we reach this point, the projectID should have been set.
	projectID := os.Getenv(ProwProjectKey)
	f := map[string]interface{}{
		"metric.type":                 EventCountMetricType,
		"resource.type":               GlobalMetricResourceType,
		"metric.label.resource_group": PubsubResourceGroup,
		"metric.label.event_type":     schemasv1.CloudPubSubMessagePublishedEventType,
		"metric.label.event_source":   schemasv1.CloudPubSubEventSource(projectID, topicName),
		"metric.label.namespace_name": client.Namespace,
		"metric.label.name":           psName,
		// We exit the target image before sending a response, thus check for 500.
		"metric.label.response_code":       http.StatusInternalServerError,
		"metric.label.response_code_class": pkgmetrics.ResponseCodeClass(http.StatusInternalServerError),
	}

	filter := metrics.StringifyStackDriverFilter(f)
	t.Logf("Filter expression: %s", filter)

	actualCount, err := client.StackDriverEventCountMetricFor(client.Namespace, projectID, filter)
	if err != nil {
		t.Fatalf("failed to get stackdriver event count metric: %v", err)
	}
	expectedCount := int64(1)
	if actualCount != expectedCount {
		t.Errorf("Actual count different than expected count, actual: %d, expected: %d", actualCount, expectedCount)
	}
}
