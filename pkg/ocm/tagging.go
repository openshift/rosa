// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocmerrors "github.com/openshift-online/ocm-sdk-go/errors"
)

const clusterTagsPath = "/api/clusters_mgmt/v1/clusters/"

// GetClusterUserTags returns customer-managed cluster AWS tags, omitting protected tags.
func (c *Client) GetClusterUserTags(clusterID string) (map[string]string, error) {
	return c.getUserTags(clusterTagsPath+url.PathEscape(clusterID), "aws")
}

// GetNodePoolUserTags returns customer-managed node-pool tags, including inherited cluster tags.
func (c *Client) GetNodePoolUserTags(clusterID, nodePoolID string) (map[string]string, error) {
	path := clusterTagsPath + url.PathEscape(clusterID) + "/node_pools/" + url.PathEscape(nodePoolID)
	return c.getUserTags(path, "aws_node_pool")
}

// getUserTags requests customer-only AWS tags and omits null deletion markers from the response.
func (c *Client) getUserTags(path, field string) (map[string]string, error) {
	response, err := c.ocm.Get().Path(path).Parameter("fetchUserTagsOnly", true).Send()
	if err != nil {
		return nil, fmt.Errorf("failed to read AWS tags: %w", err)
	}
	if err := tagResponseError(response.Status(), response.Bytes()); err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(response.Bytes(), &object); err != nil {
		return nil, fmt.Errorf("failed to decode AWS tags response: %w", err)
	}
	var awsFields struct {
		Tags map[string]*string `json:"tags"`
	}
	if len(object[field]) > 0 {
		if err := json.Unmarshal(object[field], &awsFields); err != nil {
			return nil, fmt.Errorf("failed to decode AWS tags: %w", err)
		}
	}
	tags := make(map[string]string, len(awsFields.Tags))
	for key, value := range awsFields.Tags {
		if value != nil {
			tags[key] = *value
		}
	}
	return tags, nil
}

// UpdateNodePoolWithTags combines the typed node-pool edit and nullable tag changes in one PATCH.
func (c *Client) UpdateNodePoolWithTags(clusterID string, nodePool *cmv1.NodePool,
	changes map[string]*string) (*cmv1.NodePool, error) {
	var typed bytes.Buffer
	if err := cmv1.MarshalNodePool(nodePool, &typed); err != nil {
		return nil, fmt.Errorf("failed to encode node pool update: %w", err)
	}
	path := clusterTagsPath + url.PathEscape(clusterID) + "/node_pools/" + url.PathEscape(nodePool.ID())
	response, err := c.patchWithTags(path, typed.Bytes(), "aws_node_pool", changes)
	if err != nil {
		return nil, err
	}
	updated, err := cmv1.UnmarshalNodePool(response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode updated node pool: %w", err)
	}
	return updated, nil
}

// updateClusterWithTags retains the typed cluster edit while adding nullable AWS tag changes.
func (c *Client) updateClusterWithTags(clusterID string, cluster *cmv1.Cluster,
	changes map[string]*string) error {
	var typed bytes.Buffer
	if err := cmv1.MarshalCluster(cluster, &typed); err != nil {
		return fmt.Errorf("failed to encode cluster update: %w", err)
	}
	_, err := c.patchWithTags(clusterTagsPath+url.PathEscape(clusterID), typed.Bytes(), "aws", changes)
	return err
}

// patchWithTags inserts nullable tag changes into a typed JSON update and sends one PATCH.
func (c *Client) patchWithTags(path string, typed []byte, field string,
	changes map[string]*string) ([]byte, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(typed, &object); err != nil {
		return nil, fmt.Errorf("failed to decode typed update: %w", err)
	}
	var awsFields map[string]json.RawMessage
	if len(object[field]) > 0 {
		if err := json.Unmarshal(object[field], &awsFields); err != nil {
			return nil, fmt.Errorf("failed to decode typed AWS update: %w", err)
		}
	}
	if awsFields == nil {
		awsFields = make(map[string]json.RawMessage)
	}
	tagsJSON, err := json.Marshal(changes)
	if err != nil {
		return nil, fmt.Errorf("failed to encode AWS tag changes: %w", err)
	}
	awsFields["tags"] = tagsJSON
	object[field], err = json.Marshal(awsFields)
	if err != nil {
		return nil, fmt.Errorf("failed to encode AWS update: %w", err)
	}
	body, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("failed to encode update: %w", err)
	}
	response, err := c.ocm.Patch().Path(path).Header("Content-Type", "application/json").Bytes(body).Send()
	if err != nil {
		return nil, fmt.Errorf("failed to update AWS tags: %w", err)
	}
	if err := tagResponseError(response.Status(), response.Bytes()); err != nil {
		return nil, err
	}
	return response.Bytes(), nil
}

// tagResponseError preserves the API validation reason when a tag request fails.
func tagResponseError(status int, body []byte) error {
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		return nil
	}
	serviceErr, err := ocmerrors.UnmarshalErrorStatus(body, status)
	if err == nil && serviceErr != nil {
		return handleErr(serviceErr, fmt.Errorf("OCM returned HTTP %d", status))
	}
	return fmt.Errorf("OCM returned HTTP %d", status)
}
