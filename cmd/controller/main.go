/*
Copyright 2019 Google LLC

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

package main

import (
	"context"
	"log"

	"google.golang.org/api/option"

	// The following line to load the gcp plugin (only required to authenticate against GKE clusters).

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	apixclient "knative.dev/pkg/client/injection/apiextensions/client"

	"github.com/google/knative-gcp/pkg/reconciler/broker"
	"github.com/google/knative-gcp/pkg/reconciler/brokercell"
	"github.com/google/knative-gcp/pkg/reconciler/deployment"
	"github.com/google/knative-gcp/pkg/reconciler/events/auditlogs"
	"github.com/google/knative-gcp/pkg/reconciler/events/build"
	"github.com/google/knative-gcp/pkg/reconciler/events/pubsub"
	"github.com/google/knative-gcp/pkg/reconciler/events/scheduler"
	"github.com/google/knative-gcp/pkg/reconciler/events/storage"
	kedapullsubscription "github.com/google/knative-gcp/pkg/reconciler/intevents/pullsubscription/keda"
	staticpullsubscription "github.com/google/knative-gcp/pkg/reconciler/intevents/pullsubscription/static"
	"github.com/google/knative-gcp/pkg/reconciler/intevents/topic"
	"github.com/google/knative-gcp/pkg/reconciler/messaging/channel"
	"github.com/google/knative-gcp/pkg/reconciler/trigger"
	"github.com/google/knative-gcp/pkg/utils/appcredentials"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
)

func checkKnativeInstalled(ctx context.Context) {
	//var ret *rest.RESTClient
	client := apixclient.Get(ctx)
	getter := client.ApiextensionsV1().CustomResourceDefinitions()
	checkCRDExists(getter, "broker.eventing.knative.dev")
}

func checkCRDExists(getter apiextensionsv1.CustomResourceDefinitionInterface , name string) {
	crd, err := getter.Get(name, metav1.GetOptions{})
	if err != nil  || crd != nil {
		log.Fatal("Knative is not properly installed because crd " + name + " is missing")
	} 
}

func main() {
	appcredentials.MustExistOrUnsetEnv()
	ctx := signals.NewContext()
	checkKnativeInstalled(ctx)
	controllers, err := InitializeControllers(ctx)
	if err != nil {
		log.Fatal(err)
	}
	sharedmain.MainWithContext(ctx, "controller", controllers...)
}

func Controllers(
	auditlogsController auditlogs.Constructor,
	storageController storage.Constructor,
	schedulerController scheduler.Constructor,
	pubsubController pubsub.Constructor,
	buildController build.Constructor,
	pullsubscriptionController staticpullsubscription.Constructor,
	kedaPullsubscriptionController kedapullsubscription.Constructor,
	topicController topic.Constructor,
	channelController channel.Constructor,
) []injection.ControllerConstructor {
	return []injection.ControllerConstructor{
		injection.ControllerConstructor(auditlogsController),
		injection.ControllerConstructor(storageController),
		injection.ControllerConstructor(schedulerController),
		injection.ControllerConstructor(pubsubController),
		injection.ControllerConstructor(buildController),
		injection.ControllerConstructor(pullsubscriptionController),
		injection.ControllerConstructor(kedaPullsubscriptionController),
		injection.ControllerConstructor(topicController),
		injection.ControllerConstructor(channelController),
		deployment.NewController,
		broker.NewController,
		trigger.NewController,
		brokercell.NewController,
	}
}

func ClientOptions() []option.ClientOption {
	return nil
}
