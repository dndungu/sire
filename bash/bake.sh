#!/bin/bash

bake(){
	# hostnamectl set-hostname $(curl -s http://169.254.169.254/latest/meta-data/local-hostname)
	apt-get update && apt-get install -y apt-transport-https ca-certificates curl software-properties-common
	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
	add-apt-repository \
	  "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
	  $(lsb_release -cs) \
	  stable"
	apt-get update && apt-get install -y docker-ce
	cat > /etc/docker/daemon.json <<-EOF
	{
	  "exec-opts": ["native.cgroupdriver=systemd"],
	  "log-driver": "json-file",
	  "log-opts": {
	    "max-size": "100m"
	  },
	  "storage-driver": "overlay2"
	}
	EOF
	mkdir -p /etc/systemd/system/docker.service.d
	systemctl daemon-reload
	systemctl restart docker
	apt-get update && apt-get install -y apt-transport-https curl
	curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
	cat <<-EOF >/etc/apt/sources.list.d/kubernetes.list
	deb https://apt.kubernetes.io/ kubernetes-xenial main
	EOF
	apt-get update
	apt-get install -y kubelet kubeadm kubectl
	apt-mark hold kubelet kubeadm kubectl
	kubeadm config images pull
}

bake
