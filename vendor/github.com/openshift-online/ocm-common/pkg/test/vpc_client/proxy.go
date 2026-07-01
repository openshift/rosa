package vpc_client

import (
	"context"
	"fmt"
	"regexp"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/file"
	"github.com/openshift-online/ocm-common/pkg/log"
	"github.com/openshift-online/ocm-common/pkg/utils"
)

// proxyInstanceReadyTimeoutMinutes is the maximum time to wait for a proxy
// EC2 instance to pass its status checks before attempting SSH setup.
const proxyInstanceReadyTimeoutMinutes = 10

// namedCmd pairs a shell command with a safe description for logging.
// When logLabel is set it is logged instead of the raw command, preventing
// sensitive values (e.g. credentials) from appearing in log output.
type namedCmd struct {
	cmd      string
	logLabel string
}

func (c namedCmd) safeLabel() string {
	if c.logLabel != "" {
		return c.logLabel
	}
	return c.cmd
}

// validateProxyCredential rejects values that contain characters capable of
// escaping a Python double-quoted string literal. Only letters, digits,
// hyphens, underscores, and dots are allowed.
func validateProxyCredential(fieldName, value string) error {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' {
			return fmt.Errorf("proxy %s contains invalid character %q: only letters, digits, hyphens, underscores, and dots are allowed", fieldName, r)
		}
	}
	return nil
}

/*
MITM Proxy Authentication Support

This package now supports username and password authentication for MITM proxy servers.
Users can connect to the proxy using either format:

1. Without authentication (backward compatible):
   http_proxy=http://10.0.0.183:8080

2. With authentication:
   http_proxy=http://username:password@10.0.0.183:8080

To use authentication, call LaunchProxyInstanceWithAuth() with username and password parameters.
To maintain backward compatibility, use the original LaunchProxyInstance() function.

Security Note: Passwords are never logged in plain text. All logging operations
are designed to protect sensitive credential information.

Example usage:
    // Without authentication (original behavior)
    instance, privateIP, caContent, err := vpc.LaunchProxyInstance(zone, keypairName, privateKeyPath)

    // With authentication
    instance, privateIP, caContent, err := vpc.LaunchProxyInstanceWithAuth(zone, keypairName, privateKeyPath, "myuser", "mypass")

    // Get proxy URLs
    httpProxy := vpc.GetProxyURL(privateIP, "myuser", "mypass")
    httpsProxy := vpc.GetHTTPSProxyURL(privateIP, "myuser", "mypass")
*/

// FindProxyLaunchImage will try to find a proper image based on the filters to launch the proxy instance
// No parameter needed here
// It will return an image ID and error if happens
func (vpc *VPC) FindProxyLaunchImage() (string, error) {
	filters := map[string][]string{
		"architecture": {
			"x86_64",
		},
		"state": {
			"available",
		},
		"image-type": {
			"machine",
		},
		"is-public": {
			"true",
		},
		"virtualization-type": {
			"hvm",
		},
		"root-device-type": {
			"ebs",
		},
	}

	output, err := vpc.AWSClient.DescribeImage([]string{}, filters)
	if err != nil {
		log.LogError("Describe image met error: %s", err)
		return "", err
	}
	if output == nil || len(output.Images) < 1 {
		log.LogError("Got the empty image via the filter: %s", filters)
		err = fmt.Errorf("got empty image list via the filter: %s", filters)
		return "", err
	}
	expectedImageID := ""
	nameRegexp := regexp.MustCompile(`al[0-9]{4}-ami[0-9\.-]*kernel[0-9-\._a-z]*`)
	for _, image := range output.Images {
		if nameRegexp.MatchString(*image.Name) {
			expectedImageID = *image.ImageId
			break
		}
	}
	if expectedImageID != "" {
		log.LogInfo("Got the image ID : %s", expectedImageID)
	} else {
		log.LogError("Got no proper image meet the regex: %s", nameRegexp.String())
		err = fmt.Errorf("got no proper image meet the regex: %s", nameRegexp.String())
	}

	return expectedImageID, err
}

