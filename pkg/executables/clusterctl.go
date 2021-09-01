package executables

import (
	"context"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/constants"
	"github.com/aws/eks-anywhere/pkg/filewriter"
	"github.com/aws/eks-anywhere/pkg/providers"
	"github.com/aws/eks-anywhere/pkg/templater"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/release/api/v1alpha1"
)

const (
	clusterCtlPath       = "clusterctl"
	clusterctlConfigFile = "clusterctl_tmp.yaml"
	capiPrefix           = "/generated/overrides"
)

//go:embed config/clusterctl.yaml
var clusterctlConfigTemplate string

type Clusterctl struct {
	executable Executable
	writer     filewriter.FileWriter
}

type clusterctlConfiguration struct {
	coreVersion              string
	bootstrapVersion         string
	controlPlaneVersion      string
	configFile               string
	etcdadmBootstrapVersion  string
	etcdadmControllerVersion string
}

func NewClusterctl(executable Executable, writer filewriter.FileWriter) *Clusterctl {
	return &Clusterctl{
		executable: executable,
		writer:     writer,
	}
}

func imageRepository(image v1alpha1.Image) string {
	return path.Dir(image.Image())
}

// This method will write the configuration files
// used by cluster api to install components.
// See: https://cluster-api.sigs.k8s.io/clusterctl/configuration.html
func buildOverridesLayer(clusterSpec *cluster.Spec, clusterName string, provider providers.Provider) error {
	bundle := clusterSpec.VersionsBundle

	// Adding cluster name to path temporarily following suggestion.
	//
	// This adds an implicit dependency between this method
	// and the writer passed to NewClusterctl
	// Ideally the writer implementation should be modified to
	// accept a path and file name and it should create the path in case it
	// does not exists.
	prefix := filepath.Join(clusterName, generatedDir, overridesDir)

	infraBundles := []types.InfrastructureBundle{
		{
			FolderName: filepath.Join("bootstrap-kubeadm", bundle.Bootstrap.Version),
			Manifests: []v1alpha1.Manifest{
				bundle.Bootstrap.Components,
				bundle.Bootstrap.Metadata,
			},
		},
		{
			FolderName: filepath.Join("cluster-api", bundle.ClusterAPI.Version),
			Manifests: []v1alpha1.Manifest{
				bundle.ClusterAPI.Components,
				bundle.ClusterAPI.Metadata,
			},
		},
		{
			FolderName: filepath.Join("control-plane-kubeadm", bundle.ControlPlane.Version),
			Manifests: []v1alpha1.Manifest{
				bundle.ControlPlane.Components,
				bundle.ControlPlane.Metadata,
			},
		},
	}

	if clusterSpec.Spec.ExternalEtcdConfiguration != nil {
		infraBundles = append(infraBundles, []types.InfrastructureBundle{
			{
				FolderName: filepath.Join("bootstrap-etcdadm-bootstrap", bundle.ExternalEtcdBootstrap.Version),
				Manifests: []v1alpha1.Manifest{
					bundle.ExternalEtcdBootstrap.Components,
					bundle.ExternalEtcdBootstrap.Metadata,
				},
			},
			{
				FolderName: filepath.Join("bootstrap-etcdadm-controller", bundle.ExternalEtcdController.Version),
				Manifests: []v1alpha1.Manifest{
					bundle.ExternalEtcdController.Components,
					bundle.ExternalEtcdController.Metadata,
				},
			},
		}...)
	}

	infraBundles = append(infraBundles, *provider.GetInfrastructureBundle(clusterSpec))
	for _, infraBundle := range infraBundles {
		if err := writeInfrastructureBundle(clusterSpec, prefix, &infraBundle); err != nil {
			return err
		}
	}

	return nil
}

func writeInfrastructureBundle(clusterSpec *cluster.Spec, rootFolder string, bundle *types.InfrastructureBundle) error {
	if bundle == nil {
		return nil
	}

	infraFolder := filepath.Join(rootFolder, bundle.FolderName)
	if err := os.MkdirAll(infraFolder, os.ModePerm); err != nil {
		return err
	}
	for _, manifest := range bundle.Manifests {
		m, err := clusterSpec.LoadManifest(manifest)
		if err != nil {
			return fmt.Errorf("can't load infrastructure bundle: %v", err)
		}

		if err := ioutil.WriteFile(filepath.Join(infraFolder, m.Filename), m.Content, 0o644); err != nil {
			return fmt.Errorf("error generating file for infrastructure bundle %s: %v", m.Filename, err)
		}
	}

	return nil
}

