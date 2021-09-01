package workflows

import (
	"context"
	"fmt"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/clustermarshaller"
	"github.com/aws/eks-anywhere/pkg/filewriter"
	"github.com/aws/eks-anywhere/pkg/logger"
	"github.com/aws/eks-anywhere/pkg/providers"
	"github.com/aws/eks-anywhere/pkg/task"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/validations"
	"github.com/aws/eks-anywhere/pkg/workflows/interfaces"
)

type Create struct {
	bootstrapper   interfaces.Bootstrapper
	provider       providers.Provider
	clusterManager interfaces.ClusterManager
	addonManager   interfaces.AddonManager
	writer         filewriter.FileWriter
}

func NewCreate(bootstrapper interfaces.Bootstrapper, provider providers.Provider,
	clusterManager interfaces.ClusterManager, addonManager interfaces.AddonManager, writer filewriter.FileWriter) *Create {
	return &Create{
		bootstrapper:   bootstrapper,
		provider:       provider,
		clusterManager: clusterManager,
		addonManager:   addonManager,
		writer:         writer,
	}
}

func (c *Create) Run(ctx context.Context, clusterSpec *cluster.Spec, forceCleanup bool) error {
	if forceCleanup {
		if err := c.bootstrapper.DeleteBootstrapCluster(ctx, &types.Cluster{
			Name: clusterSpec.Name,
		}, false); err != nil {
			return err
		}
	}
	commandContext := &task.CommandContext{
		Bootstrapper:   c.bootstrapper,
		Provider:       c.provider,
		ClusterManager: c.clusterManager,
		AddonManager:   c.addonManager,
		ClusterSpec:    clusterSpec,
		Rollback:       false,
		Writer:         c.writer,
	}
	err := task.NewTaskRunner(&SetAndValidateTask{}).RunTask(ctx, commandContext)
	if err != nil {
		_ = commandContext.ClusterManager.SaveLogs(ctx, commandContext.BootstrapCluster)
	}
	return err
}

// Task related entities

type CreateBootStrapClusterTask struct{}

type DeleteKindClusterTask struct{}

type SetAndValidateTask struct{}

type CreateWorkloadClusterTask struct{}

type InstallEksaComponentsTask struct{}

type InstallAddonManagerTask struct{}

type MoveClusterManagementTask struct{}

type WriteClusterConfigTask struct{}

type DeleteBootstrapClusterTask struct{}

// CreateBootStrapClusterTask implementation

func (s *CreateBootStrapClusterTask) Run(ctx context.Context, commandContext *task.CommandContext) task.Task {
	logger.Info("Creating new bootstrap cluster")
	bootstrapOptions, err := commandContext.Provider.BootstrapClusterOpts()
	if err != nil {
		commandContext.SetError(err)
		return nil
	}

	bootstrapCluster, err := commandContext.Bootstrapper.CreateBootstrapCluster(ctx, commandContext.ClusterSpec, bootstrapOptions...)
	if err != nil {
		commandContext.SetError(err)
		return &DeleteKindClusterTask{}
	}
	commandContext.BootstrapCluster = bootstrapCluster

	logger.Info("Installing cluster-api providers on bootstrap cluster")
	err = commandContext.ClusterManager.InstallCapi(ctx, commandContext.ClusterSpec, bootstrapCluster, commandContext.Provider)
	if err != nil {
		commandContext.SetError(err)
		return &DeleteKindClusterTask{}
	}

	logger.Info("Provider specific setup")
	err = commandContext.Provider.BootstrapSetup(ctx, commandContext.ClusterSpec.Cluster, bootstrapCluster)
	if err != nil {
		commandContext.SetError(err)
		return &DeleteKindClusterTask{}
	}

	return &CreateWorkloadClusterTask{}
}

func (s *CreateBootStrapClusterTask) Name() string {
	return "bootstrap-cluster-init"
}

// DeleteKindClusterTask implementation

func (s *DeleteKindClusterTask) Run(ctx context.Context, commandContext *task.CommandContext) task.Task {
	if commandContext.BootstrapCluster != nil {
		if err := commandContext.Bootstrapper.DeleteBootstrapCluster(ctx, commandContext.BootstrapCluster, false); err != nil {
			commandContext.SetError(err)
		}
		return nil
	}
	logger.Info("Bootstrap cluster information missing - skipping delete kind cluster")
	return nil
}

func (s *DeleteKindClusterTask) Name() string {
	return "delete-kind-cluster"
}

// SetAndValidateTask implementation

func (s *SetAndValidateTask) Run(ctx context.Context, commandContext *task.CommandContext) task.Task {
	logger.Info("Performing setup and validations")
	runner := validations.NewRunner()
	runner.Register(s.providerValidation(ctx, commandContext)...)
	runner.Register(commandContext.AddonManager.Validations(ctx, commandContext.ClusterSpec)...)

	err := runner.Run()
	if err != nil {
		commandContext.SetError(err)
		return nil
	}
	return &CreateBootStrapClusterTask{}
}

func (s *SetAndValidateTask) providerValidation(ctx context.Context, commandContext *task.CommandContext) []validations.Validation {
	return []validations.Validation{
		func() *validations.ValidationResult {
			return &validations.ValidationResult{
				Name: fmt.Sprintf("%s Provider setup is valid", commandContext.Provider.Name()),
				Err:  commandContext.Provider.SetupAndValidateCreateCluster(ctx, commandContext.ClusterSpec),
			}
		},
	}
}

