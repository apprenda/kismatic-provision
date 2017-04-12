package vagrant

import (
	"fmt"

	"github.com/apprenda/kismatic-provision/provision/utils"
	"github.com/spf13/cobra"
)

type VagrantCmdOpts struct {
	PlanOpts
	NoPlan                  bool
	OnlyGenerateVagrantfile bool
}

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vagrant",
		Short: "Provision a virtualized infrastructure using Vagrant.",
		Long:  `Provision a virtualized infrastructure using Vagrant.`,
	}

	cmd.AddCommand(VagrantCreateCmd())
	cmd.AddCommand(VagrantCreateMinikubeCmd())

	return cmd
}

func AddSharedFlags(cmd *cobra.Command, opts *VagrantCmdOpts) {
	//InfrastructureOps
	//(*cmd).Flags().StringVarP(&opts.NodeCIDR, "nodeCIDR", "c", "192.168.205.0/24", "Network CIDR to use in creating the VM Nodes")
	opts.NodeCIDR = "192.168.42.2/24"
	(*cmd).Flags().BoolVarP(&opts.Redhat, "useCentOS", "r", false, "If present, will install CentOS 7.3 rather than Ubuntu 16.04")
	// (*cmd).Flags().StringVarP(&opts.PrivateSSHKeyPath, "keypath", "k", "", "Path to private SSH key to use in provisioning VMs.")
	//(*cmd).Flags().StringVarP(&opts.Vagrantfile, "vagrantfile", "f", "Vagrantfile", "Path to Vagrantfile to generate")
	opts.Vagrantfile = "Vagrantfile"

	//PlanOpts
	// (*cmd).Flags().BoolVar(&opts.AllowPackageInstallation, "allowPackageInstallation", true, "If true, allows os packages to be installed automatically")
	opts.AllowPackageInstallation = true
	//(*cmd).Flags().BoolVar(&opts.AutoConfiguredDockerRegistry, "autoConfiguredDockerRegistry", true, "If true, installs a auto-configured Docker registry")
	opts.AutoConfiguredDockerRegistry = true
	// (*cmd).Flags().StringVar(&opts.DockerRegistryHost, "dockerRegistryIP", "", "IP or hostname for your Docker registry. An internal registry will NOT be setup when this field is provided. Must be accessible from all the nodes in the cluster.")
	// (*cmd).Flags().Uint16Var(&opts.DockerRegistryPort, "dockerRegistryPort", 443, "Port for your Docker registry")
	// (*cmd).Flags().StringVar(&opts.DockerRegistryCAPath, "dockerRegistryCAPath", "", "Absolute path to the CA that was used when starting your Docker registry. The docker daemons on all nodes in the cluster will be configured with this CA.")
	(*cmd).Flags().StringVar(&opts.AdminPassword, "adminPassword", utils.GenerateAlphaNumericPassword(), "This password is used to login to the Kubernetes Dashboard and can also be used for administration without a security certificate")
	opts.AdminPassword = utils.GenerateAlphaNumericPassword()
	// (*cmd).Flags().StringVar(&opts.PodCIDR, "podCIDR", "172.16.0.0/16", "Kubernetes will assign pods IPs in this range. Do not use a range that is already in use on your local network!")
	opts.PodCIDR = "172.16.0.0/16"
	// (*cmd).Flags().StringVar(&opts.ServiceCIDR, "serviceCIDR", "172.20.0.0/16", "Kubernetes will assign services IPs in this range. Do not use a range that is already in use by your local network or pod network!")
	opts.ServiceCIDR = "172.20.0.0/16"
	// VagrantCmdOpts
	// (*cmd).Flags().BoolVar(&opts.OnlyGenerateVagrantfile, "onlyGenerateVagrantFile", false, "If present, forgoes performing `vagrant up` on the generated Vagrantfile")
	(*cmd).Flags().BoolVar(&opts.NoPlan, "noplan", false, "If present, foregoes generating a plan file in this directory referencing the newly created nodes")
	(*cmd).Flags().BoolVarP(&opts.Storage, "storage-cluster", "s", false, "Create a storage cluster from all Worker nodes.")
}

