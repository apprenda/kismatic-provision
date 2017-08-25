# Using a Mac & Packet

## Prerequisites

### Packet.net Setup

1. Make a [Packet account](https://app.packet.net/#/registration/) (or reuse an existing one)
2. Make a [new Project](https://app.packet.net/portal#/projects/new).
   * Projects are tied to a payment method, so you may choose to use an existing one, just be aware that you shouldn't use the convenient `./provision packet delete --all` to decommission a cluster as it may remove other instances you care about it.
3. Make (or re-use) a Packet API key: https://app.packet.net/portal#/api-keys

### Download KET

In a terminal, run the following commands to download KET, and configure the environment
variables required for Packet.net.

You need to grab your Packet API Key and project's ID. The project ID will be part of the URL when you browse your project, and look like this: ```12345678-1234-5678-90ab-cdef12345678```

```
# Make a new directory for Kismatic and make it the working dir
mkdir ~/kismatic
cd ~/kismatic

# Download and unpack KET
wget -O - https://kismatic-installer.s3-accelerate.amazonaws.com/latest-darwin/kismatic.tar.gz | tar -zx

# Setup Packet.net environment variables
export PACKET_API_KEY=YOURAPIKEY
export PACKET_PROJECT_ID=YOURPROJECTID
```

### SSH Key
#### If you don't have an SSH key associated with your Packet.net account:

1. Create an SSH Keypair and copy the public key:
```
# Create a new SSH keypair
ssh-keygen -t rsa -f kismatic-packet.pem -N ""

# Copy public key to clipboard
cat kismatic-packet.pem.pub | pbcopy
```

2. Upload the public key to Packet.net on their [new SSH key page](https://app.packet.net/portal#/ssh-keys/new)

3. Set the path to the SSH private key
```
export PACKET_SSH_KEY_PATH=$(PWD)/kismatic-packet.pem
```

#### If you already have an SSH Key associated with your Packet.net account:

Set the path to the SSH private key
```
export PACKET_SSH_KEY_PATH=ABSOLUTE_PATH_TO_SSH_KEY
```

## Make a new cluster

```
./provision packet create
./kismatic install apply -f kismatic-cluster.yaml
```

## Tear it down when you're done with it

```
./provision packet delete --all
```
