package vagrant

import (
	"bufio"
	"html/template"
	"os"
)

type PlanOpts struct {
	InfrastructureOpts
	AllowPackageInstallation     bool
	AutoConfiguredDockerRegistry bool
	DockerRegistryHost           string
	DockerRegistryPort           uint16
	DockerRegistryCAPath         string
	AdminPassword                string
	PodCIDR                      string
	ServiceCIDR                  string
}

type Plan struct {
	Opts           *PlanOpts
	Infrastructure *Infrastructure
}

func (p *Plan) Write(file *os.File) error {
	template, err := template.New("planVagrantOverlay").Parse(planVagrantOverlay)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(file)

	if err = template.Execute(w, &p); err != nil {
		return err
	}

	w.Flush()

	return nil
}

func (p *Plan) Etcd() []NodeDetails {
	return p.Infrastructure.nodesByType(Etcd)
}

func (p *Plan) Master() []NodeDetails {
	return p.Infrastructure.nodesByType(Master)
}

func (p *Plan) Worker() []NodeDetails {
	return p.Infrastructure.nodesByType(Worker)
}

func (p *Plan) Ingress() []NodeDetails {
	return p.Infrastructure.nodesByType(Worker)[0:1]
}

func (p *Plan) Storage() []NodeDetails {
	if p.Opts.Storage {
		return p.Infrastructure.nodesByType(Worker)
	}
	return []NodeDetails{}
}

const planVagrantOverlay = `cluster:
  name: kubernetes
  admin_password: {{.Opts.AdminPassword}}      # This password is used to login to the Kubernetes Dashboard and can also be used for administration without a security certificate
  allow_package_installation: {{.Opts.AllowPackageInstallation}}      # When false, installation will not occur if any node is missing the correct deb/rpm packages. When true, the installer will attempt to install missing packages for you.
  disconnected_installation: false							# Set to true if you have local package and Docker repositories seeded with Kismatic binaries.
  networking:
    type: overlay                        # overlay or routed. Routed pods can be addressed from outside the Kubernetes cluster; Overlay pods can only address each other.
    pod_cidr_block: {{.Opts.PodCIDR}}        # Kubernetes will assign pods IPs in this range. Do not use a range that is already in use on your local network!
    service_cidr_block: {{.Opts.ServiceCIDR}}   # Kubernetes will assign services IPs in this range. Do not use a range that is already in use by your local network or pod network!
    policy_enabled: false                # When true, enables network policy enforcement on the Kubernetes Pod network. This is an advanced feature.
    update_hosts_files: true            # When true, the installer will add entries for all nodes to other nodes' hosts files. Use when you don't have access to DNS.
  certificates:
    expiry: 17520h                       # Self-signed certificate expiration period in hours; default is 2 years.
  ssh:
    user: vagrant
    ssh_key: {{.Infrastructure.PrivateSSHKeyPath}}             # Absolute path to the ssh public key we should use to manage nodes.
    ssh_port: 22
docker_registry:                         # Here you will provide the details of your Docker registry or setup an internal one to run in the cluster. This is optional and the cluster will always have access to the Docker Hub.
  setup_internal: {{.Opts.AutoConfiguredDockerRegistry}}                  # When true, a Docker Registry will be installed on top of your cluster and used to host Docker images needed for its installation.
  address: {{.Opts.DockerRegistryHost}}                            # IP or hostname for your Docker registry. An internal registry will NOT be setup when this field is provided. Must be accessible from all the nodes in the cluster.
  port: {{.Opts.DockerRegistryPort}}                              # Port for your Docker registry.
  CA: {{.Opts.DockerRegistryCAPath}}                                 # Absolute path to the CA that was used when starting your Docker registry. The docker daemons on all nodes in the cluster will be configured with this CA.
etcd:
  expected_count: {{len .Etcd}}
  nodes:{{range .Etcd}}
  - host: {{.Name}}
    ip: {{.IP.String}}{{end}}
master:
  expected_count: {{len .Master}}
  nodes:{{range .Master}}
  - host: {{.Name}}
    ip: {{.IP.String}}{{end}}
  load_balanced_fqdn: {{with index .Master 0 }}{{.IP.String}}{{end}}
  load_balanced_short_name: {{with index .Master 0}}{{.IP.String}}{{end}}
worker:
  expected_count: {{len .Worker}}
  nodes:{{range .Worker}}
  - host: {{.Name}}
    ip: {{.IP.String}}{{end}}
ingress:
  expected_count: {{len .Ingress}}
  nodes:{{range .Ingress}}
  - host: {{.Name}}
    ip: {{.IP.String}}{{end}}
storage:
  expected_count: {{len .Storage}}
  nodes:{{range .Storage}}
  - host: {{.Name}}
    ip: {{.IP.String}}{{end}}
`