func VagrantCreateCmd() *cobra.Command {
	var etcdCount, masterCount, workerCount, ingressCount uint16

	opts := VagrantCmdOpts{
		PlanOpts: PlanOpts{
			InfrastructureOpts: InfrastructureOpts{
				Count: map[NodeType]uint16{},
			},
		},
	}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Creates infrastructure for a new cluster.",
		Long: `Creates infrastructure for a new cluster.

Smallish instances will be created with public IP addresses. Unless option onlyGenerateVagrantfile is true, the command will not return 
until the instances are all online and accessible via SSH.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Count[Etcd] = etcdCount
			opts.Count[Master] = masterCount
			opts.Count[Worker] = workerCount
			opts.Count[Ingress] = ingressCount
			return makeInfrastructure(&opts)
		},
	}

	cmd.Flags().Uint16VarP(&etcdCount, "etcdNodeCount", "e", 1, "Count of etcd nodes to produce.")
	cmd.Flags().Uint16VarP(&masterCount, "masterdNodeCount", "m", 1, "Count of master nodes to produce.")
	cmd.Flags().Uint16VarP(&workerCount, "workerNodeCount", "w", 1, "Count of worker nodes to produce.")
	// cmd.Flags().Uint16VarP(&ingressCount, "ingressNodeCount", "i", 1, "Count of ingress nodes to produce")
	// cmd.Flags().BoolVar(&opts.OverlapRoles, "overlapRoles", false, "Overlap roles to create as few nodes as possible")

	AddSharedFlags(cmd, &opts)

	return cmd
}

func VagrantCreateMinikubeCmd() *cobra.Command {
	opts := VagrantCmdOpts{
		PlanOpts: PlanOpts{
			InfrastructureOpts: InfrastructureOpts{
				Count: map[NodeType]uint16{
					Etcd:    1,
					Master:  1,
					Worker:  1,
					Ingress: 1,
				},
				OverlapRoles: true,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "create-mini",
		Short: "Creates infrastructure for a single-node instance.",
		Long: `Creates infrastructure for a single-node instance. 

A smallish instance will be created with public IP addresses. The command will not return until the instance is online and accessible via SSH.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return makeInfrastructure(&opts)
		},
	}

	AddSharedFlags(cmd, &opts)

	return cmd
}

func makeInfrastructure(opts *VagrantCmdOpts) error {
	infrastructure, infraErr := NewInfrastructure(&opts.InfrastructureOpts)
	if infraErr != nil {
		return infraErr
	}

	_, vagrantErr := createVagrantfile(opts, infrastructure)
	if vagrantErr != nil {
		return vagrantErr
	}

	if opts.OnlyGenerateVagrantfile {
		fmt.Println("To create your local VMs, run:")
		fmt.Println("vagrant up")
	} else {
		if vagrantUpErr := vagrantUp(); vagrantUpErr != nil {
			return vagrantUpErr
		}
	}

	infrastructure.PrivateSSHKeyPath = grabSSHConfig()

	if !opts.NoPlan {
		planFile, planErr := createPlan(opts, infrastructure)
		if planErr != nil {
			return planErr
		}

		fmt.Println("To install your cluster, run:")
		fmt.Println("./kismatic install apply -f " + planFile)
	}

	return nil
}

func createVagrantfile(opts *VagrantCmdOpts, infrastructure *Infrastructure) (string, error) {
	vagrantfile, err := utils.MakeFileAskOnOverwrite("Vagrantfile")
	if err != nil {
		return "", err
	}

	defer vagrantfile.Close()

	vagrant := &Vagrant{
		Opts:           &opts.InfrastructureOpts,
		Infrastructure: infrastructure,
	}

	err = vagrant.Write(vagrantfile)
	if err != nil {
		return "", err
	}

	return vagrantfile.Name(), nil
}

func createPlan(opts *VagrantCmdOpts, infrastructure *Infrastructure) (string, error) {
	planFile, err := utils.MakeUniqueFile("kismatic-cluster", ".yaml", 0)
	if err != nil {
		return "", err
	}

	defer planFile.Close()

	plan := &Plan{
		Opts:           &opts.PlanOpts,
		Infrastructure: infrastructure,
	}

	err = plan.Write(planFile)
	if err != nil {
		return "", err
	}

	return planFile.Name(), nil
}
