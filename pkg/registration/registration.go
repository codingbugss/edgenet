/*
Copyright 2019 Sorbonne Université

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

package registration

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/authorization"
	custconfig "edgenet/pkg/config"

	"k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
	cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
)

// CreateSpecificRoleBindings generates role bindings to allow users to access their user objects and the authority to which they belong
func CreateSpecificRoleBindings(userCopy *apps_v1alpha.User) {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// When a user is deleted, the owner references feature allows the related objects to be automatically removed
	userOwnerReferences := setOwnerReferences(userCopy)
	// Put the service account dedicated to the user into the role bind subjects
	rbSubjects := []rbacv1.Subject{{Kind: "User", Name: userCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"}}
	// This section allows the user to get user object that belongs to him. The role, which gets used by the binding object,
	// generated by the user controller when the user object created.
	roleName := fmt.Sprintf("user-%s", userCopy.GetName())
	roleRef := rbacv1.RoleRef{Kind: "Role", Name: roleName}
	roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: userCopy.GetNamespace(), Name: fmt.Sprintf("%s-%s", userCopy.GetNamespace(), roleName),
		OwnerReferences: userOwnerReferences}, Subjects: rbSubjects, RoleRef: roleRef}
	_, err = clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Create(roleBind)
	if err != nil {
		log.Printf("Couldn't create %s role binding in namespace of %s: %s, err: %s", roleName, userCopy.GetNamespace(), userCopy.GetName(), err)
		if errors.IsAlreadyExists(err) {
			userRoleBind, err := clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Get(roleBind.GetName(), metav1.GetOptions{})
			if err == nil {
				userRoleBind.Subjects = rbSubjects
				userRoleBind.RoleRef = roleRef
				_, err = clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Update(userRoleBind)
				if err == nil {
					log.Printf("Completed: role binding in namespace of %s: %s", userCopy.GetNamespace(), userCopy.GetName())
				}
			}
		}
	}

	// This section allows the user to get the authority object in which he/she participates. The role, which gets used by the binding object,
	// generated by the authority controller when the authority object created.
	userOwnerNamespace, _ := clientset.CoreV1().Namespaces().Get(userCopy.GetNamespace(), metav1.GetOptions{})
	roleName = fmt.Sprintf("authority-%s", userOwnerNamespace.Labels["authority-name"])
	roleRef = rbacv1.RoleRef{Kind: "ClusterRole", Name: roleName}
	clusterRoleBind := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s-for-authority", userCopy.GetNamespace(), userCopy.GetName()),
		OwnerReferences: userOwnerReferences}, Subjects: rbSubjects, RoleRef: roleRef}
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(clusterRoleBind)
	if err != nil {
		log.Printf("Couldn't create %s role binding in namespace of %s: %s", roleName, userCopy.GetNamespace(), userCopy.GetName())
		log.Println(err.Error())
	}
}

// EstablishRoleBindings generates the rolebindings according to user roles in the namespace specified
func EstablishRoleBindings(userCopy *apps_v1alpha.User, namespace string, namespaceType string) {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// When a user is deleted, the owner references feature allows the related objects to be automatically removed
	ownerReferences := setOwnerReferences(userCopy)
	// Put the service account dedicated to the user into the role bind subjects
	rbSubjects := []rbacv1.Subject{{Kind: "User", Name: userCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"}}
	// Roles are pre-generated by the controllers
	roleName := fmt.Sprintf("%s-%s", strings.ToLower(namespaceType), strings.ToLower(userCopy.Status.Type))
	roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: roleName}
	roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: fmt.Sprintf("%s-%s-%s", userCopy.GetNamespace(), userCopy.GetName(), roleName),
		OwnerReferences: ownerReferences}, Subjects: rbSubjects, RoleRef: roleRef}
	_, err = clientset.RbacV1().RoleBindings(namespace).Create(roleBind)
	if err != nil {
		log.Printf("Couldn't create %s role binding in namespace of %s: %s - %s, err: %s", userCopy.Status.Type, namespace, userCopy.GetNamespace(), userCopy.GetName(), err)
		if errors.IsAlreadyExists(err) {
			userRoleBind, err := clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Get(roleBind.GetName(), metav1.GetOptions{})
			if err == nil {
				userRoleBind.Subjects = rbSubjects
				userRoleBind.RoleRef = roleRef
				_, err = clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Update(userRoleBind)
				if err == nil {
					log.Printf("Completed: %s role binding in namespace of %s: %s - %s", userCopy.Status.Type, namespace, userCopy.GetNamespace(), userCopy.GetName())
				}
			}
		}
	}
}

// CreateServiceAccount makes a service account to serve the user. This functionality covers two types of service accounts
// in EdgeNet use, permanent for main use and temporary for safety.
func CreateServiceAccount(userCopy *apps_v1alpha.User, accountType string) (*corev1.ServiceAccount, error) {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Set the name of service account according to the type
	name := userCopy.GetName()
	ownerReferences := setOwnerReferences(userCopy)
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, OwnerReferences: ownerReferences}}
	serviceAccountCreated, err := clientset.CoreV1().ServiceAccounts(userCopy.GetNamespace()).Create(serviceAccount)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return serviceAccountCreated, nil
}

// CreateConfig checks serviceaccount of the user (actually, the namespace) to detect whether it contains the required.
// Then it gets that secret to use CA and token information. Subsequently, this reads cluster and server info of the current context
// from the config file to be consumed on the creation of kubeconfig.
func CreateConfig(serviceAccount *corev1.ServiceAccount) string {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// To find out the secret name to use
	accountSecretName := ""
	for _, accountSecret := range serviceAccount.Secrets {
		match, _ := regexp.MatchString("([a-z0-9]+)-token-([a-z0-9]+)", accountSecret.Name)
		if match {
			accountSecretName = accountSecret.Name
			break
		}
	}
	// If there is no matching secret terminate this function as generating kubeconfig file is not possible
	if accountSecretName == "" {
		log.Printf("Serviceaccount %s in %s doesn't have a serviceaccount token", serviceAccount.GetName(), serviceAccount.GetNamespace())
		return fmt.Sprintf("Serviceaccount %s doesn't have a serviceaccount token\n", serviceAccount.GetName())
	}
	secret, err := clientset.CoreV1().Secrets(serviceAccount.GetNamespace()).Get(accountSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Secret for %s in %s not found", serviceAccount.GetName(), serviceAccount.GetNamespace())
		return fmt.Sprintf("Secret %s not found\n", serviceAccount.GetName())
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting secret %s in %s: %v", serviceAccount.GetName(), serviceAccount.GetNamespace(), statusError.ErrStatus)
		return fmt.Sprintf("Error getting secret %s: %v\n", serviceAccount.GetName(), statusError.ErrStatus)
	} else if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Define the cluster and server by taking advantage of the current config file
	cluster, server, _, err := custconfig.GetClusterServerOfCurrentContext()
	if err != nil {
		log.Println(err)
		return fmt.Sprintf("Err: %s", err)
	}
	// Put the collected data into new kubeconfig file
	newKubeConfig := kubeconfigutil.CreateWithToken(server, cluster, serviceAccount.GetName(), secret.Data["ca.crt"], string(secret.Data["token"]))
	newKubeConfig.Contexts[newKubeConfig.CurrentContext].Namespace = serviceAccount.GetNamespace()
	kubeconfigutil.WriteToDisk(fmt.Sprintf("../../assets/kubeconfigs/edgenet-%s-%s.cfg", serviceAccount.GetNamespace(), serviceAccount.GetName()), newKubeConfig)
	// Check whether the creation process is completed
	dat, err := ioutil.ReadFile(fmt.Sprintf("../../assets/kubeconfigs/edgenet-%s-%s.cfg", serviceAccount.GetNamespace(), serviceAccount.GetName()))
	if err != nil {
		log.Println(err)
		return fmt.Sprintf("Err: %s", err)
	}
	return string(dat)
}

// setOwnerReferences put the user or userregistrationrequest as owner
func setOwnerReferences(objCopy interface{}) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newReference := metav1.OwnerReference{}
	switch userObj := objCopy.(type) {
	case *apps_v1alpha.UserRegistrationRequest:
		newReference = *metav1.NewControllerRef(userObj, apps_v1alpha.SchemeGroupVersion.WithKind("UserRegistrationRequest"))
	case *apps_v1alpha.User:
		newReference = *metav1.NewControllerRef(userObj, apps_v1alpha.SchemeGroupVersion.WithKind("User"))
	}
	takeControl := false
	newReference.Controller = &takeControl
	ownerReferences = append(ownerReferences, newReference)
	return ownerReferences
}

// MakeUser generates key and certificate and then set user credentials into the config file.
func MakeUser(authority, username, email string, clientset kubernetes.Interface) ([]byte, []byte, error) {
	path := fmt.Sprintf("../../assets/certs/%s", email)
	reader := rand.Reader
	bitSize := 4096

	key, err := rsa.GenerateKey(reader, bitSize)
	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	subject := pkix.Name{
		CommonName:   email,
		Organization: []string{authority},
	}
	dnsSANs := []string{"sandbox1.planet-lab.eu"}
	ipSANs := []net.IP{net.ParseIP("132.227.123.48")}

	csr, _ := cert.MakeCSR(key, &subject, dnsSANs, ipSANs)

	var CSRCopy *v1beta1.CertificateSigningRequest
	CSRObject := v1beta1.CertificateSigningRequest{}
	CSRObject.Name = fmt.Sprintf("%s-%s", authority, username)
	CSRObject.Spec.Groups = []string{"system:authenticated"}
	CSRObject.Spec.Usages = []v1beta1.KeyUsage{"digital signature", "key encipherment", "server auth", "client auth"}
	CSRObject.Spec.Request = csr
	CSRCopyCreated, err := clientset.CertificatesV1beta1().CertificateSigningRequests().Create(&CSRObject)
	if err != nil {
		return nil, nil, err
	}
	CSRCopy = CSRCopyCreated
	CSRCopy.Status.Conditions = append(CSRCopy.Status.Conditions, v1beta1.CertificateSigningRequestCondition{
		Type:           v1beta1.CertificateApproved,
		Reason:         "User creation is completed",
		Message:        "This CSR was approved automatically by EdgeNet",
		LastUpdateTime: metav1.Now(),
	})
	_, err = clientset.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(CSRCopy)
	if err != nil {
		return nil, nil, err
	}
	timeout := time.After(5 * time.Minute)
	ticker := time.Tick(15 * time.Second)
check:
	for {
		select {
		case <-timeout:
			return nil, nil, err
		case <-ticker:
			CSRCopy, err = clientset.CertificatesV1beta1().CertificateSigningRequests().Get(CSRCopy.GetName(), metav1.GetOptions{})
			if err != nil {
				return nil, nil, err
			}
			if len(CSRCopy.Status.Certificate) != 0 {
				break check
			}
		}
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s.crt", path), CSRCopy.Status.Certificate, 0700)
	if err != nil {
		return nil, nil, err
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s.key", path), pemdata, 0700)
	if err != nil {
		return nil, nil, err
	}
	pathOptions := clientcmd.NewDefaultPathOptions()
	buf := bytes.NewBuffer([]byte{})
	kcmd := cmdconfig.NewCmdConfigSetAuthInfo(buf, pathOptions)
	kcmd.SetArgs([]string{email})
	kcmd.Flags().Parse([]string{
		fmt.Sprintf("--client-certificate=../../assets/certs/%s.crt", email),
		fmt.Sprintf("--client-key=../../assets/certs/%s.key", email),
	})

	if err := kcmd.Execute(); err != nil {
		log.Printf("Couldn't set auth info on the kubeconfig file: %s", username)
		return nil, nil, err
	}
	return CSRCopy.Status.Certificate, pemdata, nil
}

// MakeConfig checks/gets serviceaccount of the user (actually, the namespace), and if the serviceaccount exists
// this function checks/gets its secret, and then CA and token info of the secret. Subsequently, this reads cluster
// and server info of the current context from the config file to use them on the creation of kubeconfig.
func MakeConfig(authority, username, email string, clientCert, clientKey []byte, clientset kubernetes.Interface) error {
	// Define the cluster and server by taking advantage of the current config file
	cluster, server, CA, err := custconfig.GetClusterServerOfCurrentContext()
	if err != nil {
		log.Println(err)
		return err
	}
	// Put the collected data into new kubeconfig file
	newKubeConfig := kubeconfigutil.CreateWithCerts(server, cluster, email, CA, clientKey, clientCert)
	newKubeConfig.Contexts[newKubeConfig.CurrentContext].Namespace = fmt.Sprintf("authority-%s", authority)
	kubeconfigutil.WriteToDisk(fmt.Sprintf("../../assets/kubeconfigs/edgenet-%s-%s.cfg", authority, username), newKubeConfig)
	// Check whether the creation process is completed
	_, err = ioutil.ReadFile(fmt.Sprintf("../../assets/kubeconfigs/edgenet-%s-%s.cfg", authority, username))
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
