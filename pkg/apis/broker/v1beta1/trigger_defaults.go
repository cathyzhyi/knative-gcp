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

package v1beta1

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apixclient "knative.dev/pkg/client/injection/apiextensions/client"
)

// SetDefaults sets the default field values for a Trigger.
func (t *Trigger) SetDefaults(ctx context.Context) {
	client := apixclient.Get(ctx)
	getter := client.ApiextensionsV1().CustomResourceDefinitions()
	crd, _ := getter.Get(ctx, "Service.serving.knative.dev", metav1.GetOptions{})
	fmt.Println(crd.GetName())

	// The Google Cloud Broker doesn't have any custom defaults. The
	// eventing webhook will add the usual defaults.
}