// LaunchProxyInstance will launch a proxy instance on the indicated zone.
// If set imageID to empty, it will find the proxy image in the ProxyImageMap map
// LaunchProxyInstance will return proxyInstance detail, privateIPAddress,CAcontent and error
func (vpc *VPC) LaunchProxyInstance(zone string, keypairName string, privateKeyPath string) (inst types.Instance, privateIP string, proxyServerCA string, err error) {
	return vpc.LaunchProxyInstanceWithAuth(zone, keypairName, privateKeyPath, "", "")
}

// LaunchProxyInstanceWithAuth will launch a proxy instance on the indicated zone with optional authentication.
// If username and password are provided, the proxy will require authentication
// If set imageID to empty, it will find the proxy image in the ProxyImageMap map
// LaunchProxyInstanceWithAuth will return proxyInstance detail, privateIPAddress,CAcontent and error
func (vpc *VPC) LaunchProxyInstanceWithAuth(zone string, keypairName string, privateKeyPath string, username string, password string) (
	inst types.Instance, privateIP string, proxyServerCA string, err error) {
	imageID, err := vpc.FindProxyLaunchImage()
	if err != nil {
		return inst, "", "", err
	}
	pubSubnet, err := vpc.PreparePublicSubnet(zone)
	if err != nil {
		log.LogInfo("Error preparing a subnet in current zone %s with image ID %s: %s", zone, imageID, err)
		return inst, "", "", err
	}
	SGID, err := vpc.CreateAndAuthorizeDefaultSecurityGroupForProxy()
	if err != nil {
		log.LogError("Prepare SG failed for the proxy preparation %s", err)
		return inst, "", "", err
	}
	randomStr := utils.RandomLabel(2)
	keyName := fmt.Sprintf("%s-%s-%s", CON.InstanceKeyNamePrefix, randomStr, keypairName)
	key, err := vpc.CreateKeyPair(keyName)
	if err != nil {
		log.LogError("Create key pair %s failed %s", keyName, err)
		return inst, "", "", err
	}
	tags := map[string]string{
		"Name":  CON.ProxyName,
		"VpcId": vpc.VpcID,
	}
	_, err = vpc.AWSClient.TagResource(*key.KeyPairId, tags)
	if err != nil {
		log.LogError("Add tag for key pair %s failed %s", *key.KeyPairId, err)
		return inst, "", "", err
	}
	privateKeyName := fmt.Sprintf("%s-%s", keypairName, "keyPair.pem")
	sshKey, err := file.WriteToFile(*key.KeyMaterial, privateKeyName, privateKeyPath)
	if err != nil {
		log.LogError("Write private key to file failed %s", err)
		return inst, "", "", err
	}

	instOut, err := vpc.AWSClient.LaunchInstance(pubSubnet.ID, imageID, 1, "t3.medium", keyName, []string{SGID}, true)
	if err != nil {
		log.LogError("Launch proxy instance failed %s", err)
		return inst, "", "", err
	} else {
		log.LogInfo("Launch proxy instance %s succeed", *instOut.Instances[0].InstanceId)
	}

	instID := *instOut.Instances[0].InstanceId
	_, err = vpc.AWSClient.TagResource(instID, tags)
	if err != nil {
		log.LogError("Add tag for instance  %s failed %s", instID, err)
		return inst, "", "", err
	}

	publicIP, err := vpc.AWSClient.AllocateEIPAndAssociateInstance(instID)
	if err != nil {
		log.LogError("Prepare EIP failed for the proxy preparation %s", err)
		return inst, "", "", err
	}
	log.LogInfo("Prepare EIP successfully for the proxy preparation. Launch with IP: %s", publicIP)

	log.LogInfo("Waiting for proxy instance %s to pass EC2 status checks before attempting SSH...", instID)
	if _, err = vpc.AWSClient.WaitForInstancesRunning(context.Background(), []string{instID}, proxyInstanceReadyTimeoutMinutes); err != nil {
		log.LogError("Proxy instance %s did not become ready within timeout: %s", instID, err)
		return inst, "", "", err
	}

	hostname := fmt.Sprintf("%s:22", publicIP)
	err = setupMITMProxyServer(sshKey, hostname, username, password)
	if err != nil {
		log.LogError("Setup MITM proxy server failed  %s", err)
		return inst, "", "", err
	}

	cmd := "cat mitm-ca.pem"
	caContent, err := Exec_CMD(CON.AWSInstanceUser, sshKey, hostname, cmd)
	if err != nil {
		log.LogError("login instance to run cmd %s:%s", cmd, err)
		return inst, "", "", err
	}
	return instOut.Instances[0], *instOut.Instances[0].PrivateIpAddress, caContent, err
}

