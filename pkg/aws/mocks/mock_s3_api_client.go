// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/aws/api_interface/s3_api_client.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	s3 "github.com/aws/aws-sdk-go-v2/service/s3"
	gomock "github.com/golang/mock/gomock"
)

// MockS3ApiClient is a mock of S3ApiClient interface.
type MockS3ApiClient struct {
	ctrl     *gomock.Controller
	recorder *MockS3ApiClientMockRecorder
}

// MockS3ApiClientMockRecorder is the mock recorder for MockS3ApiClient.
type MockS3ApiClientMockRecorder struct {
	mock *MockS3ApiClient
}

// NewMockS3ApiClient creates a new mock instance.
func NewMockS3ApiClient(ctrl *gomock.Controller) *MockS3ApiClient {
	mock := &MockS3ApiClient{ctrl: ctrl}
	mock.recorder = &MockS3ApiClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockS3ApiClient) EXPECT() *MockS3ApiClientMockRecorder {
	return m.recorder
}

// CreateBucket mocks base method.
func (m *MockS3ApiClient) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CreateBucket", varargs...)
	ret0, _ := ret[0].(*s3.CreateBucketOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBucket indicates an expected call of CreateBucket.
func (mr *MockS3ApiClientMockRecorder) CreateBucket(ctx, params interface{}, optFns ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBucket", reflect.TypeOf((*MockS3ApiClient)(nil).CreateBucket), varargs...)
}

// DeleteBucket mocks base method.
func (m *MockS3ApiClient) DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteBucket", varargs...)
	ret0, _ := ret[0].(*s3.DeleteBucketOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteBucket indicates an expected call of DeleteBucket.
func (mr *MockS3ApiClientMockRecorder) DeleteBucket(ctx, params interface{}, optFns ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteBucket", reflect.TypeOf((*MockS3ApiClient)(nil).DeleteBucket), varargs...)
}

// DeleteObject mocks base method.
func (m *MockS3ApiClient) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteObject", varargs...)
	ret0, _ := ret[0].(*s3.DeleteObjectOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteObject indicates an expected call of DeleteObject.
func (mr *MockS3ApiClientMockRecorder) DeleteObject(ctx, params interface{}, optFns ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteObject", reflect.TypeOf((*MockS3ApiClient)(nil).DeleteObject), varargs...)
}

// HeadBucket mocks base method.
func (m *MockS3ApiClient) HeadBucket(arg0 context.Context, arg1 *s3.HeadBucketInput, arg2 ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "HeadBucket", varargs...)
	ret0, _ := ret[0].(*s3.HeadBucketOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HeadBucket indicates an expected call of HeadBucket.
func (mr *MockS3ApiClientMockRecorder) HeadBucket(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HeadBucket", reflect.TypeOf((*MockS3ApiClient)(nil).HeadBucket), varargs...)
}

// ListObjects mocks base method.
func (m *MockS3ApiClient) ListObjects(ctx context.Context, params *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListObjects", varargs...)
	ret0, _ := ret[0].(*s3.ListObjectsOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListObjects indicates an expected call of ListObjects.
func (mr *MockS3ApiClientMockRecorder) ListObjects(ctx, params interface{}, optFns ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListObjects", reflect.TypeOf((*MockS3ApiClient)(nil).ListObjects), varargs...)
}

// PutBucketPolicy mocks base method.
func (m *MockS3ApiClient) PutBucketPolicy(ctx context.Context, params *s3.PutBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.PutBucketPolicyOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PutBucketPolicy", varargs...)
	ret0, _ := ret[0].(*s3.PutBucketPolicyOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PutBucketPolicy indicates an expected call of PutBucketPolicy.
func (mr *MockS3ApiClientMockRecorder) PutBucketPolicy(ctx, params interface{}, optFns ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutBucketPolicy", reflect.TypeOf((*MockS3ApiClient)(nil).PutBucketPolicy), varargs...)
}

// PutBucketTagging mocks base method.
func (m *MockS3ApiClient) PutBucketTagging(ctx context.Context, params *s3.PutBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.PutBucketTaggingOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PutBucketTagging", varargs...)
	ret0, _ := ret[0].(*s3.PutBucketTaggingOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PutBucketTagging indicates an expected call of PutBucketTagging.
func (mr *MockS3ApiClientMockRecorder) PutBucketTagging(ctx, params interface{}, optFns ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutBucketTagging", reflect.TypeOf((*MockS3ApiClient)(nil).PutBucketTagging), varargs...)
}

// PutObject mocks base method.
func (m *MockS3ApiClient) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PutObject", varargs...)
	ret0, _ := ret[0].(*s3.PutObjectOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PutObject indicates an expected call of PutObject.
func (mr *MockS3ApiClientMockRecorder) PutObject(ctx, params interface{}, optFns ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutObject", reflect.TypeOf((*MockS3ApiClient)(nil).PutObject), varargs...)
}

// PutPublicAccessBlock mocks base method.
func (m *MockS3ApiClient) PutPublicAccessBlock(ctx context.Context, params *s3.PutPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.PutPublicAccessBlockOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PutPublicAccessBlock", varargs...)
	ret0, _ := ret[0].(*s3.PutPublicAccessBlockOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PutPublicAccessBlock indicates an expected call of PutPublicAccessBlock.
func (mr *MockS3ApiClientMockRecorder) PutPublicAccessBlock(ctx, params interface{}, optFns ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutPublicAccessBlock", reflect.TypeOf((*MockS3ApiClient)(nil).PutPublicAccessBlock), varargs...)
}