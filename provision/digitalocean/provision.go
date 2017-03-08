package digitalocean

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/sashajeltuhin/kismatic-provision/provision/plan"
)

const (
	SSHKEY = "kismatic-key"
)

type infrastructureProvisioner interface {
	ProvisionNodes(NodeCount, LinuxDistro) (ProvisionedNodes, error)

	TerminateNodes(ProvisionedNodes) error

	TerminateAllNodes() error

	//	ForceProvision() error

	SSHKey() string
}

type LinuxDistro string

type NodeCount struct {
	Etcd     uint16
	Master   uint16
	Worker   uint16
	Boostrap uint16
}

func (nc NodeCount) Total() uint16 {
	return nc.Etcd + nc.Master + nc.Worker
}

type ProvisionedNodes struct {
	Etcd     []plan.Node
	Master   []plan.Node
	Worker   []plan.Node
	Boostrap []plan.Node
}

func (p ProvisionedNodes) allNodes() []plan.Node {
	n := []plan.Node{}
	n = append(n, p.Etcd...)
	n = append(n, p.Master...)
	n = append(n, p.Worker...)
	n = append(n, p.Boostrap...)
	return n
}

type sshMachineProvisioner struct {
	sshKey string
}

func (p sshMachineProvisioner) SSHKey() string {
	return p.sshKey
}

type doProvisioner struct {
	sshMachineProvisioner
	client *Client
}

func GetProvisioner() (*doProvisioner, bool) {
	c := Client{}
	p := doProvisioner{client: &c}
	return &p, true
}

//func (p doProvisioner) TerminateAllNodes() error {
//	nodes, err := p.client.GetNodes()
//	if err != nil {
//		return err
//	}

//	if len(nodes) > 0 {
//		return p.client.DestroyNodes(nodes)
//	}
//	return nil
//}

func dropletToNode(drop *Droplet, opts *DOOpts) plan.Node {
	node := plan.Node{}
	node.ID = string(drop.ID)
	node.Host = drop.Name
	node.PublicIPv4 = drop.PublicIP
	node.PrivateIPv4 = drop.PrivateIP
	node.SSHUser = opts.SSHUser
	return node
}

func optionsToConfig(opts *DOOpts, name string) NodeConfig {
	config := NodeConfig{}
	config.Image = opts.Image
	config.Name = name
	config.Region = opts.Region
	config.PrivateNetworking = true
	config.Size = opts.InstanceType
	if opts.ClusterTag != "" {
		config.Tags = append(config.Tags, opts.ClusterTag)
	} else {
		config.Tags = append(config.Tags, "kismatic")
	}
	return config
}

