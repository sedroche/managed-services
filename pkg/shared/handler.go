package shared

import (
	"context"

	"github.com/aerogear/shared-service-operator-poc/pkg/apis/aerogear/v1alpha1"

	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	sc "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/pkg/errors"
	"github.com/lestrrat/go-jsschema"
	"bytes"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"encoding/json"
	"os"
)

func NewHandler(k8sClient kubernetes.Interface, sharedServiceClient dynamic.ResourceInterface, operatorNS string, svcCatalog sc.Interface) sdk.Handler {
	return &Handler{
		k8client:            k8sClient,
		operatorNS:          operatorNS,
		sharedServiceClient: sharedServiceClient,
		serviceCatalogClient:svcCatalog,
	}
}

type Handler struct {
	// Fill me
	k8client            kubernetes.Interface
	operatorNS          string
	sharedServiceClient dynamic.ResourceInterface
	serviceCatalogClient sc.Interface
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {

	switch o := event.Object.(type) {
	case *v1alpha1.SharedService:
		fmt.Println("shared service recieved ", o.Namespace, o.Name, o.Status, event.Deleted)
		if event.Deleted {
			return h.handleSharedServiceDelete(o)
		}
		return h.handleSharedServiceCreateUpdate(o)
	case *v1alpha1.SharedServiceSlice:
		fmt.Println("shared service slice recieved ", o.Namespace, o.Name, o.Status, event.Deleted)
		if event.Deleted {
			return h.handleSharedServiceSliceDelete(o)
		}
		return h.handleSharedServiceSliceCreateUpdate(o)

	case *v1alpha1.SharedServiceClient:
		fmt.Println("shared service slice recieved ", o.Namespace, o.Name, o.Status, event.Deleted)
		if event.Deleted {
			return h.handleSharedServiceClientDelete(o)
		}
		return h.handleSharedServiceClientCreateUpdate(o)
	}
	return nil
}

func (h *Handler) handleSharedServiceCreateUpdate(service *v1alpha1.SharedService) error {
	fmt.Println("called handleSharedServiceCreateUpdate ")
	fmt.Printf("service: %+v", service)


	return nil
}

func (h *Handler) handleSharedServiceDelete(service *v1alpha1.SharedService) error {
	fmt.Println("called handleSharedServiceDelete")
	return nil
}

func (h *Handler)allocateSharedServiceInstanceWithCapacity(serviceType string)(*v1beta1.ServiceInstance, error)  {
	// look up sharedserviceinstanc of the given type and increment the capcity
	// look up the service instance referenced in the sharedserviceinstance and return it
	//work around for testing
	sharedServiceInstsnce := os.Getenv("SSI")
	return h.serviceCatalogClient.ServicecatalogV1beta1().ServiceInstances(h.operatorNS).Get(sharedServiceInstsnce,metav1.GetOptions{})
}


func (h*Handler) getParamValue(slice *v1alpha1.SharedServiceSlice, key string, paramSchema *schema.Schema)(interface{}, error) {
	// todo don't have  sharedserviceconfig to work with yet but we will do the following
	// look up the param in the shared service config
	// look up the param in the shared service instance secret (named after the service)
	// if not there return nil
	// it will then be pulled from the shared service slice params
	//hard coded for keycloak right now
	// get the secret we create from the apb
	//TODO this is currently broken but as we cannot deploy more than one keycloak until we change the apb it works !!
	superSecretCreds, err := h.k8client.CoreV1().Secrets(h.operatorNS).Get(slice.Spec.ServiceType, metav1.GetOptions{})
	if err != nil && ! apierrors.IsNotFound(err){
		return nil, errors.Wrap(err, "failed to find credentials secret for service ")
	}

	if v, ok := slice.Spec.Params[key];ok{
		return v, nil
	}
	if superSecretCreds != nil{
	  if v, ok := superSecretCreds.Data[key]; ok {
		  return string(v), nil
	  }
	}
	return nil, nil
	//take a service look up the SharedServiceConfig for that service pull out the default params as a map of key values

}

//TODO not happy with the signature here we are returning the parent and slice ids as strings which is not clear
func (h *Handler)provisionSlice(serviceSlice *v1alpha1.SharedServiceSlice, serviceType, plan string)(string, string, error){
	fmt.Println("provisioning slice")
	// find shared service with capacity of the given type
	si, err := h.allocateSharedServiceInstanceWithCapacity(serviceType)
	if err != nil{
		return  "","",errors.Wrap(err, "unexpected error when looking for a service instance with capacity")
	}
	if si == nil{
		// todo update status
		fmt.Println("no si found with capcity")
		return "","", errors.New("failed to find a service instance with capacity")
	}

	availablePlans, err := h.serviceCatalogClient.ServicecatalogV1beta1().ClusterServicePlans().List(metav1.ListOptions{FieldSelector:"spec.externalName=shared"})
	if err != nil{
		return "","", errors.Wrap(err, "failed to get service plans")
	}
	if len(availablePlans.Items) != 1{
		//this is bad
		return "","",errors.New(fmt.Sprintf("expected a single plan but found %v",len(availablePlans.Items)))
	}
	ap := availablePlans.Items[0]
	fmt.Println("plan name ", ap.Spec.ExternalName, string(ap.Spec.ServiceInstanceCreateParameterSchema.Raw))
	paramSchema, err := schema.Read(bytes.NewBuffer(ap.Spec.ServiceInstanceCreateParameterSchema.Raw))
	if err != nil{
		logrus.Error("failed to read schema", err)
	}

	params := map[string]interface{}{}

	if paramSchema != nil {
		for name, p := range paramSchema.Properties {
			fmt.Println("property ", p.Type, name)
			val , err := h.getParamValue(serviceSlice,name,p)
			if err != nil{
				// have to bail out no way forward
				return "","", err
			}
			params[name] = val
		}
	}

	pData, err := json.Marshal(params)
	if err != nil{
		return "","",errors.Wrap(err,"failed to encode params")
	}

	fmt.Println("params for slice provision ", string(pData))

	provisionInstance := &v1beta1.ServiceInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "servicecatalog.k8s.io/v1beta1",
			Kind:       "ServiceInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    h.operatorNS,
			GenerateName: serviceType + "-",
		},
		Spec: v1beta1.ServiceInstanceSpec{
			PlanReference: v1beta1.PlanReference{
				ClusterServiceClassExternalName: si.Spec.ClusterServiceClassExternalName,
				ClusterServicePlanExternalName:plan,
			},
			//TODO should prob come from secret
			Parameters:&runtime.RawExtension{
				Raw:pData,
			},
		},
	}

	fmt.Println("would have provisioned slice ", provisionInstance, "shared service ", si)
	csi, err := h.serviceCatalogClient.ServicecatalogV1beta1().ServiceInstances(h.operatorNS).Create(provisionInstance)
	if err != nil{
		return "","",err
	}
	return si.Name, csi.Name, nil
}

