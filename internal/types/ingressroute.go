package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: "traefik.io", Version: "v1alpha1"}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&IngressRoute{},
		&IngressRouteList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

type IngressRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              IngressRouteSpec `json:"spec,omitempty"`
}

type IngressRouteSpec struct {
	Routes []Route `json:"routes"`
}

type Route struct {
	Match string `json:"match"`
	Kind  string `json:"kind"`
}

type IngressRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IngressRoute `json:"items"`
}

func (in *IngressRoute) DeepCopyObject() runtime.Object {
	out := &IngressRoute{}
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec.Routes = make([]Route, len(in.Spec.Routes))
	copy(out.Spec.Routes, in.Spec.Routes)
	return out
}

func (in *IngressRouteList) DeepCopyObject() runtime.Object {
	out := &IngressRouteList{}
	*out = *in
	out.Items = make([]IngressRoute, len(in.Items))
	for i := range in.Items {
		out.Items[i] = *in.Items[i].DeepCopyObject().(*IngressRoute)
	}
	return out
}
