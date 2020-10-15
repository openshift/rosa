## moactl create cluster

Create cluster

### Synopsis

Create cluster.

```
moactl create cluster [flags]
```

### Examples

```
  # Create a cluster named "mycluster"
  moactl create cluster --cluster-name=mycluster

  # Create a cluster in the us-east-2 region
  moactl create cluster --cluster-name=mycluster --region=us-east-2
```

### Options

```
  -c, --cluster-name string           Name of the cluster. This will be used when generating a sub-domain for your cluster on openshiftapps.com.
      --multi-az                      Deploy to multiple data centers.
  -r, --region string                 AWS region where your worker pool will be located. (overrides the AWS_REGION environment variable)
      --version string                Version of OpenShift that will be used to install the cluster, for example "4.3.10"
      --channel-group string          Channel group is the name of the group where this image belongs, for example "stable" or "fast". (default "stable")
      --compute-machine-type string   Instance type for the compute nodes. Determines the amount of memory and vCPU allocated to each compute node.
      --compute-nodes int             Number of worker nodes to provision per zone. Single zone clusters need at least 2 nodes, multizone clusters need at least 3 nodes. (default 2)
      --machine-cidr ipNet            Block of IP addresses used by OpenShift while installing the cluster, for example "10.0.0.0/16".
      --service-cidr ipNet            Block of IP addresses for services, for example "172.30.0.0/16".
      --pod-cidr ipNet                Block of IP addresses from which Pod IP addresses are allocated, for example "10.128.0.0/14".
      --host-prefix int               Subnet prefix length to assign to each individual node. For example, if host prefix is set to "23", then each node is assigned a /23 subnet out of the given CIDR.
      --private                       Restrict master API endpoint and application routes to direct, private connectivity.
      --watch                         Watch cluster installation logs.
      --dry-run                       Simulate creating the cluster.
      --use-paid-ami                  Whether to use the paid AMI from AWS. Requires a valid subscription to the MOA Product.
  -h, --help                          help for cluster
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
  -i, --interactive      Enable interactive mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [moactl create](moactl_create.md)	 - Create a resource from stdin