func (h *Handler)checkServiceInstanceReady(sid string)(bool,error)  {
	fmt.Println("checking service instance ready ", sid)
	if sid == ""{
		return false, nil
	}
	si, err := h.serviceCatalogClient.ServicecatalogV1beta1().ServiceInstances(h.operatorNS).Get(sid,metav1.GetOptions{})
	if err != nil{

	}
	if si == nil{
		return false, nil
	}

	fmt.Println("si status ", err, si.Status.Conditions)
	for _, c := range si.Status.Conditions{
		if c.Type == v1beta1.ServiceInstanceConditionReady{
			return c.Status == v1beta1.ConditionTrue, nil
		}
	}
	return false, nil
}

func (h *Handler)handleSharedServiceSliceCreateUpdate(service *v1alpha1.SharedServiceSlice)error{
	fmt.Println("called handleSharedServiceSliceCreateUpdate", service.Status.Phase, service.Status.Action)
	ssCopy := service.DeepCopy()
	if ssCopy.Status.Phase != v1alpha1.AcceptedPhase && ssCopy.Status.Phase != v1alpha1.CompletePhase{
		ssCopy.Status.Phase = v1alpha1.AcceptedPhase
		return sdk.Update(ssCopy)
	}
	if ssCopy.Status.Action == "provisioned"{
		// look up the secret and save to the shared service slice and set the status to complete
		ssCopy.Status.Phase = v1alpha1.CompletePhase
		return sdk.Update(ssCopy)
	}
	if ssCopy.Status.Action == "provisioning"{
		fmt.Print("provisioning")
		ready, err := h.checkServiceInstanceReady(ssCopy.Status.SliceServiceInstance)
		if err != nil{
			return err
		}
		if ready{
			ssCopy.Status.Phase = v1alpha1.CompletePhase
			ssCopy.Status.Action = "provisioned"
			// get the secret name and add to the service slice
			ssCopy.Status.CredentialRef = "somesecret"
			return sdk.Update(ssCopy)
		}
		return nil
	}


	if ssCopy.Status.Action != "provisioning"{
		sharedServiceID, sliceID ,err := h.provisionSlice(ssCopy,ssCopy.Spec.ServiceType, "shared")
		if err != nil && !apierrors.IsNotFound(err){
			// if is a not found err return
			return err
		}
		ssCopy.Status.Action = "provisioning"
		// not sure what I want here yet
		if ssCopy.Labels == nil{
			ssCopy.Labels = map[string]string{}
		}
		ssCopy.Labels["SliceServiceInstance"] = sliceID
		ssCopy.Labels["SharedServiceInstance"] = sharedServiceID
		ssCopy.Status.SliceServiceInstance= sliceID
		ssCopy.Status.SharedServiceInstance= sharedServiceID
		return sdk.Update(ssCopy)
	}



	return nil
}

func (h *Handler)handleSharedServiceSliceDelete(service *v1alpha1.SharedServiceSlice)error{
	fmt.Println("called handleSharedServiceSliceDelete")
	return nil
}

func (h *Handler) handleSharedServiceClientCreateUpdate(serviceClient *v1alpha1.SharedServiceClient) error {
	fmt.Println("called handleSharedServiceClientCreateUpdate")
	return nil
}

func (h *Handler) handleSharedServiceClientDelete(serviceClient *v1alpha1.SharedServiceClient) error {
	fmt.Println("called handleSharedServiceClientDelete")
	return nil
}
