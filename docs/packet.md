# Packet.net

## How to use with Packet

Required environment variables:

* **PACKET_API_KEY**: Your Packet.net API key, required for all operations
* **PACKET_PROJECT_ID**: The ID of the project where machines will be provisioned

Optional:

* **PACKET_SSH_KEY_PATH**: The path to the SSH private key to be used for accessing the machines. If empty, 
			    a file called `kismatic-packet.pem` in the current working directory is used as 
			    the SSH private key.

## Sample Commands
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