/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha "headnode/pkg/apis/apps/v1alpha"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeLogins implements LoginInterface
type FakeLogins struct {
	Fake *FakeAppsV1alpha
	ns   string
}

var loginsResource = schema.GroupVersionResource{Group: "apps.edgenet.io", Version: "v1alpha", Resource: "logins"}

var loginsKind = schema.GroupVersionKind{Group: "apps.edgenet.io", Version: "v1alpha", Kind: "Login"}

// Get takes name of the login, and returns the corresponding login object, and an error if there is any.
func (c *FakeLogins) Get(name string, options v1.GetOptions) (result *v1alpha.Login, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(loginsResource, c.ns, name), &v1alpha.Login{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.Login), err
}

// List takes label and field selectors, and returns the list of Logins that match those selectors.
func (c *FakeLogins) List(opts v1.ListOptions) (result *v1alpha.LoginList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(loginsResource, loginsKind, c.ns, opts), &v1alpha.LoginList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha.LoginList{ListMeta: obj.(*v1alpha.LoginList).ListMeta}
	for _, item := range obj.(*v1alpha.LoginList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested logins.
func (c *FakeLogins) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(loginsResource, c.ns, opts))

}

// Create takes the representation of a login and creates it.  Returns the server's representation of the login, and an error, if there is any.
func (c *FakeLogins) Create(login *v1alpha.Login) (result *v1alpha.Login, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(loginsResource, c.ns, login), &v1alpha.Login{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.Login), err
}

// Update takes the representation of a login and updates it. Returns the server's representation of the login, and an error, if there is any.
func (c *FakeLogins) Update(login *v1alpha.Login) (result *v1alpha.Login, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(loginsResource, c.ns, login), &v1alpha.Login{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.Login), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeLogins) UpdateStatus(login *v1alpha.Login) (*v1alpha.Login, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(loginsResource, "status", c.ns, login), &v1alpha.Login{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.Login), err
}

// Delete takes name of the login and deletes it. Returns an error if one occurs.
func (c *FakeLogins) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(loginsResource, c.ns, name), &v1alpha.Login{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeLogins) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(loginsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha.LoginList{})
	return err
}

// Patch applies the patch and returns the patched login.
func (c *FakeLogins) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha.Login, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(loginsResource, c.ns, name, pt, data, subresources...), &v1alpha.Login{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.Login), err
}