func (c *Clusterctl) MoveManagement(ctx context.Context, from, to *types.Cluster) error {
	params := []string{"move", "--to-kubeconfig", to.KubeconfigFile, "--namespace", constants.EksaSystemNamespace}
	if from.KubeconfigFile != "" {
		params = append(params, "--kubeconfig", from.KubeconfigFile)
	}
	_, err := c.executable.Execute(ctx, params...)
	if err != nil {
		return fmt.Errorf("failed moving management cluster: %v", err)
	}
	return err
}

func (c *Clusterctl) GetWorkloadKubeconfig(ctx context.Context, clusterName string, cluster *types.Cluster) ([]byte, error) {
	stdOut, err := c.executable.Execute(
		ctx, "get", "kubeconfig", clusterName,
		"--kubeconfig", cluster.KubeconfigFile,
		"--namespace", constants.EksaSystemNamespace,
	)
	if err != nil {
		return nil, fmt.Errorf("error executing get kubeconfig: %v", err)
	}
	return stdOut.Bytes(), nil
}

func (c *Clusterctl) InitInfrastructure(ctx context.Context, clusterSpec *cluster.Spec, cluster *types.Cluster, provider providers.Provider) error {
	if cluster == nil {
		return fmt.Errorf("invalid cluster (nil)")
	}
	if cluster.Name == "" {
		return fmt.Errorf("invalid cluster name '%s'", cluster.Name)
	}
	clusterctlConfig, err := c.buildConfig(clusterSpec, cluster.Name, provider)
	if err != nil {
		return err
	}

	params := []string{
		"init",
		"--core", clusterctlConfig.coreVersion,
		"--bootstrap", clusterctlConfig.bootstrapVersion,
		"--control-plane", clusterctlConfig.controlPlaneVersion,
		"--infrastructure", fmt.Sprintf("%s:%s", provider.Name(), provider.Version(clusterSpec)),
		"--config", clusterctlConfig.configFile,
	}
	if clusterSpec.Spec.ExternalEtcdConfiguration != nil {
		params = append(params, "--bootstrap", clusterctlConfig.etcdadmBootstrapVersion,
			"--bootstrap", clusterctlConfig.etcdadmControllerVersion)
	}
	if cluster.KubeconfigFile != "" {
		params = append(params, "--kubeconfig", cluster.KubeconfigFile)
	}

	envMap, err := provider.EnvMap()
	if err != nil {
		return err
	}

	_, err = c.executable.ExecuteWithEnv(ctx, envMap, params...)
	if err != nil {
		return fmt.Errorf("error executing init: %v", err)
	}

	return nil
}

