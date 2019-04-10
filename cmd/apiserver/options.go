/*
Copyright 2017 The Kubernetes Authors.

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

package server

import (
	"os"

	"github.com/spf13/pflag"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	genericserveroptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/klog"
)

const (
	// Store generated SSL certificates in a place that won't collide with the
	// k8s core API server.
	certDirectory = "/var/run/kubernetes-workflow-server"
)

// WorkflowServerOptions contains the aggregation of configuration structs for
// the workflow server. It contains everything needed to configure a basic API server.
// It is public so that integration tests can access it.
type WorkflowServerOptions struct {
	// the runtime configuration of our server
	GenericServerRunOptions *genericserveroptions.ServerRunOptions
	// the admission options
	AdmissionOptions *genericserveroptions.AdmissionOptions
	// the https configuration. certs, etc
	SecureServingOptions *genericserveroptions.SecureServingOptionsWithLoopback
	// authn for the API
	AuthenticationOptions *genericserveroptions.DelegatingAuthenticationOptions
	// authz for the API
	AuthorizationOptions *genericserveroptions.DelegatingAuthorizationOptions
	// EtcdOptions are options for serving with etcd as the backing store
	EtcdOptions *EtcdOptions
	// DisableAuth disables delegating authentication and authorization for testing scenarios
	DisableAuth bool
	// StandaloneMode if true asserts that we will not depend on a kube-apiserver
	StandaloneMode bool
	// whether or not to serve the OpenAPI spec (at /swagger.json)
	ServeOpenAPISpec bool
	// KubeconfigPath, if specified, is used over the in-cluster service account token.
	KubeconfigPath string
}

// NewWorkflowServerOptions creates a new instances of
// WorkflowServerOptions with all sub-options filled in.
func NewWorkflowServerOptions() *WorkflowServerOptions {
	opts := &WorkflowServerOptions{
		GenericServerRunOptions: genericserveroptions.NewServerRunOptions(),
		AdmissionOptions:        genericserveroptions.NewAdmissionOptions(),
		SecureServingOptions:    genericserveroptions.NewSecureServingOptions().WithLoopback(),
		AuthenticationOptions:   genericserveroptions.NewDelegatingAuthenticationOptions(),
		AuthorizationOptions:    genericserveroptions.NewDelegatingAuthorizationOptions(),
		EtcdOptions:             NewEtcdOptions(),
		StandaloneMode:          standaloneMode(),
	}
	// register all admission plugins
	registerAllAdmissionPlugins(opts.AdmissionOptions.Plugins)
	// Set generated SSL cert path correctly
	opts.SecureServingOptions.ServerCert.CertDirectory = certDirectory
	return opts
}

// AddFlags adds to the flag set the flags to configure the API Server.
func (s *WorkflowServerOptions) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(
		&s.DisableAuth,
		"disable-auth",
		false,
		"Disable authentication and authorization for testing purposes",
	)

	flags.BoolVar(
		&s.ServeOpenAPISpec,
		"serve-openapi-spec",
		false,
		"Whether this API server should serve the OpenAPI spec (problematic with older versions of kubectl)",
	)
	flags.StringVar(
		&s.KubeconfigPath,
		"kubeconfig",
		"",
		"Path to kubeconfig to use over the in-cluster service account token",
	)

	s.GenericServerRunOptions.AddUniversalFlags(flags)
	s.AdmissionOptions.AddFlags(flags)
	s.SecureServingOptions.AddFlags(flags)
	s.AuthenticationOptions.AddFlags(flags)
	s.AuthorizationOptions.AddFlags(flags)
	s.EtcdOptions.addFlags(flags)
}

// Validate checks all subOptions flags have been set and that they
// have not been set in a conflictory manner.
func (s *WorkflowServerOptions) Validate() error {
	errors := []error{}
	// TODO uncomment after 1.8 rebase expecting
	// https://github.com/kubernetes/kubernetes/pull/50308/files
	// errors = append(errors, s.AdmissionOptions.Validate()...)
	errors = append(errors, s.SecureServingOptions.Validate()...)
	errors = append(errors, s.AuthenticationOptions.Validate()...)
	errors = append(errors, s.AuthorizationOptions.Validate()...)
	// etcd options
	etcdErrs := s.EtcdOptions.Validate()
	if len(etcdErrs) > 0 {
		klog.Errorln("Error validating etcd options, do you have `--etcd-servers localhost` set?")
	}
	errors = append(errors, etcdErrs...)
	// TODO add alternative storage validation
	// errors = append(errors, s.CRDOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

// standaloneMode returns true if the env var WORKFLOW_STANALONE=true
// If enabled, we will assume no integration with Kubernetes API server is performed.
// It is intended for testing purposes only.
func standaloneMode() bool {
	val := os.Getenv("WORKFLOW_STANDALONE")
	return val == "true"
}
