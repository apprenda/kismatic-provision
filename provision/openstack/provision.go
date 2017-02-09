package openstack

import (
	"bufio"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type KetBag struct {
	Auth      Auth
	Config    Config
	Opts      KetOpts
	Installer KetNode
}

type ProvisionedNodes struct {
	Etcd   []KetNode
	Master []KetNode
	Worker []KetNode
}

type NodesMeta struct {
	num  uint16
	name string
}

type CachedNodesMeta struct {
	Etcd   NodesMeta
	Master NodesMeta
	Worker NodesMeta
}

func GetClient(a Auth, conf Config) error {

	c := Client{}
	var err = c.getAPIClient(a, conf)
	return err
}

func prepNodeTemplate(auth Auth, conf Config, nodeData serverData, opts KetOpts, nodeType string, webIP string) (map[string]string, error) {
	var tokens map[string]string = make(map[string]string)
	bag := KetBag{Auth: auth, Config: conf, Opts: opts, Installer: KetNode{Host: nodeData.Server.Name}}

	jsonStr, parseErr := json.Marshal(bag)
	if parseErr != nil {
		return nil, parseErr
	}
	tokens["nodeName"] = nodeData.Server.Name
	tokens["webPort"] = "8013"
	tokens["dcip"] = opts.DNSip
	tokens["domainName"] = opts.Domain
	tokens["domainSuf"] = opts.Suffix
	tokens["rootPass"] = opts.AdminPass
	tokens["nodeType"] = nodeType
	if webIP != "" {
		tokens["webIP"] = webIP
	}
	tokens["postData"] = b64.StdEncoding.EncodeToString([]byte(jsonStr))

	return tokens, nil
}

func buildNodeData(name string, opts KetOpts) serverData {
	var server serverData
	server.Server.Name = name
	server.Server.ImageRef = opts.Image
	server.Server.FlavorRef = opts.Flavor
	var n network
	n.Uuid = opts.Network
	server.Server.Networks = append(server.Server.Networks, n)
	var sec secgroup
	sec.Name = opts.SecGroup
	server.Server.Security_groups = append(server.Server.Security_groups, sec)
	return server
}

func buildNode(auth Auth, conf Config, nodeData serverData, opts KetOpts, nodeType string, webIP string) (string, error) {
	c := Client{}
	if conf.InstallscriptURL == "" {
		switch nodeType {
		case "install":
			conf.InstallscriptURL = "https://raw.githubusercontent.com/sashajeltuhin/ket/master/provision/openstack/scripts/ketinstall.sh"
			break
		default:
			conf.InstallscriptURL = "https://raw.githubusercontent.com/sashajeltuhin/ket/master/provision/openstack/scripts/ketnode.sh"
			break
		}
	}
	var script, scriptErr = c.downloadInitScript(conf.InstallscriptURL)
	if scriptErr != nil {
		return "", fmt.Errorf("Error downloading script %v", scriptErr)
	}
	scriptRaw := string(script)
	tokens, parseErr := prepNodeTemplate(auth, conf, nodeData, opts, nodeType, webIP)
	if parseErr != nil {
		return "", parseErr
	}
	for key := range tokens {
		tokenized := "^^" + key + "^^"
		scriptRaw = strings.Replace(scriptRaw, tokenized, tokens[key], 1)
	}
	nodeData.Server.User_data = b64.StdEncoding.EncodeToString([]byte(scriptRaw))
	var nodeID, err = c.buildNode(auth, conf, nodeData, nodeType)
	if err != nil {
		return "", fmt.Errorf("Error spinning up node %s. Error: %v", nodeData.Server.Name, err)
	}
	return nodeID, err
}
func listImages(auth Auth, conf Config) (map[string]string, error) {
	c := Client{}
	return c.listImages(auth, conf)
}

func listFlavors(auth Auth, conf Config) (map[string]string, error) {
	c := Client{}
	return c.listFlavors(auth, conf)
}

func listNetworks(auth Auth, conf Config) (map[string]string, error) {
	c := Client{}
	return c.listNetworks(auth, conf)
}

func listSecGroups(auth Auth, conf Config) (map[string]string, error) {
	c := Client{}
	return c.listSecGroups(auth, conf)
}

func listFloatingIPs(auth Auth, conf Config) (map[string]string, error) {
	c := Client{}
	return c.listFloatingIPs(auth, conf)
}

func cacheNode(nodeName string, nodeIP string, bag KetBag) {
	//sshpass -p passwd ssh-copy-id -i /ket/kismaticuser.key.pub -o StrictHostKeyChecking=no kismaticuser@nodeIP
	args := []string{"-p", bag.Opts.AdminPass, "ssh-copy-id", "-i", "/ket/kismaticuser.key.pub", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("kismaticuser@%s", nodeIP)}
	out, err := exec.Command("sshpass", args...).Output()
	if err != nil {
		log.Println("Error pushing ssh cert", out, err)
	}

	cached := KetNode{ID: nodeName, Host: nodeName, PrivateIPv4: nodeIP, PublicIPv4: nodeIP, SSHUser: bag.Opts.SSHUser}
	cachedJson, _ := json.Marshal(cached)
	errwrite := ioutil.WriteFile(nodeName, cachedJson, 0644)
	if errwrite != nil {
		log.Printf("Issues serializing cachedJson file %v\n", errwrite)
	}
}

func checkIfStartKetInstall(bag KetBag) {
	var check bool = false
	var nodeMeta CachedNodesMeta
	nodeMeta.Etcd = NodesMeta{num: bag.Opts.EtcdNodeCount, name: bag.Opts.EtcdName}
	nodeMeta.Master = NodesMeta{num: bag.Opts.MasterNodeCount, name: bag.Opts.MasterName}
	nodeMeta.Worker = NodesMeta{num: bag.Opts.WorkerNodeCount, name: bag.Opts.WorkerName}
	check = isCacheComplete(nodeMeta)
	if check {
		nodes := ProvisionedNodes{}
		for i := 0; i < int(nodeMeta.Etcd.num); i++ {
			fileName := buildHostName(nodeMeta.Etcd.name, i)
			n, cacheErr := getCachedNode(fileName)
			if cacheErr != nil {
				log.Println("Cannot read data for node", fileName, cacheErr)
			}
			nodes.Etcd = append(nodes.Etcd, KetNode{ID: n.Host, Host: n.Host, PublicIPv4: n.PublicIPv4, PrivateIPv4: n.PrivateIPv4, SSHUser: bag.Opts.SSHUser})
		}

		for i := 0; i < int(nodeMeta.Master.num); i++ {
			fileName := buildHostName(nodeMeta.Master.name, i)
			n, cacheErr := getCachedNode(fileName)
			if cacheErr != nil {
				log.Println("Cannot read data for node", fileName, cacheErr)
			}
			nodes.Master = append(nodes.Master, KetNode{ID: n.Host, Host: n.Host, PublicIPv4: n.PublicIPv4, PrivateIPv4: n.PrivateIPv4, SSHUser: bag.Opts.SSHUser})
		}

		for i := 0; i < int(nodeMeta.Worker.num); i++ {
			fileName := buildHostName(nodeMeta.Worker.name, i)
			n, cacheErr := getCachedNode(fileName)
			if cacheErr != nil {
				log.Println("Cannot read data for node", fileName, cacheErr)
			}
			nodes.Worker = append(nodes.Worker, KetNode{ID: n.Host, Host: n.Host, PublicIPv4: n.PublicIPv4, PrivateIPv4: n.PrivateIPv4, SSHUser: bag.Opts.SSHUser})
		}

		//assign floating IP to the ingress, if requested
		if bag.Opts.IngressIP != "" {
			ingressNode, err := getCachedNode("ingress")
			if err != nil {
				log.Println("Cannot read ingress file", err)
			}
			c := Client{}
			errIP := c.assignFloatingIP(bag.Auth, bag.Config, ingressNode.ID, bag.Opts.IngressIP)
			if errIP != nil {
				log.Println("Error assigning floating ip to Ingress", errIP)
			}
		}

		startInstall(bag.Opts, nodes)
	}
}

func buildHostName(fileName string, i int) string {
	if i > 0 {
		fileName = fmt.Sprintf("%s_%s", fileName, i)
	}
	return fileName
}

func provisionKetNodes(bag KetBag, ip string) error {
	if ip == "" {
		return errors.New("To provision nodes valid IP of the installer node is required")
	}
	bag.Config.InstallscriptURL = ""

	for i := 0; i < int(bag.Opts.EtcdNodeCount); i++ {
		nodeName := buildHostName(bag.Opts.EtcdName, i)
		var _, erretcd = buildNode(bag.Auth, bag.Config, buildNodeData(nodeName, bag.Opts), bag.Opts, "etcd", ip)
		if erretcd != nil {
			log.Println("Error instantiating etcd node", erretcd)
			return erretcd
		}
	}

	for i := 0; i < int(bag.Opts.MasterNodeCount); i++ {
		nodeName := buildHostName(bag.Opts.MasterName, i)
		var _, errMaster = buildNode(bag.Auth, bag.Config, buildNodeData(nodeName, bag.Opts), bag.Opts, "master", ip)
		if errMaster != nil {
			log.Println("Error instantiating master node", errMaster)
			return errMaster
		}
	}

	for i := 0; i < int(bag.Opts.WorkerNodeCount); i++ {
		nodeName := buildHostName(bag.Opts.WorkerName, i)
		var nodeid, errWorker = buildNode(bag.Auth, bag.Config, buildNodeData(nodeName, bag.Opts), bag.Opts, "worker", ip)
		if errWorker != nil {
			log.Println("Error instantiating worker node", errWorker)
			return errWorker
		}
		//assume that ingress, if requested is on the first worker
		if i == 0 && bag.Opts.IngressIP != "" {
			nodeid = strings.Trim(nodeid, "\"")
			fmt.Println("Caching ingress node:", nodeName, nodeid)
			cached := KetNode{ID: nodeid}
			cachedJson, _ := json.Marshal(cached)
			errwrite := ioutil.WriteFile("ingress", cachedJson, 0644)
			if errwrite != nil {
				log.Printf("Issues serializing cachedJson file %v\n", errwrite)
			}
		}
	}
	return nil
}

func startInstall(opts KetOpts, nodes ProvisionedNodes) {
	storageNodes := []KetNode{}
	if opts.Storage {
		storageNodes = []KetNode{nodes.Worker[0]}
	}
	fileName, err := makePlan(&Plan{
		AdminPassword:       opts.AdminPass,
		Etcd:                nodes.Etcd,
		Master:              nodes.Master,
		Worker:              nodes.Worker,
		Ingress:             []KetNode{nodes.Worker[0]},
		Storage:             storageNodes,
		MasterNodeFQDN:      nodes.Master[0].Host,
		MasterNodeShortName: nodes.Master[0].Host,
		SSHKeyFile:          opts.SSHFile,
		SSHUser:             opts.SSHUser,
	})
	if err != nil {
		log.Printf("Error creating plan", err)
		return
	}

	//"/ket/kismatic install apply -f " + fileName
	errDir := os.Chdir("/ket")
	if errDir != nil {
		log.Println("Error switching to KET folder", errDir)
	}
	cmd := "/ket/kismatic"
	args := []string{"install", "apply", "-f", fileName}
	log.Println("Running KET install", cmd, args)
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		log.Println("Error installing Kismatic", string(out), err)
	}
	log.Println("Kismatic Install:", string(out))

}

