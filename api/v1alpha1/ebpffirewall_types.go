/*
Copyright 2023.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EbpffirewallSpec defines the desired state of Ebpffirewall
type EbpffirewallSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Ebpffirewall. Edit ebpffirewall_types.go to remove/update
	//Foo string `json:"foo,omitempty"`
	Block []Block `json:"block,omitempty"`
}

// EbpffirewallStatus defines the observed state of Ebpffirewall
type EbpffirewallStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Blocklist []string `json:"Blocklist,omitempty"`
}

type Block struct {
	// +kubebuilder:validation:Pattern=`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	Ip string `json:"ip,omitempty"`
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9.-]+$`
	Dns string `json:"dns,omitempty"`
	// +kubebuilder:validation:Pattern=`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	Subnet string `json:"subnet,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Ebpffirewall is the Schema for the ebpffirewalls API
type Ebpffirewall struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EbpffirewallSpec   `json:"spec,omitempty"`
	Status EbpffirewallStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EbpffirewallList contains a list of Ebpffirewall
type EbpffirewallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Ebpffirewall `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Ebpffirewall{}, &EbpffirewallList{})
}