func (c *Clusterctl) buildConfig(clusterSpec *cluster.Spec, clusterName string, provider providers.Provider) (*clusterctlConfiguration, error) {
	t := templater.New(c.writer)
	bundle := clusterSpec.VersionsBundle

	path, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	data := map[string]string{
		"CertManagerInjectorRepository":                   imageRepository(bundle.CertManager.Cainjector),
		"CertManagerInjectorTag":                          bundle.CertManager.Cainjector.Tag(),
		"CertManagerControllerRepository":                 imageRepository(bundle.CertManager.Controller),
		"CertManagerControllerTag":                        bundle.CertManager.Controller.Tag(),
		"CertManagerWebhookRepository":                    imageRepository(bundle.CertManager.Webhook),
		"CertManagerWebhookTag":                           bundle.CertManager.Webhook.Tag(),
		"ClusterApiControllerRepository":                  imageRepository(bundle.ClusterAPI.Controller),
		"ClusterApiControllerTag":                         bundle.ClusterAPI.Controller.Tag(),
		"ClusterApiKubeRbacProxyRepository":               imageRepository(bundle.ClusterAPI.KubeProxy),
		"ClusterApiKubeRbacProxyTag":                      bundle.ClusterAPI.KubeProxy.Tag(),
		"KubeadmBootstrapControllerRepository":            imageRepository(bundle.Bootstrap.Controller),
		"KubeadmBootstrapControllerTag":                   bundle.Bootstrap.Controller.Tag(),
		"KubeadmBootstrapKubeRbacProxyRepository":         imageRepository(bundle.Bootstrap.KubeProxy),
		"KubeadmBootstrapKubeRbacProxyTag":                bundle.Bootstrap.KubeProxy.Tag(),
		"KubeadmControlPlaneControllerRepository":         imageRepository(bundle.ControlPlane.Controller),
		"KubeadmControlPlaneControllerTag":                bundle.ControlPlane.Controller.Tag(),
		"KubeadmControlPlaneKubeRbacProxyRepository":      imageRepository(bundle.ControlPlane.KubeProxy),
		"KubeadmControlPlaneKubeRbacProxyTag":             bundle.ControlPlane.KubeProxy.Tag(),
		"ClusterApiAwsControllerRepository":               imageRepository(bundle.Aws.Controller),
		"ClusterApiAwsControllerTag":                      bundle.Aws.Controller.Tag(),
		"ClusterApiAwsKubeRbacProxyRepository":            imageRepository(bundle.Aws.KubeProxy),
		"ClusterApiAwsKubeRbacProxyTag":                   bundle.Aws.KubeProxy.Tag(),
		"ClusterApiVSphereControllerRepository":           imageRepository(bundle.VSphere.ClusterAPIController),
		"ClusterApiVSphereControllerTag":                  bundle.VSphere.ClusterAPIController.Tag(),
		"ClusterApiVSphereKubeRbacProxyRepository":        imageRepository(bundle.VSphere.KubeProxy),
		"ClusterApiVSphereKubeRbacProxyTag":               bundle.VSphere.KubeProxy.Tag(),
		"DockerKubeRbacProxyRepository":                   imageRepository(bundle.Docker.KubeProxy),
		"DockerKubeRbacProxyTag":                          bundle.Docker.KubeProxy.Tag(),
		"DockerManagerRepository":                         imageRepository(bundle.Docker.Manager),
		"DockerManagerTag":                                bundle.Docker.Manager.Tag(),
		"EtcdadmBootstrapProviderRepository":              imageRepository(bundle.ExternalEtcdBootstrap.Controller),
		"EtcdadmBootstrapProviderTag":                     bundle.ExternalEtcdBootstrap.Controller.Tag(),
		"EtcdadmBootstrapProviderKubeRbacProxyRepository": imageRepository(bundle.ExternalEtcdBootstrap.KubeProxy),
		"EtcdadmBootstrapProviderKubeRbacProxyTag":        bundle.ExternalEtcdBootstrap.KubeProxy.Tag(),
		"EtcdadmControllerRepository":                     imageRepository(bundle.ExternalEtcdController.Controller),
		"EtcdadmControllerTag":                            bundle.ExternalEtcdController.Controller.Tag(),
		"EtcdadmControllerKubeRbacProxyRepository":        imageRepository(bundle.ExternalEtcdController.KubeProxy),
		"EtcdadmControllerKubeRbacProxyTag":               bundle.ExternalEtcdController.KubeProxy.Tag(),
		"dir":                                             path + "/" + clusterName + capiPrefix,
	}

	filePath, err := t.WriteToFile(clusterctlConfigTemplate, data, clusterctlConfigFile)
	if err != nil {
		return nil, fmt.Errorf("error generating configuration file for clusterctl: %v", err)
	}
	if err := buildOverridesLayer(clusterSpec, clusterName, provider); err != nil {
		return nil, err
	}

	return &clusterctlConfiguration{
		configFile:               filePath,
		bootstrapVersion:         fmt.Sprintf("kubeadm:%s", bundle.Bootstrap.Version),
		controlPlaneVersion:      fmt.Sprintf("kubeadm:%s", bundle.ControlPlane.Version),
		coreVersion:              fmt.Sprintf("cluster-api:%s", bundle.ClusterAPI.Version),
		etcdadmBootstrapVersion:  fmt.Sprintf("etcdadm-bootstrap:%s", bundle.ExternalEtcdBootstrap.Version),
		etcdadmControllerVersion: fmt.Sprintf("etcdadm-controller:%s", bundle.ExternalEtcdController.Version),
	}, nil
}