/*
Copyright 2016 The Kubernetes Authors.

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

package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/aerogear/managed-services/pkg/apis/aerogear/v1alpha1"
	"github.com/aerogear/managed-services/pkg/broker"
	brokerapi "github.com/aerogear/managed-services/pkg/broker"
	glog "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Controller defines the APIs that all controllers are expected to support. Implementations
// should be concurrency-safe
type Controller interface {
	Catalog() (*broker.Catalog, error)

	GetServiceInstanceLastOperation(instanceID, serviceID, planID, operation string) (*broker.LastOperationResponse, error)
	CreateServiceInstance(instanceID string, req *broker.CreateServiceInstanceRequest) (*broker.CreateServiceInstanceResponse, error)
	RemoveServiceInstance(instanceID, serviceID, planID string, acceptsIncomplete bool) (*broker.DeleteServiceInstanceResponse, error)

	Bind(instanceID, bindingID string, req *broker.BindingRequest) (*broker.CreateServiceBindingResponse, error)
	UnBind(instanceID, bindingID, serviceID, planID string) error
}

type errNoSuchInstance struct {
	instanceID string
}

func (e errNoSuchInstance) Error() string {
	return fmt.Sprintf("no such instance with ID %s", e.instanceID)
}

type userProvidedServiceInstance struct {
	Name       string
	Credential *brokerapi.Credential
}

type userProvidedController struct {
	rwMutex             sync.RWMutex
	instanceMap         map[string]*userProvidedServiceInstance
	sharedServiceClient dynamic.ResourceInterface
}

// CreateController creates an instance of a User Provided service broker controller.
func CreateController(sharedServiceClient dynamic.ResourceInterface) Controller {
	var instanceMap = make(map[string]*userProvidedServiceInstance)
	return &userProvidedController{
		instanceMap:         instanceMap,
		sharedServiceClient: sharedServiceClient,
	}
}

func (c *userProvidedController) Catalog() (*brokerapi.Catalog, error) {
	glog.Info("Catalog()")
	//look up the sharedservice in the namespace

	listed, err := c.sharedServiceClient.List(metav1.ListOptions{})
	listed.(*unstructured.UnstructuredList).EachListItem(func(object runtime.Object) error {
		bytes, err := object.(*unstructured.Unstructured).MarshalJSON()
		if err != nil{
			return err
		}
		s := &v1alpha1.SharedService{}
		if err := json.Unmarshal(bytes,s); err != nil{
			return err
		}
		fmt.Println("shared service is ", s)
		return nil
	})
	var services []*brokerapi.Service
	// _ = services
	_ = err

	fmt.Println("%v \n", reflect.TypeOf(listed))
	li := v1alpha1.SharedServiceList{}
	// _ = ok
	for _, sharedServiceCrd := range li.Items {
		fmt.Printf("%v\n dhsgsahgdshadgha", sharedServiceCrd)
		// 	if sharedServiceCrd.Status.Ready {

		// 	}

		// 	services = append(services, &brokerapi.Service{
		// 		Name:        "user-provided-service",
		// 		ID:          "4f6e6cf6-ffdd-425f-a2c7-3c9258ad2468",
		// 		Description: "A user provided service",
		// 		Plans: []brokerapi.ServicePlan{{
		// 			Name:        "default",
		// 			ID:          "86064792-7ea2-467b-af93-ac9694d96d52",
		// 			Description: "Sample plan description",
		// 			Free:        true,
		// 		}, {
		// 			Name:        "premium",
		// 			ID:          "cc0d7529-18e8-416d-8946-6f7456acd589",
		// 			Description: "Premium plan",
		// 			Free:        false,
		// 		},
		// 		},
		// 		Bindable:       true,
		// 		PlanUpdateable: true,
		// 	})
	}

	return &brokerapi.Catalog{
		Services: services,
	}, nil
	// fmt.Printf("%v %v\n dhsgsahgdshadgha", err, listed)
	// return &brokerapi.Catalog{
	// 	Services: []*brokerapi.Service{
	// 		{
	// 			Name:        "user-provided-service",
	// 			ID:          "4f6e6cf6-ffdd-425f-a2c7-3c9258ad2468",
	// 			Description: "A user provided service",
	// 			Plans: []brokerapi.ServicePlan{{
	// 				Name:        "default",
	// 				ID:          "86064792-7ea2-467b-af93-ac9694d96d52",
	// 				Description: "Sample plan description",
	// 				Free:        true,
	// 			}, {
	// 				Name:        "premium",
	// 				ID:          "cc0d7529-18e8-416d-8946-6f7456acd589",
	// 				Description: "Premium plan",
	// 				Free:        false,
	// 			},
	// 			},
	// 			Bindable:       true,
	// 			PlanUpdateable: true,
	// 		},
	// 		{
	// 			Name:        "user-provided-service-single-plan",
	// 			ID:          "5f6e6cf6-ffdd-425f-a2c7-3c9258ad2468",
	// 			Description: "A user provided service",
	// 			Plans: []brokerapi.ServicePlan{
	// 				{
	// 					Name:        "default",
	// 					ID:          "96064792-7ea2-467b-af93-ac9694d96d52",
	// 					Description: "Sample plan description",
	// 					Free:        true,
	// 				},
	// 			},
	// 			Bindable:       true,
	// 			PlanUpdateable: true,
	// 		},
	// 		{
	// 			Name:        "user-provided-service-with-schemas",
	// 			ID:          "8a6229d4-239e-4790-ba1f-8367004d0473",
	// 			Description: "A user provided service",
	// 			Plans: []brokerapi.ServicePlan{
	// 				{
	// 					Name:        "default",
	// 					ID:          "4dbcd97c-c9d2-4c6b-9503-4401a789b558",
	// 					Description: "Plan with parameter and response schemas",
	// 					Free:        true,
	// 					Schemas: &brokerapi.Schemas{
	// 						ServiceInstance: &brokerapi.ServiceInstanceSchema{
	// 							Create: &brokerapi.InputParametersSchema{
	// 								Parameters: map[string]interface{}{ // TODO: use a JSON Schema library instead?
	// 									"$schema": "http://json-schema.org/draft-04/schema#",
	// 									"type":    "object",
	// 									"properties": map[string]interface{}{
	// 										"param-1": map[string]interface{}{
	// 											"description": "First input parameter",
	// 											"type":        "string",
	// 										},
	// 										"param-2": map[string]interface{}{
	// 											"description": "Second input parameter",
	// 											"type":        "string",
	// 										},
	// 									},
	// 								},
	// 							},
	// 							Update: &brokerapi.InputParametersSchema{
	// 								Parameters: map[string]interface{}{
	// 									"$schema": "http://json-schema.org/draft-04/schema#",
	// 									"type":    "object",
	// 									"properties": map[string]interface{}{
	// 										"param-1": map[string]interface{}{
	// 											"description": "First input parameter",
	// 											"type":        "string",
	// 										},
	// 										"param-2": map[string]interface{}{
	// 											"description": "Second input parameter",
	// 											"type":        "string",
	// 										},
	// 									},
	// 								},
	// 							},
	// 						},
	// 						ServiceBinding: &brokerapi.ServiceBindingSchema{
	// 							Create: &brokerapi.RequestResponseSchema{
	// 								InputParametersSchema: brokerapi.InputParametersSchema{
	// 									Parameters: map[string]interface{}{
	// 										"$schema": "http://json-schema.org/draft-04/schema#",
	// 										"type":    "object",
	// 										"properties": map[string]interface{}{
	// 											"param-1": map[string]interface{}{
	// 												"description": "First input parameter",
	// 												"type":        "string",
	// 											},
	// 											"param-2": map[string]interface{}{
	// 												"description": "Second input parameter",
	// 												"type":        "string",
	// 											},
	// 										},
	// 									},
	// 								},
	// 								Response: map[string]interface{}{
	// 									"$schema": "http://json-schema.org/draft-04/schema#",
	// 									"type":    "object",
	// 									"properties": map[string]interface{}{
	// 										"credentials": map[string]interface{}{
	// 											"type": "object",
	// 											"properties": map[string]interface{}{
	// 												"special-key-1": map[string]interface{}{
	// 													"description": "Special key 1",
	// 													"type":        "string",
	// 												},
	// 												"special-key-2": map[string]interface{}{
	// 													"description": "Special key 2",
	// 													"type":        "string",
	// 												},
	// 											},
	// 										},
	// 									},
	// 								},
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 			Bindable:       true,
	// 			PlanUpdateable: true,
	// 		},
	// 	},
	// }, nil
}

func (c *userProvidedController) CreateServiceInstance(
	id string,
	req *brokerapi.CreateServiceInstanceRequest,
) (*brokerapi.CreateServiceInstanceResponse, error) {
	glog.Info("CreateServiceInstance()")
	credString, ok := req.Parameters["credentials"]
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	if ok {
		jsonCred, err := json.Marshal(credString)
		if err != nil {
			glog.Errorf("Failed to marshal credentials: %v", err)
			return nil, err
		}
		var cred brokerapi.Credential
		err = json.Unmarshal(jsonCred, &cred)
		if err != nil {
			glog.Errorf("Failed to unmarshal credentials: %v", err)
			return nil, err
		}

		c.instanceMap[id] = &userProvidedServiceInstance{
			Name:       id,
			Credential: &cred,
		}
	} else {
		c.instanceMap[id] = &userProvidedServiceInstance{
			Name: id,
			Credential: &brokerapi.Credential{
				"special-key-1": "special-value-1",
				"special-key-2": "special-value-2",
			},
		}
	}

	glog.Infof("Created User Provided Service Instance:\n%v\n", c.instanceMap[id])
	return &brokerapi.CreateServiceInstanceResponse{}, nil
}

func (c *userProvidedController) GetServiceInstanceLastOperation(
	instanceID,
	serviceID,
	planID,
	operation string,
) (*brokerapi.LastOperationResponse, error) {
	glog.Info("GetServiceInstanceLastOperation()")
	return nil, errors.New("Unimplemented")
}

func (c *userProvidedController) RemoveServiceInstance(
	instanceID,
	serviceID,
	planID string,
	acceptsIncomplete bool,
) (*brokerapi.DeleteServiceInstanceResponse, error) {
	glog.Info("RemoveServiceInstance()")
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	_, ok := c.instanceMap[instanceID]
	if ok {
		delete(c.instanceMap, instanceID)
		return &brokerapi.DeleteServiceInstanceResponse{}, nil
	}

	return &brokerapi.DeleteServiceInstanceResponse{}, nil
}

func (c *userProvidedController) Bind(
	instanceID,
	bindingID string,
	req *brokerapi.BindingRequest,
) (*brokerapi.CreateServiceBindingResponse, error) {
	glog.Info("Bind()")
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	instance, ok := c.instanceMap[instanceID]
	if !ok {
		return nil, errNoSuchInstance{instanceID: instanceID}
	}
	cred := instance.Credential
	return &brokerapi.CreateServiceBindingResponse{Credentials: *cred}, nil
}

func (c *userProvidedController) UnBind(instanceID, bindingID, serviceID, planID string) error {
	glog.Info("UnBind()")
	// Since we don't persist the binding, there's nothing to do here.
	return nil
}
