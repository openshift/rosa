module github.com/openshift/rosa

go 1.15

require (
	github.com/AlecAivazis/survey/v2 v2.2.15
	github.com/aws/aws-sdk-go v1.39.3
	github.com/briandowns/spinner v1.11.1
	github.com/dustin/go-humanize v1.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/mock v1.6.0
	github.com/hashicorp/go-version v1.3.0
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/openshift-online/ocm-sdk-go v0.1.214
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/zgalor/weberr v0.6.0
	gitlab.com/c0b/go-ordered-json v0.0.0-20171130231205-49bbdab258c2
)

replace github.com/golang/glog => github.com/kubermatic/glog-logrus v0.0.0-20180829085450-3fa5b9870d1d
