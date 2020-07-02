## moactl create addon

Install add-ons on cluster

### Synopsis

Install Red Hat managed add-ons on a cluster

```
moactl create addon [flags]
```

### Examples

```
  # Add the CodeReady Workspaces add-on installation to the cluster
  moactl create addon --cluster=mycluster codeready-workspaces
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to add the IdP to (required).
  -h, --help             help for addon
```

### Options inherited from parent commands

```
      --debug         Enable debug mode.
  -i, --interactive   Enable interactive mode.
  -v, --v Level       log level for V logs
```

### SEE ALSO

* [moactl create](moactl_create.md)	 - Create a resource from stdin

