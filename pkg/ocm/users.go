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
	errors "github.com/zgalor/weberr"
	"net/http"

	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func (c *Client) GetUser(clusterID string, group string, username string) (*cmv1.User, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Groups().Group(group).
		Users().User(username).
		Get().
		Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil, nil
		}
		return nil, handleErr(response.Error(), err)
	}

	return response.Body(), nil
}

func (c *Client) GetUsers(clusterID string, group string) ([]*cmv1.User, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Groups().Group(group).
		Users().
		List().Page(1).Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return response.Items().Slice(), nil
}

func (c *Client) CreateUser(clusterID string, group string, user *cmv1.User) (*cmv1.User, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Groups().Group(group).
		Users().
		Add().Body(user).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

func (c *Client) DeleteUser(clusterID string, group string, username string) error {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Groups().Group(group).
		Users().User(username).
		Delete().
		Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}
	return nil
}

func (c *Client) CreateRoleBinding(subscriptionID string, userName string, roleID string) (*amv1.RoleBinding, error) {
	roleBinding, err := amv1.NewRoleBinding().Account(amv1.NewAccount().Username(userName)).
		Role(amv1.NewRole().ID(roleID)).Build()
	if err != nil {
		return nil, err
	}
	response, err := c.ocm.AccountsMgmt().V1().Subscriptions().Subscription(subscriptionID).RoleBindings().
		Add().Body(roleBinding).Send()

	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

func (c *Client) DeleteRoleBinding(subscriptionID string, userName string, roleID string) error {
	query := fmt.Sprintf("account_username = '%s' and role.id = '%s'", userName, roleID)
	response, err := c.ocm.AccountsMgmt().V1().Subscriptions().Subscription(subscriptionID).RoleBindings().
		List().Search(query).Send()
	if err != nil {
		return err
	}
	if err != nil {
		return handleErr(response.Error(), err)
	}
	if response.Size() == 0 {
		return errors.NotFound.UserErrorf("Role binding '%s' for the user '%s' is not found", roleID, userName)
	}
	_, err = c.ocm.AccountsMgmt().V1().Subscriptions().Subscription(subscriptionID).RoleBindings().
		RoleBinding(response.Items().Get(0).ID()).Delete().Send()
	if err != nil {
		return err
	}
	return nil
}
