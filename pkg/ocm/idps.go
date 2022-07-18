/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm

import (
	"fmt"
	"net/http"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	HTPasswdIDPType = "HTPasswd"
	GithubIDPType   = "GitHub"
	GitlabIDPType   = "GitLab"
	GoogleIDPType   = "Google"
	LDAPIDPType     = "LDAP"
	OpenIDIDPType   = "OpenID"
)

func (c *Client) GetIdentityProviders(clusterID string) ([]*cmv1.IdentityProvider, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		IdentityProviders().
		List().Page(1).Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return response.Items().Slice(), nil
}

func (c *Client) CreateIdentityProvider(clusterID string, idp *cmv1.IdentityProvider) (*cmv1.IdentityProvider, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		IdentityProviders().
		Add().Body(idp).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

func (c *Client) GetHTPasswdUserList(clusterID, htpasswdIDPId string) (*cmv1.HTPasswdUserList, error) {
	listResponse, err := c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterID).
		IdentityProviders().IdentityProvider(htpasswdIDPId).HtpasswdUsers().List().Send()
	if err != nil {
		if listResponse.Error().Status() == http.StatusNotFound {
			return nil, nil
		}
		return nil, handleErr(listResponse.Error(), err)
	}
	return listResponse.Items(), nil
}

func (c *Client) AddHTPasswdUser(username, password, clusterID, idpID string) error {
	htpasswdUser, _ := cmv1.NewHTPasswdUser().Username(username).Password(password).Build()
	response, err := c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterID).
		IdentityProviders().IdentityProvider(idpID).HtpasswdUsers().Add().Body(htpasswdUser).Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}
	return nil
}

func (c *Client) DeleteHTPasswdUser(username, clusterID string, htpasswdIDP *cmv1.IdentityProvider) error {
	var userID string

	idp, ok := htpasswdIDP.GetHtpasswd()
	if !ok {
		return fmt.Errorf("Failed to get htpasswd idp for cluster '%s'", clusterID)
	}
	if idp.Username() != "" {
		//the admin was created with ROSA release less than 4.10
		//remove the entire idp
		response, err := c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterID).
			IdentityProviders().IdentityProvider(htpasswdIDP.ID()).Delete().Send()
		if err != nil {
			return handleErr(response.Error(), err)
		}
	} else {
		listResponse, err := c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterID).
			IdentityProviders().IdentityProvider(htpasswdIDP.ID()).HtpasswdUsers().List().Send()
		if err != nil {
			if listResponse.Error().Status() == http.StatusNotFound {
				return nil
			}
			return handleErr(listResponse.Error(), err)
		}
		listResponse.Items().Each(func(user *cmv1.HTPasswdUser) bool {
			if user.Username() == username {
				userID = user.ID()
			}
			return true
		})
		if userID == "" && listResponse.Items().Len() != 0 {
			return fmt.Errorf("HTPasswd user named '%s' on cluster '%s' does not exist", username, clusterID)
		}
		deleteResponse, err := c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterID).
			IdentityProviders().IdentityProvider(htpasswdIDP.ID()).HtpasswdUsers().
			HtpasswdUser(userID).Delete().Send()
		if err != nil {
			return handleErr(deleteResponse.Error(), err)
		}
	}
	return nil
}

func (c *Client) DeleteIdentityProvider(clusterID string, idpID string) error {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		IdentityProviders().IdentityProvider(idpID).
		Delete().
		Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}
	return nil
}

func IdentityProviderType(idp *cmv1.IdentityProvider) string {
	switch idp.Type() {
	case "GithubIdentityProvider":
		return GithubIDPType
	case "GitlabIdentityProvider":
		return GitlabIDPType
	case "GoogleIdentityProvider":
		return GoogleIDPType
	case "HTPasswdIdentityProvider":
		return HTPasswdIDPType
	case "LDAPIdentityProvider":
		return LDAPIDPType
	case "OpenIDIdentityProvider":
		return OpenIDIDPType
	}

	return ""
}
