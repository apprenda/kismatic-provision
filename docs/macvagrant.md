# Using Mac and Vagrant

## Envionment Setup

### Install Vagrant

**Managing virtual machines is REALLY HARD**. This is why we use Vagrant. However, it's also why Vagrant and Virtualbox are constantly racing each other with features, simplifications, defects and stabilization. It's strongly urged that you update both fairly often and be prepared to back off a version if it doesn't work for your environment.

1. Install a Vagrant compatible virtual-machine provider such as [VirtualBox](https://www.virtualbox.org/wiki/Downloads)
2. Install (Vagrant)[https://www.vagrantup.com/docs/installation/]

### Download KET

In a terminal, run the following commands to download KET.

```
# Make a new directory for Kismatic and make it the working dir
mkdir ~/kismatic
cd ~/kismatic
```

Browse to https://github.com/apprenda/kismatic/releases and get the latest darwin release URL

``` bash
# Download and unpack KET 
wget -O - <latest darwin release url> | tar -zx
```

## Make a new cluster

```
./provision vagrant create-mini
./kismatic install apply -f kismatic-cluster.yaml
```

## Tear it down when you're done with it

```
vagrant destroy --force
```
