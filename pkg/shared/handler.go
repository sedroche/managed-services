package shared

import (
	"context"

	"github.com/aerogear/shared-service-operator-poc/pkg/apis/aerogear/v1alpha1"

	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	sc "github.com/aerogear/mobile-crd-client/pkg/client/servicecatalog/clientset/versioned"

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
	fmt.Printf("sc: %+v", sc)
	if service.Status.Ready {
		//delete the pod
	}
	if service.Status.Status == "" {
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: service.Name + "-",
				//Labels: extContext.Metadata,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  service.Name + "-create",
						Image: service.Spec.Image,
						Args: []string{
							"provision",
							"--extra-vars",
							"", // need to figure out how to get and pass the needed params
						},
						//Env:             createPodEnv(extContext),
						ImagePullPolicy: "Always",
					},
				},
				RestartPolicy: v1.RestartPolicyNever,
				//ServiceAccountName: extContext.Account,
				//Volumes:            volumes,
			},
		}

		logrus.Infof(fmt.Sprintf("Creating pod %q in the %s namespace", pod.Name, h.operatorNS))
		_, err := h.k8client.CoreV1().Pods(h.operatorNS).Create(pod)
		if err != nil {
			return err
		}
		//watch pod until complete and update the status of the shared service
	}
	service.Status.Status = "provisioning"
	unstructObj := k8sutil.UnstructuredFromRuntimeObject(service)
	if _, err := h.sharedServiceClient.Update(unstructObj); err != nil {
		return err
	}

	return nil
}

func (h *Handler) handleSharedServiceDelete(service *v1alpha1.SharedService) error {
	fmt.Println("called handleSharedServiceDelete")
	return nil
}

func provision(serviceType string)error{
	return  nil
}

func checkServiceInstanceReady(sid string)(bool,error)  {
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
		return nil
	}


	if ssCopy.Status.Action != "provisioning"{
		err := provision(ssCopy.Spec.ServiceType)
		if err != nil && !apierrors.IsNotFound(err){
			// if is a not found err return
			return err
		}
		ssCopy.Status.Action = "provisioning"
		return sdk.Update(ssCopy)
	}

	ready, err := checkServiceInstanceReady("")
	if err != nil{
		return err
	}
	if ready{
		ssCopy.Status.Phase = v1alpha1.CompletePhase
		// get the secret name and add to the service slice
		ssCopy.Status.CredentialREf = "somesecret"
		return sdk.Update(ssCopy)
	}

	// do provision

	return sdk.Update(ssCopy)
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