func (s *SetAndValidateTask) Name() string {
	return "setup-validate"
}

// CreateWorkloadClusterTask implementation

func (s *CreateWorkloadClusterTask) Run(ctx context.Context, commandContext *task.CommandContext) task.Task {
	logger.Info("Creating new workload cluster")
	workloadCluster, err := commandContext.ClusterManager.CreateWorkloadCluster(ctx, commandContext.BootstrapCluster, commandContext.ClusterSpec, commandContext.Provider)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}
	commandContext.WorkloadCluster = workloadCluster

	logger.Info("Installing networking on workload cluster")
	err = commandContext.ClusterManager.InstallNetworking(ctx, workloadCluster, commandContext.ClusterSpec)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}

	logger.Info("Installing storage class on workload cluster")
	err = commandContext.ClusterManager.InstallStorageClass(ctx, workloadCluster, commandContext.Provider)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}

	logger.Info("Installing cluster-api providers on workload cluster")
	err = commandContext.ClusterManager.InstallCapi(ctx, commandContext.ClusterSpec, commandContext.WorkloadCluster, commandContext.Provider)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}

	logger.V(4).Info("Installing machine health checks on bootstrap cluster")
	err = commandContext.ClusterManager.InstallMachineHealthChecks(ctx, commandContext.BootstrapCluster, commandContext.Provider)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}

	return &MoveClusterManagementTask{}
}

func (s *CreateWorkloadClusterTask) Name() string {
	return "workload-cluster-init"
}

// MoveClusterManagementTask implementation

func (s *MoveClusterManagementTask) Run(ctx context.Context, commandContext *task.CommandContext) task.Task {
	logger.Info("Moving cluster management from bootstrap to workload cluster")
	err := commandContext.ClusterManager.MoveCapi(ctx, commandContext.BootstrapCluster, commandContext.WorkloadCluster)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}

	return &InstallEksaComponentsTask{}
}

func (s *MoveClusterManagementTask) Name() string {
	return "capi-management-move"
}

// InstallEksaComponentsTask implementation

func (s *InstallEksaComponentsTask) Run(ctx context.Context, commandContext *task.CommandContext) task.Task {
	logger.Info("Installing EKS-A custom components (CRD and controller) on workload cluster")
	err := commandContext.ClusterManager.InstallCustomComponents(ctx, commandContext.ClusterSpec, commandContext.WorkloadCluster)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}

	logger.Info("Creating EKS-A CRDs instances on workload cluster")
	datacenterConfig := commandContext.Provider.DatacenterConfig()
	machineConfigs := commandContext.Provider.MachineConfigs()

	// this disables create-webhook validation during create
	commandContext.ClusterSpec.PauseReconcile()
	datacenterConfig.PauseReconcile()

	err = commandContext.ClusterManager.CreateEKSAResources(ctx, commandContext.WorkloadCluster, commandContext.ClusterSpec, datacenterConfig, machineConfigs)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}
	err = commandContext.ClusterManager.ResumeEKSAControllerReconcile(ctx, commandContext.WorkloadCluster, commandContext.ClusterSpec, commandContext.Provider)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}
	return &InstallAddonManagerTask{}
}

func (s *InstallEksaComponentsTask) Name() string {
	return "eksa-components-install"
}

// InstallAddonManagerTask implementation

func (s *InstallAddonManagerTask) Run(ctx context.Context, commandContext *task.CommandContext) task.Task {
	logger.Info("Installing AddonManager and GitOps Toolkit on workload cluster")

	err := commandContext.AddonManager.InstallGitOps(ctx, commandContext.WorkloadCluster, commandContext.ClusterSpec, commandContext.Provider.DatacenterConfig(), commandContext.Provider.MachineConfigs())
	if err != nil {
		logger.MarkFail("Error when installing GitOps toolkits on workload cluster; EKS-A will continue with cluster creation, but GitOps will not be enabled", "error", err)
		return &WriteClusterConfigTask{}
	}
	return &WriteClusterConfigTask{}
}

func (s *InstallAddonManagerTask) Name() string {
	return "addon-manager-install"
}

func (s *WriteClusterConfigTask) Run(ctx context.Context, commandContext *task.CommandContext) task.Task {
	logger.Info("Writing cluster config file")
	err := clustermarshaller.WriteClusterConfig(commandContext.ClusterSpec, commandContext.Provider.DatacenterConfig(), commandContext.Provider.MachineConfigs(), commandContext.Writer)
	if err != nil {
		commandContext.SetError(err)
	}
	return &DeleteBootstrapClusterTask{}
}

func (s *WriteClusterConfigTask) Name() string {
	return "write-cluster-config"
}

// DeleteBootstrapClusterTask implementation

func (s *DeleteBootstrapClusterTask) Run(ctx context.Context, commandContext *task.CommandContext) task.Task {
	logger.Info("Deleting bootstrap cluster")
	err := commandContext.Bootstrapper.DeleteBootstrapCluster(ctx, commandContext.BootstrapCluster, false)
	if err != nil {
		commandContext.SetError(err)
	}
	if commandContext.OriginalError == nil {
		logger.MarkSuccess("Cluster created!")
	}
	return nil
}

func (s *DeleteBootstrapClusterTask) Name() string {
	return "delete-kind-cluster"
}