func (p doProvisioner) ProvisionNodes(opts DOOpts, nodeCount NodeCount) (ProvisionedNodes, error) {
	provisioned := ProvisionedNodes{}
	keyconf := KeyConfig{}
	keyconf.Name = SSHKEY
	keyconf.PublicKeyFile = opts.SshPublic
	existing, _ := p.client.FindKeyByName(opts.Token, keyconf.Name)
	var key KeyConfig
	var errkey error
	if existing.Fingerprint != "" {
		key = existing
		fmt.Println("Using existing key", key)
	} else {
		fmt.Println("Creating new key")
		key, errkey = p.client.CreateKey(opts.Token, keyconf)
	}
	if errkey != nil {
		fmt.Println("Cannot create key", errkey)
		return provisioned, errkey
	}

	var dropletsETCD []Droplet
	var i uint16
	for i = 0; i < nodeCount.Etcd; i++ {
		config := optionsToConfig(&opts, fmt.Sprintf("Etcd-%d", i+1))
		drop, err := p.client.CreateNode(opts.Token, config, key)
		if err != nil {
			return provisioned, err
		}
		dropletsETCD = append(dropletsETCD, drop)
	}
	var dropletsMaster []Droplet
	for i = 0; i < nodeCount.Master; i++ {
		config := optionsToConfig(&opts, fmt.Sprintf("Master-%d", i+1))
		drop, err := p.client.CreateNode(opts.Token, config, key)
		if err != nil {
			return provisioned, err
		}
		dropletsMaster = append(dropletsMaster, drop)
	}
	var dropletsWorker []Droplet
	for i = 0; i < nodeCount.Worker; i++ {
		config := optionsToConfig(&opts, fmt.Sprintf("Worker-%d", i+1))
		drop, err := p.client.CreateNode(opts.Token, config, key)
		if err != nil {
			return provisioned, err
		}
		dropletsWorker = append(dropletsWorker, drop)
	}

	var dropletsBoot []Droplet
	for i = 0; i < nodeCount.Boostrap; i++ {
		config := optionsToConfig(&opts, fmt.Sprintf("Bootstrap-%d", i+1))
		drop, err := p.client.CreateNode(opts.Token, config, key)
		if err != nil {
			return provisioned, err
		}
		dropletsBoot = append(dropletsBoot, drop)
	}

	//Wait for assigned IPs

	for i = 0; i < nodeCount.Etcd; i++ {
		drop := p.WaitForIPs(opts, dropletsETCD[i])
		if drop != nil {
			n := dropletToNode(drop, &opts)
			provisioned.Etcd = append(provisioned.Etcd, n)
		} else {
			return provisioned, fmt.Errorf("Unable to get IPs from %s", dropletsETCD[i].Name)
		}
	}

	for i = 0; i < nodeCount.Master; i++ {
		drop := p.WaitForIPs(opts, dropletsMaster[i])
		if drop != nil {
			n := dropletToNode(drop, &opts)
			provisioned.Master = append(provisioned.Master, n)
		} else {
			return provisioned, fmt.Errorf("Unable to get IPs from %s", dropletsMaster[i].Name)
		}
	}

	for i = 0; i < nodeCount.Worker; i++ {
		drop := p.WaitForIPs(opts, dropletsWorker[i])
		if drop != nil {
			n := dropletToNode(drop, &opts)
			provisioned.Worker = append(provisioned.Worker, n)
		} else {
			return provisioned, fmt.Errorf("Unable to get IPs from %s", dropletsWorker[i].Name)
		}
	}

	for i = 0; i < nodeCount.Boostrap; i++ {
		drop := p.WaitForIPs(opts, dropletsBoot[i])
		if drop != nil {
			n := dropletToNode(drop, &opts)
			provisioned.Boostrap = append(provisioned.Boostrap, n)
		} else {
			return provisioned, fmt.Errorf("Unable to get IPs from %s", dropletsBoot[i].Name)
		}
	}

	fmt.Println("Done provisioning")
	return provisioned, nil
}

func (p doProvisioner) WaitForIPs(opts DOOpts, drop Droplet) *Droplet {
	fmt.Printf("Waiting for IPs to be assigned for node %s\n", drop.Name)
	for {
		init, err := p.client.GetDroplet(opts.Token, drop.ID)

		if init.PublicIP != "" && err == nil {
			// command succeeded
			fmt.Printf("IP assinged to %s: Public = %s ; Private %s\n", init.Name, init.PublicIP, init.PrivateIP)
			return &init
		}
		fmt.Printf(".")
		time.Sleep(3 * time.Second)
	}
}

func (p doProvisioner) TerminateNodes(opts DOOpts) error {

	key := ""
	if opts.RemoveKey {
		key = SSHKEY
	}

	return p.client.DeleteDropletsByTag(opts.Token, opts.ClusterTag, key)
}

func WaitForSSH(ProvisionedNodes ProvisionedNodes, sshKey string) error {
	nodes := ProvisionedNodes.allNodes()
	for _, n := range nodes {
		BlockUntilSSHOpen(n.Host, n.PublicIPv4, n.SSHUser, sshKey)
	}
	fmt.Println("SSH established on all nodes")
	//run commands on bootstrap node

	if len(ProvisionedNodes.Boostrap) > 0 {
		boot := ProvisionedNodes.Boostrap[0]
		fmt.Println("Exec commands on bootstrap node:", boot.Host, boot.PublicIPv4)
		cmd, errload := loadBootCmds()
		if errload != nil {
			fmt.Println("Cannot get path to exec", errload)
		}
		out, cmderr := ExecuteCmd(cmd, boot.PublicIPv4, boot.SSHUser, sshKey)
		fmt.Println("SSH command output:", out, cmderr)
	}
	return nil
}

func loadBootCmds() (string, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", fmt.Errorf("Cannot get path to exec %v\n", err)
	}
	cmdpath := filepath.Join(dir, "digitalocean/scripts/bootinit.sh")
	cmd, errcmd := ioutil.ReadFile(cmdpath)
	if errcmd != nil {
		fmt.Println("Cannot read public boot init file", errcmd)
		return "", errcmd
	}
	s := string(cmd)
	re := regexp.MustCompile(`\r?\n`)
	s = re.ReplaceAllString(s, " ")

	return s, nil
}