func setupMITMProxyServer(sshKey string, hostname string, username string, password string) (err error) {
	cmds := []namedCmd{
		{cmd: "sudo yum install -y wget"},
		{cmd: "wget https://snapshots.mitmproxy.org/7.0.2/mitmproxy-7.0.2-linux.tar.gz"},
		{cmd: "mkdir mitm"},
		{cmd: "tar zxvf mitmproxy-7.0.2-linux.tar.gz -C mitm"},
	}

	// Reject partial credentials: supplying only one would silently launch an
	// unauthenticated proxy, which is never the intended behaviour.
	if (username == "") != (password == "") {
		return fmt.Errorf("proxy credentials must be provided together: username provided=%t password provided=%t",
			username != "", password != "")
	}

	if username != "" && password != "" {
		if err := validateProxyCredential("username", username); err != nil {
			return err
		}
		if err := validateProxyCredential("password", password); err != nil {
			return err
		}
		authScript := fmt.Sprintf(`cat > ~/proxy_auth.py << 'EOF'
import mitmproxy.http
import mitmproxy.addons.core

class ProxyAuth:
    def __init__(self, username, password):
        self.username = username
        self.password = password
    
    def request(self, flow: mitmproxy.http.HTTPFlow) -> None:
        if flow.request.headers.get("Proxy-Authorization"):
            return
        
        # Add basic authentication header
        import base64
        auth = base64.b64encode(f"{self.username}:{self.password}".encode()).decode()
        flow.request.headers["Proxy-Authorization"] = f"Basic {auth}"

addons = [
    ProxyAuth("%s", "%s"),
    mitmproxy.addons.core.Core()
]
EOF`, username, password)

		cmds = append(cmds,
			namedCmd{cmd: authScript, logLabel: "[proxy auth configuration — redacted]"},
			namedCmd{cmd: "nohup ./mitm/mitmdump --showhost --ssl-insecure --script ~/proxy_auth.py > mitm.log 2>&1 &"},
		)
	} else {
		cmds = append(cmds,
			namedCmd{cmd: "nohup ./mitm/mitmdump --showhost --ssl-insecure > mitm.log 2>&1 &"},
		)
	}

	cmds = append(cmds,
		namedCmd{cmd: "sleep 5"},
		namedCmd{cmd: "http_proxy=127.0.0.1:8080 curl http://mitm.it/cert/pem -s > ~/mitm-ca.pem"},
	)

	for _, c := range cmds {
		_, err = Exec_CMD(CON.AWSInstanceUser, sshKey, hostname, c.cmd)
		if err != nil {
			return err
		}
		log.LogDebug("Run the cmd successfully: %s", c.safeLabel())
	}
	return
}

// GetProxyURL returns the HTTP proxy URL with optional authentication
// If username and password are provided, they will be included in the URL
func (vpc *VPC) GetProxyURL(privateIP string, username string, password string) string {
	if username != "" && password != "" {
		return fmt.Sprintf("http://%s:%s@%s:8080", username, password, privateIP)
	}
	return fmt.Sprintf("http://%s:8080", privateIP)
}

// GetHTTPSProxyURL returns the HTTPS proxy URL with optional authentication
// If username and password are provided, they will be included in the URL
func (vpc *VPC) GetHTTPSProxyURL(privateIP string, username string, password string) string {
	if username != "" && password != "" {
		return fmt.Sprintf("https://%s:%s@%s:8080", username, password, privateIP)
	}
	return fmt.Sprintf("https://%s:8080", privateIP)
}
