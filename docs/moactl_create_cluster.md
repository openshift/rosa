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
  moactl create cluster --name=mycluster

  # Create a cluster in the us-east-2 region
  moactl create cluster --name=mycluster --region=us-east-2
```

### Options

```
  -n, --name string                   Name of the cluster. This will be used when generating a sub-domain for your cluster on openshiftapps.com.
  -r, --region string                 AWS region where your worker pool will be located. (overrides the AWS_REGION environment variable)
      --version string                Version of OpenShift that will be used to install the cluster, for example "4.3.10"
      --multi-az                      Deploy to multiple data centers.
      --expiration-time string        Specific time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.
      --expiration duration           Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.
                                      
      --compute-machine-type string   Instance type for the compute nodes. Determines the amount of memory and vCPU allocated to each compute node.
      --compute-nodes int             Number of worker nodes to provision per zone. Single zone clusters need at least 4 nodes, while multizone clusters need at least 9 nodes (3 per zone) for resiliency.
                                      
      --machine-cidr ipNet            Block of IP addresses used by OpenShift while installing the cluster, for example "10.0.0.0/16".
      --service-cidr ipNet            Block of IP addresses for services, for example "172.30.0.0/16".
      --pod-cidr ipNet                Block of IP addresses from which Pod IP addresses are allocated, for example "10.128.0.0/14".
      --host-prefix int               Subnet prefix length to assign to each individual node. For example, if host prefix is set to "23", then each node is assigned a /23 subnet out of the given CIDR.
      --private                       Restrict master API endpoint and application routes to direct, private connectivity.
  -h, --help                          help for cluster
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [moactl create](moactl_create.md)	 - Create a resource from stdin

