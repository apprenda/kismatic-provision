#!/bin/bash
rootPass="^^rootPass^^"
nodeName="^^nodeName^^"
dcip="^^dcip^^"
domainName="^^domainName^^"
domainSuf="^^domainSuf^^"
webPort="^^webPort^^"
postData="^^postData^^"
echo $postData
sed -i \"s/mirrorlist=https/mirrorlist=http/\" /etc/yum.repos.d/epel.repo
yum check-update
yum -y install wget libcgroup cifs-utils nano openssh-clients libcgroup-tools unzip iptables-services net-tools bind bind-utils
service cgconfig start
echo "root:$rootPass" | chpasswd
echo "Updating hosts file"
x=$(hostname -I)
eval ipval=($x)
ip=${ipval[0]}
echo "$ip $nodeName" >> /etc/hosts
hostnamectl set-hostname $serverName
echo "Updating sshd to allow root login via ssh"
sed -i 's/#\?\(RSAAuthentication\s*\).*$/\1 yes/' /etc/ssh/sshd_config
sed -i 's/#\?\(PermitRootLogin\s*\).*$/\1 yes/' /etc/ssh/sshd_config
sed -i 's/#\?\(PasswordAuthentication\s*\).*$/\1 yes/' /etc/ssh/sshd_config
service sshd restart
echo "Updating domain info in resolv.conf"
cat > /etc/resolv.conf << EOF
nameserver $dcip
search $domainName.$domainSuf
domain $domainName.$domainSuf
EOF
chattr +i /etc/resolv.conf
echo "Installing Docker"
tee /etc/yum.repos.d/docker.repo <<-'EOF'
[dockerrepo]
name=Docker Repository
baseurl=https://yum.dockerproject.org/repo/main/centos/7
enabled=1
gpgcheck=1
gpgkey=https://yum.dockerproject.org/gpg
EOF
yum install -y http://yum.dockerproject.org/repo/main/centos/7/Packages/docker-engine-1.11.1-1.el7.centos.x86_64.rpm
systemctl daemon-reload
systemctl start docker
systemctl enable docker

echo "Install git"
yum install -y git

echo "Install go"
cd /tmp
curl -LO https://storage.googleapis.com/golang/go1.7.linux-amd64.tar.gz
tar -C /usr/local -xvzf go1.7.linux-amd64.tar.gz
mkdir -p ~/webket/{bin,pkg,src}
touch /etc/profile.d/path.sh
echo "export PATH=$PATH:/usr/local/go/bin" > /etc/profile.d/path.sh
echo 'export GOBIN="$HOME/webket/bin"' >> ~/.bash_profile
echo 'export GOPATH="$HOME/webket/src"' >> ~/.bash_profile
source /etc/profile && source ~/.bash_profile
echo "Get and install KET orchestrator service"
go get github.com/sashajeltuhin/ket/provision/exec/provision-web
cd $GOPATH/src/github.com/sashajeltuhin/ket/provision/exec/provision-web
docker build -t sashaz/ketinstall .
docker build -t sashaz/ketinstall .
docker run -d --name ket -p 8013:8013 -v /ket:/ket sashaz/ketinstall
echo "Configure KET user and download KET"
useradd -d /home/kismaticuser -m kismaticuser
echo "kismaticuser:$rootPass" | chpasswd
echo "kismaticuser ALL = (root) NOPASSWD:ALL" | tee /etc/sudoers.d/kismaticuser
chmod 0440 /etc/sudoers.d/kismaticuser
curl https://kismatic-packages-rpm.s3-accelerate.amazonaws.com/kismatic.repo -o /etc/yum.repos.d/kismatic.repo
mkdir /ket
chmod -R 777 /ket
cd /ket
curl -L https://kismatic-installer.s3-accelerate.amazonaws.com/latest/kismatic.tar.gz | tar -zx
ssh-keygen -t rsa -b 4096 -f kismaticuser.key -P ""

curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x ./kubectl
mv ./kubectl /usr/local/bin/kubectl
#cp generated/kubeconfig -p $HOME/.kube/config

echo -e "server $dcip\nupdate add $nodeName.$domainName.$domainSuf 3600 A $ip\nsend\n" | nsupdate -v

echo "Post to its own web server"
wget http://$ip:$webPort/install?ip=$ip --post-data $postData -o /tmp/appscale.log