func makePlan(pln *Plan) (string, error) {
	template, err := template.New("planOverlay").Parse(OverlayNetworkPlan)
	if err != nil {
		return "", err
	}

	f, err := makeUniqueFile()
	if err != nil {
		return "", err
	}

	defer f.Close()
	w := bufio.NewWriter(f)

	if err = template.Execute(w, &pln); err != nil {
		return "", err
	}

	w.Flush()

	return f.Name(), nil
}

func makeUniqueFile() (*os.File, error) {
	filename := "/ket/kismatic-cluster.yaml"

	return os.Create(filename)

}

func isCacheComplete(nodeMeta CachedNodesMeta) bool {
	//check etcd
	for i := 0; i < int(nodeMeta.Etcd.num); i++ {
		fileName := buildHostName(nodeMeta.Etcd.name, i)
		if doesFileExist(fileName) == false {
			return false
		}
	}
	for i := 0; i < int(nodeMeta.Master.num); i++ {
		fileName := buildHostName(nodeMeta.Master.name, i)
		if doesFileExist(fileName) == false {
			return false
		}
	}
	for i := 0; i < int(nodeMeta.Worker.num); i++ {
		fileName := buildHostName(nodeMeta.Worker.name, i)
		if doesFileExist(fileName) == false {
			return false
		}
	}

	log.Println("All nodes are in place")
	return true
}

func doesFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	if err != nil {
		log.Println("Node is not there yet", fileName)
		return false
	}
	return true
}

func getCachedNode(fileName string) (KetNode, error) {
	var node KetNode
	dat, readerr := ioutil.ReadFile(fileName)
	if readerr == nil && dat != nil && len(dat) > 0 {
		deserr := json.Unmarshal(dat, &node)
		return node, deserr
	}

	return node, readerr

}
