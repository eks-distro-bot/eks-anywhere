---
title: "vSphere configuration reference"
linkTitle: "vSphere"
weight: 10
description: >
  Full EKS-A configuration reference for a VMware vSphere cluster.
---

This is a generic template with detailed descriptions below for reference

```yaml
apiVersion: anywhere.eks.amazonaws.com/v1alpha1
kind: Cluster
metadata:
   name: my-cluster-name
spec:
   clusterNetwork:
      cni: "cilium"
      pods:
         cidrBlocks:
            - 192.168.0.0/16
      services:
         cidrBlocks:
            - 10.96.0.0/12
   controlPlaneConfiguration:
      count: 1
      endpoint:
         host: ""
      machineGroupRef:
         kind: VSphereMachineConfig
         name: my-cluster-machines
   datacenterRef:
      kind: VSphereDatacenterConfig
      name: my-cluster-datacenter
   externalEtcdConfiguration:
     count: 3
   kubernetesVersion: "1.21"
   workerNodeGroupConfigurations:
   - count: 1
      machineGroupRef:
         kind: VSphereMachineConfig
         name: my-cluster-machines

---
apiVersion: anywhere.eks.amazonaws.com/v1alpha1
kind: VSphereDatacenterConfig
metadata:
   name: my-cluster-datacenter
spec:
  datacenter: ""
  server: ""
  network: ""
  insecure:
  thumbprint: ""

---
apiVersion: anywhere.eks.amazonaws.com/v1alpha1
kind: VSphereMachineConfig
metadata:
   name: my-cluster-machines
spec:
  diskGiB:
  datastore: ""
  folder: ""
  numCPUs:
  memoryMiB:
  osFamily: ""
  resourcePool: ""
  storagePolicyName: ""
  template: ""
  users:
  - name: ""
    sshAuthorizedKeys:
    - ""
```

## Cluster Fields

The following additional optional configuration can also be included.

* [OIDC]({{< relref "oidc.md" >}})
* [etcd]({{< relref "etcd.md" >}})
* [proxy]({{< relref "proxy.md" >}})

### name
Name of your cluster `my-cluster-name` in this example

### clusterNetwork
Specific network configuration for your Kubernetes cluster.

### clusterNetwork.cni
CNI plugin to be installed in the cluster. The only supported value at the moment is `cilium`.

### clusterNetwork.pods.cidrBlocks[0]
Subnet used by pods in CIDR notation. Please note that only 1 custom pods CIDR block specification is permitted.

### clusterNetwork.services.cidrBlocks[0]
Subnet used by services in CIDR notation. Please note that only 1 custom services CIDR block specification is permitted.

### controlPlaneConfiguration
Specific control plane configuration for your Kubernetes cluster.

### controlPlaneConfiguration.count
Number of control plane nodes

### controlPlaneConfiguration.machineGroupRef
Refers to the Kubernetes object with vsphere specific configuration for your nodes. See `VSphereMachineConfig Fields` below.

### controlPlaneConfiguration.endpoint.host
A unique IP you want to use for the control plane VM in your EKS Anywhere cluster. Choose an IP in your networks
range that does not conflict with other VMs.

### workerNodeGroupsConfiguration
This takes in a list of node groups that you can define for your workers. Please note that at this time only 1 node group is permitted.

### workerNodeGroupsConfiguration[0].count
Number of worker nodes

### workerNodeGroupsConfiguration[0].machineGroupRef
Refers to the Kubernetes object with vsphere specific configuration for your nodes. See `VSphereMachineConfig Fields` below.

### datacenterRef
Refers to the Kubernetes object with vsphere environment specific configuration. See `VSphereDatacenterConfig Fields` below.

### kubernetesVersion
The Kubernetes version you want to use for your cluster. Supported values: `1.20`, `1.21`

## VSphereDatacenterConfig Fields

### datacenter
The vSphere datacenter to deploy the EKS Anywhere cluster on. For example `SDDC-Datacenter`.

### network
The VM network to deploy your EKS Anywhere cluster on.

### server
The vCenter server fully qualified domain name or IP address. If the server IP is used, the `thumbprint` must be set
or `insecure` must be set to true.

### insecure
Set insecure to `true` if the vCenter server does not have a valid certificate.

### thumbprint
The SHA1 thumbprint of the vCenter server certificate which is only required if you have a self signed certificate.

There are several ways to obtain your vCenter thumbprint. The easiest way is if you have `govc` installed, you
can run:

```
govc about.cert -thumbprint -k
```

Another way is from the vCenter web UI, go to Administration/Certificate Management and click view details of the
machine certificate. The format of this thumbprint does not exactly match the format required though and you will
need to add `:` to separate each hexadecimal value.

Another way to get the thumbprint is use this command with your servers certificate in a file named `ca.crt`:

```
openssl x509 -sha1 -fingerprint -in ca.crt -noout
```

If you specify the wrong thumbprint, an error message will be printed with the expected thumbprint. If no valid
certificate is being used, `insecure` must be set to true.


## VSphereMachineConfig Fields

### memoryMiB
Size of RAM on virtual machines (optional) (Default: 8192)

### numCPUs
Number of CPUs on virtual machines (optional) (Default: 2)

### osFamily
Operating System on virtual machines (optional) (Default: ubuntu). Permitted values: ubuntu, bottlerocket

### diskGiB
Size of disk on virtual machines if snapshots aren't included (optional) (Default: 25)

### users
The users you want to configure to access your virtual machines. Only one is permitted at this time

### users[0].name
The name of the user you want to configure to access your virtual machines through ssh.

### users[0].sshAuthorizedKeys
The SSH public keys you want to configure to access your virtual machines through ssh (as described below). Only 1 is supported at this time.

### users[0].sshAuthorizedKeys[0]
This is the SSH public key that will be placed in `authorized_keys` on all EKS Anywhere cluster VMs so you can ssh into
them. The user will be what is defined under name above. For example:

```
ssh -i <private-key-file> <user>@<VM-IP>
```

### template
The VM template to use for your EKS Anywhere cluster. This template was created when you
[imported the OVA file into vSphere](#import-an-ovaovf-template-to-vsphere).
This is a required field if you are using Bottlerocket OVAs.

### datastore
The vSphere [datastore](https://docs.vmware.com/en/VMware-vSphere/7.0/com.vmware.vsphere.storage.doc/GUID-3CC7078E-9C30-402C-B2E1-2542BEE67E8F.html)
to deploy your EKS Anywhere cluster on.

### folder
The VM folder for your EKS anywhere cluster VMs. This allows you to organize your VMs. If the folder does not exist,
it will be created for you. If the folder is blank, the VMs will go in the root folder.

### resourcePool
The vSphere [Resource pools](https://docs.vmware.com/en/VMware-vSphere/7.0/com.vmware.vsphere.resmgmt.doc/GUID-60077B40-66FF-4625-934A-641703ED7601.html)
for your VMs in the EKS Anywhere cluster. Examples of resource pool values include:

* If there is no resource pool: `/<datacenter>/host/<cluster-name>/Resources`
* If there is a resource pool:  `/<datacenter>/host/<cluster-name>/Resources/<resource-pool-name>`
* The wild card option `*/Resources` also often works.

### storagePolicyName
The storage policy name associated with your vms.