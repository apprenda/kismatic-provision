[![Build Status](https://snap-ci.com/apprenda/kismatic-provision/branch/master/build_image)](https://snap-ci.com/apprenda/kismatic-provision/branch/master)

# Kismatic provision

Quickly build Kubernetes development, test and demo clusters on AWS and [Packet](https://packet.net) (other provisioners coming!)

[Mac & AWS] (docs/macaws.md)

[Linux & AWS] (docs/linuxaws.md)

[Mac & Packet] (docs/macpacket.md)

[Linux & Packet] (docs/linuxpacket.md)

[Mac & Vagrant] (docs/macvagrant.md)

# Download

Extract to the same location as kismatic.

[Download latest executable (OSX)](https://kismatic-installer.s3-accelerate.amazonaws.com/latest-darwin/provision)

`wget -O provision https://kismatic-installer.s3-accelerate.amazonaws.com/latest-darwin/provision`

`chmod +x provision`

[Download latest executable (Linux)](https://kismatic-installer.s3-accelerate.amazonaws.com/latest/provision)

`curl -L https://kismatic-installer.s3-accelerate.amazonaws.com/latest/provision`

`chmod +x provision`

# How to use with AWS

Set environment variables:

* **AWS_ACCESS_KEY_ID**: Your AWS access key, required for all operations
* **AWS_SECRET_ACCESS_KEY**: Your AWS secret key, required for all operations

Your user will need access to create EC2 instances, as well as access to create VPCs and other 
networking objects if you want these to be provisioned for you.

`provision aws create-minikube -f`

to create infrastructure for a minikube (single machine instance) along with a kismatic "plan" 
file. The -f flag forces the creation of a new VPC with wide open security.

`provision aws create -f -e 3 -m 2 -w 5`

to create infrastructure for a 3 node etcd, 2 master node and 5 worker node cluster, along with 
a kismatic "plan" file identifying these resources. Again, -f forces the creation of a new VPC.

`provision aws delete-all`

to delete all of the instances that have been created by Kismatic Provision and from the host you
run the command from. Any created VPCs or other networking objects will not be cleaned and will
be reused by future kismatic provision runs.

## Building a more secure cluster

The -f flag should not be used to construct clusters for production workloads -- it uses security
groups that are wide open. Kismatic will not alter your existing networking

You can build your own security group for infrastructure, opening whatever ports you may need plus
an ssh port for kismatic to use for the provisioning of your cluster.

You will need to specify environment variables for this SG and also for the corresponding subnet.

*  **AWS_SUBNET_ID**: The ID of a subnet to try to place machines into. If this environment variable exists,
                      it must be a real subnet in the us-east-1 region or all commands will fail.
*  **AWS_SECURITY_GROUP_ID**: The ID of a security group to place all new machines in. Must be a part of the
                              above subnet or commands will fail.
*  **AWS_KEY_NAME**: The name of a Keypair in AWS to be used to create machines. If empty, we will attempt
                     to use a key named `kismatic-integration-testing` and fail if it does not exist.
*  **AWS_SSH_KEY_PATH**: The absolute path to the private key associated with the Key Name above. If left blank,
                    we will attempt to use a key named 'kismaticuser.key' in the same directory as the
		    provision tool. This key is important as part of provisioning is ensuring that your
		    instance is online and is able to be reached via SSH.

# How to use with Packet

Required environment variables:

* **PACKET_API_KEY**: Your Packet.net API key, required for all operations
* **PACKET_PROJECT_ID**: The ID of the project where machines will be provisioned

Optional:

* ** PACKET_SSH_KEY_PATH**: The path to the SSH private key to be used for accessing the machines. If empty, 
			    a file called `kismatic-packet.pem` in the current working directory is used as 
			    the SSH private key.

`provision packet create-minikube`

to create infrastructure for a minikube (single machine instance) on a Type 0 along with a kismatic "plan" 
file.

`provision aws create -e 3 -m 2 -w 5`

to create infrastructure for a 3 node etcd, 2 master node and 5 worker node cluster, along with 
a kismatic "plan" file identifying these resources.

`provision aws delete <hostname>`

to delete a Packet host by name. You will need to call once for every created node.

`provision aws delete --all`

to delete all of the instances in your packet project. I mean all of 'em, even ones NOT created by the provision tool! Use with caution!

# How to use with Openstack
DNS and appropriate security group for the resultuing cluster has to pre-exist in the target Openstack environment.
To start provisioning run:
`provision-cmd openstack create`

The command line accepts the following parameters:
`
Usage:
  provision openstack create [flags]

Flags:
      --admin-pass string         Admin password for the cluster. This password will be also used as root password when nodes are created
      --dns-ip string             IP of the DNS server 
      --domain string             Domain name (default "ket")
      --etcd-name string          ETCD node name pattern (default "ketautoetcd")
  -e, --etcdNodeCount uint16      Count of etcd nodes to produce. (default 1)
      --flavor string             Preferred Flavor
  -f, --force-provision           If present, generate anything needed to build a cluster including VPCs, keypairs, routes, subnets, & a very insecure security group.
      --image string              Preferred Image
      --ingress-ip string         Floating IP for the ingress server
      --master-name string        Master node name pattern (default "ketautomaster")
  -m, --masterdNodeCount uint16   Count of master nodes to produce. (default 1)
      --network string            Preferred Network
  -n, --noplan                    If present, foregoes generating a plan file in this directory referencing the newly created nodes
      --os-pass string            Openstack User Password
      --os-tenant string          Openstack Tenant ID 
      --os-url string             Openstack URL 
      --os-user string            Openstack User Name
      --sec-grp string            Preferred Security Group
      --ssh-file string           SSH File (default "/ket/kismaticuser.key")
      --ssh-user string           SSH User (default "kismaticuser")
  -s, --storage-cluster           Create a storage cluster from all Worker nodes.
      --suffix string             Domain suffix (default "local")
      --worker-name string        Worker node name pattern (default "ketautoworker")
  -w, --workerNodeCount uint16    Count of worker nodes to produce. (default 1)

`
If required parameters are not provided, the provisioner will launch an interactive wizard to collect the missing information

The provisioning process on Openstack starts with creating an installer node. This node downloads all the necessary installation packages, starts a local web server (runs in Docker) and launches the requested nodes for the Kubernetes cluster via Openstack APIs. As the cluster members come online and initialize, they report back to the installer node. Once all the requested nodes are up, the installer pushes the required ssh keys to each node and starts the installation of the Kubernetes cluster. At the end of the installation, the Ingress node is assigned a floating IP (if requested) to provide access to future workloads from the outside.

# Current limitations

1. AWS is imited to us-east-1 region. (Packet has no such restriction)
2. CentOS support requires a "subscription" to the AMI on the Amazon Marketplace. If you try to build CentOS nodes without first having clicked through the EULA, you will receive an error with a URL you will need to visit on AWS. This happens once per account.
3. Master nodes are not properly load balanced.
4. The first Worker node is called out as an Ingress node in generated plan files. You can remove this if you don't have a need for Ingress.
