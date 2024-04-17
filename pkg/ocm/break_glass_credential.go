package ocm

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	errors "github.com/zgalor/weberr"
)

const pollKubeconfigInterval = 200 * time.Second

func (c *Client) CreateBreakGlassCredential(clusterID string,
	breakGlassCredential *cmv1.BreakGlassCredential) (*cmv1.BreakGlassCredential, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).BreakGlassCredentials().
		Add().Body(breakGlassCredential).Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

func (c *Client) GetBreakGlassCredentials(clusterID string) ([]*cmv1.BreakGlassCredential, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		BreakGlassCredentials().
		List().Page(1).Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Items().Slice(), nil
}

func (c *Client) GetBreakGlassCredential(clusterID string,
	breakGlassCredentialID string) (*cmv1.BreakGlassCredential, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).BreakGlassCredentials().
		BreakGlassCredential(breakGlassCredentialID).
		Get().
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

func (c *Client) DeleteBreakGlassCredentials(clusterID string) error {

	response, err := c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterID).BreakGlassCredentials().Delete().Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}
	return nil
}

func (c *Client) PollKubeconfig(clusterID string, credentialID string) (kubeconfig string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer func() {
		cancel()
	}()

	credentialClient := c.ocm.ClustersMgmt().V1().Clusters().
		Cluster(clusterID).BreakGlassCredentials().BreakGlassCredential(credentialID)
	response, err := credentialClient.Poll().
		Interval(pollKubeconfigInterval).
		StartContext(ctx)
	if err != nil {
		err = fmt.Errorf("Failed to poll kubeconfig for cluster '%s' with break glass credential '%s': %v",
			clusterID, credentialID, err)
		if response.Status() == http.StatusNotFound {
			err = errors.NotFound.UserErrorf("Failed to poll kubeconfig for cluster '%s' with break glass credential '%s'",
				clusterID, credentialID)
		}
		return
	}

	return response.Body().Kubeconfig(), nil
}
