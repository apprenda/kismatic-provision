# Using Linux & AWS

## Prerequisites

### AWS Account

1. Make a new [AWS account](https://aws.amazon.com/free/) (you may re-use an existing one, but it's more likely you will run in to IAM issues)

2. You will need an `Access Key Id` and `Secret Access Key`. 
   * You can either [generate one for your root user](https://console.aws.amazon.com/iam/home?region=us-east-1#/security_credential) or build a user in IAM with full access to VPCs and EC2

### Download KET

In a terminal, run the following commands to download KET, and configure the environment
variables required for AWS. You will need your AWS Access Key ID and AWS Secret Access Key.

```
# Make a new directory for Kismatic and make it the working dir
mkdir ~/kismatic
cd ~/kismatic

# Download and unpack KET
wget -O - https://kismatic-installer.s3-accelerate.amazonaws.com/latest-darwin/kismatic.tar.gz | tar -zx

# Export AWS environment variables
export AWS_ACCESS_KEY_ID=YOURACCESSKEYID
export AWS_SECRET_ACCESS_KEY=YOURSECRETACCESSKEY
```

## Provision nodes and install cluster

```
# Create 3 instances in AWS inside a Kismatic VPC and create a Kismatic keypair
./provision aws create -f

# Run KET to install kubernetes
./kismatic install apply -f kismatic-cluster.yaml
```

## Destroy cluster

Remove any EC2 instances tagged as:
* created by kismatic AND
* created on this machine

```
./provision aws delete-all
```
