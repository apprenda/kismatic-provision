package openstack

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/sashajeltuhin/ket/provision/openstack/utils"
	"github.com/spf13/cobra"
)

type KetOpts struct {
	EtcdNodeCount   uint16
	MasterNodeCount uint16
	WorkerNodeCount uint16
	EtcdName        string
	MasterName      string
	WorkerName      string
	LeaveArtifacts  bool
	RunKismatic     bool
	NoPlan          bool
	ForceProvision  bool
	KeyPairName     string
	InstanceType    string
	OS              string
	Storage         bool
	AdminPass       string
	SSHUser         string
	SSHFile         string
	Domain          string
	Suffix          string
	DNSip           string
	Image           string
	Flavor          string
	Network         string
	SecGroup        string
	OSUrl           string
	OSTenant        string
	OSUser          string
	OSUserPass      string
	IngressIP       string
}

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openstack",
		Short: "Provision infrastructure on Openstack.",
		Long:  `Provision infrastructure on Openstack.`,
	}

	cmd.AddCommand(CreateCmd())

	return cmd
}

func CreateCmd() *cobra.Command {
	opts := KetOpts{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Creates infrastructure for a new cluster.",
		Long: `Creates infrastructure for a new cluster. 
		

Smallish instances will be created with public IP addresses. The command will not return until the instances are all online and accessible via SSH.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return makeInfra(opts)
		},
	}

	cmd.Flags().Uint16VarP(&opts.EtcdNodeCount, "etcdNodeCount", "e", 1, "Count of etcd nodes to produce.")
	cmd.Flags().Uint16VarP(&opts.MasterNodeCount, "masterdNodeCount", "m", 1, "Count of master nodes to produce.")
	cmd.Flags().Uint16VarP(&opts.WorkerNodeCount, "workerNodeCount", "w", 1, "Count of worker nodes to produce.")
	cmd.Flags().BoolVarP(&opts.NoPlan, "noplan", "n", false, "If present, foregoes generating a plan file in this directory referencing the newly created nodes")
	cmd.Flags().BoolVarP(&opts.ForceProvision, "force-provision", "f", false, "If present, generate anything needed to build a cluster including VPCs, keypairs, routes, subnets, & a very insecure security group.")
	cmd.Flags().BoolVarP(&opts.Storage, "storage-cluster", "s", false, "Create a storage cluster from all Worker nodes.")
	cmd.Flags().StringVarP(&opts.AdminPass, "admin-pass", "", "", "Admin password for the cluster. This password will be also used as root password when nodes are created")
	cmd.Flags().StringVarP(&opts.SSHUser, "ssh-user", "", "kismaticuser", "SSH User")
	cmd.Flags().StringVarP(&opts.SSHFile, "ssh-file", "", "/ket/kismaticuser.key", "SSH File")
	cmd.Flags().StringVarP(&opts.Domain, "domain", "", "ket", "Domain name")
	cmd.Flags().StringVarP(&opts.Suffix, "suffix", "", "local", "Domain suffix")
	cmd.Flags().StringVarP(&opts.DNSip, "dns-ip", "", "", "Domain IP")
	cmd.Flags().StringVarP(&opts.Image, "image", "", "", "Preferred Image")
	cmd.Flags().StringVarP(&opts.Flavor, "flavor", "", "", "Preferred Flavor")
	cmd.Flags().StringVarP(&opts.Network, "network", "", "", "Preferred Network")
	cmd.Flags().StringVarP(&opts.SecGroup, "sec-grp", "", "", "Preferred Security Group")
	cmd.Flags().StringVarP(&opts.EtcdName, "etcd-name", "", "ketautoetcd", "ETCD node name pattern")
	cmd.Flags().StringVarP(&opts.MasterName, "master-name", "", "ketautomaster", "Master node name pattern")
	cmd.Flags().StringVarP(&opts.WorkerName, "worker-name", "", "ketautoworker", "Worker node name pattern")
	cmd.Flags().StringVarP(&opts.OSUrl, "os-url", "", "", "Openstack URL")
	cmd.Flags().StringVarP(&opts.OSTenant, "os-tenant", "", "", "Openstack Tenant ID")
	cmd.Flags().StringVarP(&opts.OSUser, "os-user", "", "", "Openstack User Name")
	cmd.Flags().StringVarP(&opts.OSUserPass, "os-pass", "", "", "Openstack User Password")
	cmd.Flags().StringVarP(&opts.IngressIP, "ingress-ip", "", "", "Floating IP for the ingress node")

	return cmd
}
func makeInfra(opts KetOpts) error {
	var conf Config
	var a Auth
	reader := bufio.NewReader(os.Stdin)
	if opts.OSUrl == "" {
		fmt.Print("Enter Openstack URL: ")
		url, _ := reader.ReadString('\n')
		opts.OSUrl = strings.Trim(url, "\n")
		fmt.Print("URL: ", opts.OSUrl)
	}
	if opts.OSTenant == "" {
		fmt.Print("Openstack Tenant ID: ")
		tenant, _ := reader.ReadString('\n') //"f4ec4723e8a541d68ef993b47ef75c94"
		a.Body.Tenant = strings.Trim(tenant, "\n")
	} else {
		a.Body.Tenant = opts.OSTenant
	}

	if opts.OSUser == "" {
		fmt.Print("Your user name: ")
		uname, _ := reader.ReadString('\n')
		a.Body.Credentials.Username = strings.Trim(uname, "\n")
	} else {
		a.Body.Credentials.Username = opts.OSUser
	}

	if opts.OSUserPass == "" {
		fmt.Print("Your password: ")
		pass, _ := gopass.GetPasswdMasked()
		a.Body.Credentials.Password = strings.Trim(string(pass), "\n")
	} else {
		a.Body.Credentials.Password = opts.OSUserPass
	}

	if opts.DNSip == "" {
		fmt.Print("Provide IP of the DNS: ")
		uname, _ := reader.ReadString('\n')
		opts.DNSip = strings.Trim(uname, "\n")
	}

	conf.Urlauth = fmt.Sprintf("%s:%s/", opts.OSUrl, KeystonePort)
	conf.Apiverauth = "v2.0"
	conf.Urlcomp = fmt.Sprintf("%s:%s/", opts.OSUrl, ComputePort)
	conf.Apivercomp = "v2"
	conf.Urlnet = fmt.Sprintf("%s:%s/", opts.OSUrl, NetworkPort)
	conf.Apivernet = "v2.0"

	if opts.IngressIP == "" {
		fmt.Print("Do you want to assign a floating IP to ingress node? (y, n)")
		answer, _ := reader.ReadString('\n')
		answer = strings.Trim(answer, "\n")
		if answer == "y" || answer == "yes" {
			ips, err := listFloatingIPs(a, conf)
			if err != nil {
				return errors.New(fmt.Sprintf("Cannot load images. %v. Provide your preferred image when calling the program", err))
			}
			fmt.Print("Select floating IP: \n")
			opts.IngressIP = askForInput(ips, reader)
		}
	}

	if opts.Image == "" {
		images, err := listImages(a, conf)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot load images. %v. Provide your preferred image when calling the program", err))
		}
		fmt.Print("Select Image: \n")
		opts.Image = askForInput(images, reader)
	}

	if opts.Flavor == "" {
		flavors, err := listFlavors(a, conf)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot load flavors. %v. Provide your preferred image when calling the program", err))
		}
		fmt.Print("Select Flavor: \n")
		opts.Flavor = askForInput(flavors, reader)
	}

	if opts.Network == "" {
		networks, err := listNetworks(a, conf)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot load networks. %v. Provide your preferred image when calling the program", err))
		}
		fmt.Print("Select Network: \n")
		opts.Network = askForInput(networks, reader)
	}

	if opts.SecGroup == "" {
		secgroups, err := listSecGroups(a, conf)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot load sec groups. %v. Provide your preferred image when calling the program", err))
		}
		fmt.Print("Select Security Group: \n")
		opts.SecGroup = askForInput(secgroups, reader)
	}

	if opts.AdminPass == "" {
		fmt.Print("Set Admin password for the cluster: ")
		pass, _ := gopass.GetPasswdMasked()
		opts.AdminPass = strings.Trim(string(pass), "\n")
	}

	server := buildNodeData("ketautoinstall", opts)
	_, err := buildNode(a, conf, server, opts, "install", "")

	if err != nil {
		fmt.Print("Error instantiating Openstack client", err)
		return err
	}

	fmt.Printf("Provisioning on Openstack at %s started", opts.OSUrl)

	return nil
}

func askForInput(objList map[string]string, reader *bufio.Reader) string {
	arrPairs := utils.SortMapbyVal(objList)
	count := len(objList)
	var arr = make([]string, count)
	for i := 0; i < count; i++ {
		arr[i] = arrPairs[i].Key
		fmt.Printf("%d - %s\n", i+1, arrPairs[i].Value)
	}

	objI, _ := reader.ReadString('\n')
	objIndex := strings.Trim(string(objI), "\n")
	index, _ := strconv.Atoi(objIndex)
	if index < 1 || index > len(objList) {
		fmt.Print("Invalid selection. Try again")
		return askForInput(objList, reader)
	} else {
		objID := arr[index-1]
		fmt.Println("You picked ", objList[objID])
		objID = strings.Trim(objID, "\"")
		return objID
	}
}
