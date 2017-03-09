package digitalocean

import (
	"bufio"
	//	"errors"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"strings"

	garbler "github.com/michaelbironneau/garbler/lib"
	"github.com/sashajeltuhin/kismatic-provision/provision/plan"
	"github.com/sashajeltuhin/kismatic-provision/provision/utils"
	"github.com/spf13/cobra"
)

type DOOpts struct {
	Token           string
	ClusterTag      string
	EtcdNodeCount   uint16
	MasterNodeCount uint16
	WorkerNodeCount uint16
	LeaveArtifacts  bool
	RunKismatic     bool
	NoPlan          bool
	ForceProvision  bool
	KeyPairName     string
	InstanceType    string
	Image           string
	Region          string
	Size            string
	Storage         bool
	SSHUser         string
	SshKeyPath      string
	SshKeyName      string
	SshPrivate      string
	SshPublic       string
	BootstrapCount  uint16
	RemoveKey       bool
}

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "do",
		Short: "Provision infrastructure on Digital Ocean.",
		Long:  `Provision infrastructure on Digital Ocean.`,
	}

	cmd.AddCommand(DOCreateCmd())
	cmd.AddCommand(DODeleteCmd())

	return cmd
}

func DOCreateCmd() *cobra.Command {
	opts := DOOpts{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Creates infrastructure for a new cluster. For now, only the US East region is supported.",
		Long: `Creates infrastructure for a new cluster. 
		
For now, only the US East region is supported.

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
	cmd.Flags().StringVarP(&opts.InstanceType, "instance-type", "i", "1gb", "Size of the instance. Current options: 1gb, 2gb, 4gb")
	cmd.Flags().StringVarP(&opts.Image, "image", "", "ubuntu-16-04-x64", "Name of the image to use")
	cmd.Flags().StringVarP(&opts.Region, "reg", "", "tor1", "Region to deploy to")
	cmd.Flags().StringVarP(&opts.Token, "token", "t", "", "Digital Ocean API token")
	cmd.Flags().StringVarP(&opts.ClusterTag, "clustertag", "", "apprenda", "TAG for all nodes in the cluster")
	cmd.Flags().StringVarP(&opts.SSHUser, "sshuser", "", "root", "SSH User name")
	cmd.Flags().StringVarP(&opts.SshKeyPath, "sshpath", "", "", "Path to the ssh key")
	cmd.Flags().StringVarP(&opts.SshKeyName, "sshfile", "", "cluster.pem", "ssh key name")
	cmd.Flags().Uint16VarP(&opts.BootstrapCount, "bootstrapCount", "", 1, "Number of bootstrap nodes to work with the cluster.")
	cmd.Flags().BoolVarP(&opts.Storage, "storage-cluster", "s", false, "Create a storage cluster from all Worker nodes.")

	return cmd
}

//func DOCreateMinikubeCmd() *cobra.Command {
//	opts := DOOpts{}
//	cmd := &cobra.Command{
//		Use:   "create-mini",
//		Short: "Creates infrastructure for a single-node instance. For now, only the US East region is supported.",
//		Long: `Creates infrastructure for a single-node instance.

//For now, only the US East region is supported.

//A smallish instance will be created with public IP addresses. The command will not return until the instance is online and accessible via SSH.`,
//		RunE: func(cmd *cobra.Command, args []string) error {
//			return makeInfraMinikube(opts)
//		},
//	}

//	cmd.Flags().StringVarP(&opts.OS, "operating-system", "o", "ubuntu", "Which flavor of Linux to provision. Try ubuntu, centos or rhel.")
//	cmd.Flags().BoolVarP(&opts.NoPlan, "noplan", "n", false, "If present, foregoes generating a plan file in this directory referencing the newly created nodes")
//	cmd.Flags().BoolVarP(&opts.ForceProvision, "force-provision", "f", false, "If present, generate anything needed to build a cluster including VPCs, keypairs, routes, subnets, & a very insecure security group.")
//	cmd.Flags().StringVarP(&opts.InstanceType, "instance-type-blueprint", "i", "small", "A blueprint of instance type(s). Current options: micro (all t2 micros), small (t2 micros, workers are t2.medium), beefy (M4.large and xlarge)")
//	cmd.Flags().BoolVarP(&opts.Storage, "storage-cluster", "s", false, "Create a storage cluster from all Worker nodes.")

//	return cmd
//}

func DODeleteCmd() *cobra.Command {
	opts := DOOpts{}
	cmd := &cobra.Command{
		Use:   "delete-all",
		Short: "Deletes all objects tagged as created by this machine with this tool. This will destroy clusters. Be ready.",
		Long: `Deletes all objects tagged as CreatedBy this machine and ProvisionedBy kismatic. 
		
This command destroys clusters.

It has no way of knowing that you had really important data on them. It is utterly remorseless.
		
Be ready.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteInfra(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Token, "token", "t", "", "Digital Ocean API token")
	cmd.Flags().StringVarP(&opts.ClusterTag, "clustertag", "", "apprenda", "TAG for all nodes in the cluster")
	cmd.Flags().BoolVarP(&opts.RemoveKey, "remove-key", "", true, "Inidicator whether the ssh key used for the cluster should be deleted")

	return cmd
}

// An Error made up of many contributing errors that all have equal weight (e.g. do not form a stack)
type CompositeError struct {
	e []error
}

func (c CompositeError) Error() string {
	ret := ""
	for _, e := range c.e {
		ret = ret + fmt.Sprintf(" - %v\n", e)
	}
	return ret
}

func (c *CompositeError) add(woe error) {
	c.e = append(c.e, woe)
}

func (c *CompositeError) merge(c2 CompositeError) {
	for _, e := range c2.e {
		c.e = append(c.e, e)
	}
}

func (c *CompositeError) hasError() bool {
	return len(c.e) > 0
}

func deleteInfra(opts DOOpts) error {
	reader := bufio.NewReader(os.Stdin)
	if opts.Token == "" {
		fmt.Print("Enter Digital Ocean API Token: ")
		url, _ := reader.ReadString('\n')
		opts.Token = strings.Trim(url, "\n")
	}

	provisioner, _ := GetProvisioner()

	return provisioner.TerminateNodes(opts)
}

func validateKeyFile(opts DOOpts) (string, string, error) {
	var filePath string

	if opts.SshKeyPath == "" {
		//try ssh dir next to the executable
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			fmt.Println("Cannot get path to exec", err)
		}
		opts.SshKeyPath = filepath.Join(dir, "ssh/")
		fmt.Println("Trying to locate key in ssh/ folder", opts.SshKeyPath)
	}

	filePath = filepath.Join(opts.SshKeyPath, opts.SshKeyName)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return "", "", fmt.Errorf("Private SSH file was not found in expected location. Create your own key pair and reference in options to the provision command. Change file permissions to allow w/r for the user (chmod 600) %v", err)
	}

	//fmt.Println("SSH File mode", s.Mode().Perm())
	//	if s.Mode().Perm()&0044 != 0000 {

	//		return "", "", fmt.Errorf("Set permissions of %v to 0600", filePath)
	//	}

	return filePath, filePath + ".pub", nil
}

func makeInfra(opts DOOpts) error {
	reader := bufio.NewReader(os.Stdin)
	if opts.Token == "" {
		fmt.Print("Enter Digital Ocean API Token: \n")
		url, _ := reader.ReadString('\n')
		opts.Token = strings.Trim(url, "\n")
		opts.Token = strings.Replace(opts.Token, "\r", "", -1) //for Windows
	}
	sshPrivate, sshPublic, errkey := validateKeyFile(opts)
	opts.SshPrivate = sshPrivate
	opts.SshPublic = sshPublic
	if errkey != nil {
		return errkey
	}

	fmt.Print("Provisioning\n")
	provisioner, _ := GetProvisioner()
	nodes, err := provisioner.ProvisionNodes(opts, NodeCount{
		Etcd:     opts.EtcdNodeCount,
		Worker:   opts.WorkerNodeCount,
		Master:   opts.MasterNodeCount,
		Boostrap: opts.BootstrapCount,
	})

	if err != nil {
		return err
	}

	fmt.Print("Waiting for SSH\n")
	if err = WaitForSSH(nodes, opts.SshPrivate); err != nil {
		return err
	}

	if opts.NoPlan {
		fmt.Println("Your instances are ready.\n")
		printNodes(&nodes)
	} else {
		storageNodes := []plan.Node{}
		if opts.Storage {
			storageNodes = nodes.Worker
		}
		remoteYaml := fmt.Sprintf("/ket/ssh/%s", opts.SshKeyName)
		return makePlan(&plan.Plan{
			AdminPassword:       generateAlphaNumericPassword(),
			Etcd:                nodes.Etcd,
			Master:              nodes.Master,
			Worker:              nodes.Worker,
			Ingress:             []plan.Node{nodes.Worker[0]},
			Storage:             storageNodes,
			MasterNodeFQDN:      nodes.Master[0].PublicIPv4,
			MasterNodeShortName: nodes.Master[0].PublicIPv4,
			SSHKeyFile:          remoteYaml,
			SSHUser:             nodes.Master[0].SSHUser,
		}, opts, nodes)
	}
	return nil
}

func makePlan(pln *plan.Plan, opts DOOpts, nodes ProvisionedNodes) error {
	template, err := template.New("planAWSOverlay").Parse(plan.OverlayNetworkPlan)
	if err != nil {
		return err
	}

	f, err := makeUniqueFile(0)

	if err != nil {
		return err
	}

	defer f.Close()
	w := bufio.NewWriter(f)

	if err = template.Execute(w, &pln); err != nil {
		return err
	}

	w.Flush()

	//scp plan file to bootstrap if requested
	if opts.BootstrapCount > 0 {
		boot := nodes.Boostrap[0]
		planPath, _ := filepath.Abs(f.Name())
		fmt.Println("File path:", planPath)
		out, scperr := scpFile(planPath, "/ket/kismatic-cluster.yaml", opts.SSHUser, boot.PublicIPv4, opts.SshPrivate)
		if scperr != nil {
			fmt.Errorf("Unable to push kismatic plan to boostrap node %v\n", scperr)
		} else {
			fmt.Println("Output:", out)
		}
	}
	fmt.Println("To install your cluster, run:")
	fmt.Println("./kismatic install apply -f " + f.Name())

	return nil
}

func makeUniqueFile(count int) (*os.File, error) {
	filename := "kismatic-cluster"
	if count > 0 {
		filename = filename + "-" + strconv.Itoa(count)
	}
	filename = filename + ".yaml"

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return os.Create(filename)
	} else {
		return makeUniqueFile(count + 1)
	}
}

func printNodes(nodes *ProvisionedNodes) {
	printRole("Etcd", &nodes.Etcd)
	printRole("Master", &nodes.Master)
	printRole("Worker", &nodes.Worker)
	printRole("Bootstrap", &nodes.Boostrap)
}

func printRole(title string, nodes *[]plan.Node) {
	fmt.Printf("%v:\n", title)
	for _, node := range *nodes {
		fmt.Printf("  %v (%v, %v)\n", node.ID, node.PublicIPv4, node.PrivateIPv4)
	}
}

func generateAlphaNumericPassword() string {
	attempts := 0
	for {
		reqs := &garbler.PasswordStrengthRequirements{
			MinimumTotalLength: 16,
			Uppercase:          rand.Intn(6),
			Digits:             rand.Intn(6),
			Punctuation:        -1, // disable punctuation
		}
		pass, err := garbler.NewPassword(reqs)
		if err != nil {
			return "weakpassword"
		}
		// validate that the library actually returned an alphanumeric password
		re := regexp.MustCompile("^[a-zA-Z1-9]+$")
		if re.MatchString(pass) {
			return pass
		}
		if attempts == 50 {
			return "weakpassword"
		}
		attempts++
	}
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
