/*
Copyright 2022 The Kubernetes Authors.

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

// Package kubeconfig contains a command to prints cluster kubeconfig
package kubeconfig

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/pki"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/completion"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
)

var errCurrentClusterNotFound = errors.New("current cluster is not configured")

type flagpole struct {
	Name                  string
	Host                  string
	InsecureSkipTLSVerify bool
	User                  string
	Groups                []string
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{
		Host:                  "127.0.0.1",
		User:                  pki.DefaultUser,
		Groups:                pki.DefaultGroups,
		InsecureSkipTLSVerify: false,
	}

	cmd := &cobra.Command{
		Args:              cobra.NoArgs,
		Use:               "kubeconfig",
		Short:             "Prints cluster kubeconfig",
		ValidArgsFunction: completion.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", flags.Host, "Override host[:port] for kubeconfig")
	cmd.Flags().BoolVar(&flags.InsecureSkipTLSVerify, "insecure-skip-tls-verify", flags.InsecureSkipTLSVerify, "Skip server certificate verification")
	cmd.Flags().StringVar(&flags.User, "user", flags.User, "Signing certificate with the specified user if modified")
	cmd.Flags().StringSliceVar(&flags.Groups, "group", flags.Groups, "Signing certificate with the specified groups if modified")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := flags.Name
	workdir := utilspath.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With(
		"cluster", flags.Name,
	)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster does not exist")
		}
		return err
	}

	kubeconfigPath := rt.GetWorkdirPath(runtime.InHostKubeconfigName)

	kubeConfig, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig file %s: %w", kubeconfigPath, err)
	}

	err = clientcmdapi.MinifyConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to minify kubeconfig file %s: %w", kubeconfigPath, err)
	}

	currentContext, clusterName, err := getCurrentCluster(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig file %s: %w", kubeconfigPath, err)
	}

	if flags.Host != "" {
		cluster := kubeConfig.Clusters[clusterName]
		host, err := modifyAddress(cluster.Server, flags.Host)
		if err != nil {
			return fmt.Errorf("failed to modify host %s: %w", cluster.Server, err)
		}
		kubeConfig.Clusters[clusterName].Server = host
	}

	if flags.InsecureSkipTLSVerify {
		cluster := kubeConfig.Clusters[clusterName]
		cluster.InsecureSkipTLSVerify = true
		cluster.CertificateAuthorityData = nil
	}

	userName := kubeConfig.Contexts[currentContext].AuthInfo

	if userName != "" && (!utilsslices.Equal(pki.DefaultGroups, flags.Groups) || flags.User != pki.DefaultUser) {
		// Load CA cert and key
		caCert, caKey, err := pki.ReadCertAndKey(rt.GetWorkdirPath(runtime.PkiName), "ca")
		if err != nil {
			return err
		}

		// Sign admin cert and key
		now := time.Now()
		notBefore := now.Add(-24 * time.Hour).UTC()
		notAfter := now.Add(pki.CertificateValidity).UTC()
		cert, key, err := pki.GenerateSignCert(flags.User, caCert, caKey, notBefore, notAfter, flags.Groups, nil)
		if err != nil {
			return fmt.Errorf("failed to generate admin cert and key: %w", err)
		}

		// Modify kubeconfig
		keyData, err := pki.EncodePrivateKeyToPEM(key)
		if err != nil {
			return err
		}
		certData := pki.EncodeCertToPEM(cert)
		kubeConfig.AuthInfos[userName].ClientCertificateData = certData
		kubeConfig.AuthInfos[userName].ClientCertificate = ""
		kubeConfig.AuthInfos[userName].ClientKeyData = keyData
		kubeConfig.AuthInfos[userName].ClientKey = ""
	}

	err = clientcmdapi.FlattenConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to flatten kubeconfig file %s: %w", kubeconfigPath, err)
	}

	// Encode kubeconfig
	kubeconfigData, err := kubeconfig.EncodeKubeconfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to encode kubeconfig: %w", err)
	}

	_, err = os.Stdout.Write(kubeconfigData)
	if err != nil {
		return err
	}

	return nil
}

func getCurrentCluster(kubeConfig *clientcmdapi.Config) (string, string, error) {
	currentContext := kubeConfig.CurrentContext
	context := kubeConfig.Contexts[currentContext]
	if context == nil {
		return "", "", errCurrentClusterNotFound
	}

	clusterName := context.Cluster
	if clusterName == "" {
		return "", "", errCurrentClusterNotFound
	}

	if kubeConfig.Clusters[clusterName] == nil {
		return "", "", errCurrentClusterNotFound
	}

	return currentContext, clusterName, nil
}

func modifyAddress(origin string, address string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return "", err
	}
	if _, _, err = net.SplitHostPort(address); err == nil {
		u.Host = address
	} else {
		_, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			return "", err
		}
		u.Host = net.JoinHostPort(trimIPv6Brackets(address), port)
	}
	return u.String(), nil
}

func trimIPv6Brackets(host string) string {
	return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
}
