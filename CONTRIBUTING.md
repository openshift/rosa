# Questions?

If you have any questions about the code or how to contribute, don't hesitate to
[open an issue](https://github.com/openshift/rosa/issues/new) in this repo.

## CI

This repository is using Prow CI running at https://prow.ci.openshift.org/,
configured in https://github.com/openshift/release repo.

`.golangciversion` file is read by the `lint` job commands there:
https://github.com/openshift/release/blob/master/ci-operator/config/openshift/rosa/openshift-rosa-master.yaml
