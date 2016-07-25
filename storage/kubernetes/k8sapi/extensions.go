/*
Copyright 2015 The Kubernetes Authors.

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

package k8sapi

// A ThirdPartyResource is a generic representation of a resource, it is used by add-ons and plugins to add new resource
// types to the API.  It consists of one or more Versions of the api.
type ThirdPartyResource struct {
	TypeMeta `json:",inline"`

	// Standard object metadata
	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Description is the description of this object.
	Description string `json:"description,omitempty" protobuf:"bytes,2,opt,name=description"`

	// Versions are versions for this third party object
	Versions []APIVersion `json:"versions,omitempty" protobuf:"bytes,3,rep,name=versions"`
}

// ThirdPartyResourceList is a list of ThirdPartyResources.
type ThirdPartyResourceList struct {
	TypeMeta `json:",inline"`

	// Standard list metadata.
	ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of ThirdPartyResources.
	Items []ThirdPartyResource `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// An APIVersion represents a single concrete version of an object model.
type APIVersion struct {
	// Name of this version (e.g. 'v1').
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
}
