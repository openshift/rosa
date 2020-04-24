module gitlab.cee.redhat.com/service/moactl

go 1.13

require (
	github.com/aws/aws-sdk-go v1.29.17
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-bindata/go-bindata v3.1.2+incompatible // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/mitchellh/go-homedir v1.1.0
	github.com/openshift-online/ocm-sdk-go v0.1.100
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.6
	github.com/spf13/pflag v1.0.5
	gitlab.com/c0b/go-ordered-json v0.0.0-20171130231205-49bbdab258c2
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	k8s.io/apimachinery v0.18.2
)

replace github.com/golang/glog => github.com/kubermatic/glog-logrus v0.0.0-20180829085450-3fa5b9870d1d
