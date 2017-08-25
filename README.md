# kismatic-provision

[![CircleCI](https://circleci.com/gh/apprenda/kismatic-provision.svg?style=svg)](https://circleci.com/gh/apprenda/kismatic-provision)

## Overview

Quickly build Kubernetes development, test and demo clusters on AWS, [Packet.net](https://packet.net) and Vagrant (other provisioners coming!).

## Download

The provisioner is included with [KET](https://github.com/apprenda/kismatic). If you
want to download the provisioner separately:

[Download latest executable (OSX)](https://kismatic-installer.s3-accelerate.amazonaws.com/latest-darwin/provision)

```
wget -O provision https://kismatic-installer.s3-accelerate.amazonaws.com/latest-darwin/provision
chmod +x provision
```

[Download latest executable (Linux)](https://kismatic-installer.s3-accelerate.amazonaws.com/latest/provision)

```
curl -O -L https://kismatic-installer.s3-accelerate.amazonaws.com/latest/provision
chmod +x provision
```

## AWS Quick Start

#### Set environment variables
* **AWS_ACCESS_KEY_ID**: Your AWS access key, required for all operations
* **AWS_SECRET_ACCESS_KEY**: Your AWS secret key, required for all operations

#### Create Minikube-style cluster

Create infrastructure for a minikube (single machine instance) along with a kismatic "plan" 
file. The -f flag forces the creation of a new VPC with wide open security.

```
./provision aws create-minikube -f
```

## Quick Start Guides
* [Mac & AWS](docs/macaws.md)
* [Linux & AWS](docs/linuxaws.md)
* [Mac & Packet](docs/macpacket.md)
* [Linux & Packet](docs/linuxpacket.md)
* [Mac & Vagrant](docs/macvagrant.md)

## Documentation
For more detailed documentation, see the [/docs directory](./docs).

## Current limitations

1. AWS is imited to us-east-1 region. (Packet has no such restriction)
2. CentOS support requires a "subscription" to the AMI on the Amazon Marketplace. If you try to build CentOS nodes without first having clicked through the EULA, you will receive an error with a URL you will need to visit on AWS. This happens once per account.
3. Master nodes are not properly load balanced.
4. The first Worker node is called out as an Ingress node in generated plan files. You can remove this if you don't have a need for Ingress.
