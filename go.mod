module gitlab.cee.redhat.com/service/moactl

go 1.13

require (
	github.com/aws/aws-sdk-go v1.29.17
	github.com/openshift-online/ocm-sdk-go v0.1.95
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/cobra v0.0.6
	github.com/spf13/pflag v1.0.5
	gitlab.com/c0b/go-ordered-json v0.0.0-20171130231205-49bbdab258c2
)

replace github.com/golang/glog => github.com/kubermatic/glog-logrus v0.0.0-20180829085450-3fa5b9870d1d
