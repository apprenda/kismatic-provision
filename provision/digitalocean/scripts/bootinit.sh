mkdir -p /ket &&
cd /ket &&
curl -L https://github.com/apprenda/kismatic/releases/download/v1.2.0/kismatic-v1.2.0-linux-amd64.tar.gz | tar -zx && 
sudo apt-get -y install git build-essential &&
sudo apt-get install -qq python2.7 && ln -s /usr/bin/python2.7 /usr/bin/python &&
git clone https://github.com/sashajeltuhin/kubernetes-workshop.git
