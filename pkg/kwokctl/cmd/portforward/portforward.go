/*
Package portforward provides commands for forwarding a local port to a port on a pod.
It supports different runtime environments such as binary, Docker, Podman, Nerdctl, and kind.
This is useful for accessing component ports that are not actively exposed when the cluster is created.
*/
package portforward

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name      string
	LocalPort string
	*internalversion.KwokctlConfiguration
}

// NewCommand returns a new cobra.Command for port-forward in a cluster
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Use:   "port-forward NAME LOCAL_PORT",
		Short: "Forward a local port to a port on a pod",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			flags.LocalPort = args[1]
			err := runE(cmd.Context(), flags, args)
			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}
	cmd.DisableFlagParsing = true
	return cmd
}

func runE(ctx context.Context, flags *flagpole, args []string) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster does not exist")
		}
		return err
	}

	switch flags.Options.Runtime {
	case consts.RuntimeTypeKind:
		return portForwardKind(ctx, name, flags.LocalPort, *logger)
	case consts.RuntimeTypeDocker, consts.RuntimeTypeKindPodman, consts.RuntimeTypeKindNerdctl:
		return portForwardContainer(ctx, name, flags.LocalPort, *logger)
	case consts.RuntimeTypeBinary:
		return portForwardBinary(name, flags.LocalPort, *logger)
	default:
		err := rt.KubectlInCluster(exec.WithStdIO(ctx), args...)
		if err != nil {
			return err
		}
	}

	return nil
}

func portForwardBinary(name string, localPort string, logger log.Logger) error {
	logger.Info("Starting TCP forward for binary runtime", "name", name, "port", localPort)
	cmd, err := exec.Command(context.TODO(), "nc", "-l", "-p", localPort, "-c", fmt.Sprintf("nc %s %s", "target-ip", localPort)) // Adjust target-ip and port as necessary
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func portForwardContainer(ctx context.Context, name string, localPort string, logger log.Logger) error {
	logger.Info("Starting TCP forward for container runtime", "name", name, "port", localPort)
	cmd, err := exec.Command(ctx, "docker", "exec", "-i", name, "nc", "localhost", localPort)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func portForwardKind(ctx context.Context, name string, localPort string, logger log.Logger) error {
	logger.Info("Converting to kubectl port-forward for kind runtime", "name", name, "port", localPort)
	cmd, err := exec.Command(ctx, "kubectl", "port-forward", name, fmt.Sprintf("%s:%s", localPort, localPort))
	if err != nil {
		return err
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd start: kubectl port-forward %s %s: %w", name, localPort, err)
	}

	err = cmd.Wait()
	if err != nil {
		if buf, ok := cmd.Stderr.(*bytes.Buffer); ok {
			return fmt.Errorf("cmd wait: kubectl port-forward %s %s: %w\n%s", name, localPort, err, buf.String())
		}
		return fmt.Errorf("cmd wait: kubectl port-forward %s %s: %w", name, localPort, err)
	}

	return nil
}
