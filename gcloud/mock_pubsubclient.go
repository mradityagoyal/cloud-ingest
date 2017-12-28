// Code generated by MockGen. DO NOT EDIT.
// Source: gcloud/pubsubclient.go

// Package gcloud is a generated GoMock package.
package gcloud

import (
	pubsub "cloud.google.com/go/pubsub"
	context "context"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockPS is a mock of PS interface
type MockPS struct {
	ctrl     *gomock.Controller
	recorder *MockPSMockRecorder
}

// MockPSMockRecorder is the mock recorder for MockPS
type MockPSMockRecorder struct {
	mock *MockPS
}

// NewMockPS creates a new mock instance
func NewMockPS(ctrl *gomock.Controller) *MockPS {
	mock := &MockPS{ctrl: ctrl}
	mock.recorder = &MockPSMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPS) EXPECT() *MockPSMockRecorder {
	return m.recorder
}

// Topic mocks base method
func (m *MockPS) Topic(id string) PSTopic {
	ret := m.ctrl.Call(m, "Topic", id)
	ret0, _ := ret[0].(PSTopic)
	return ret0
}

// Topic indicates an expected call of Topic
func (mr *MockPSMockRecorder) Topic(id interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Topic", reflect.TypeOf((*MockPS)(nil).Topic), id)
}

// TopicInProject mocks base method
func (m *MockPS) TopicInProject(id, projectID string) PSTopic {
	ret := m.ctrl.Call(m, "TopicInProject", id, projectID)
	ret0, _ := ret[0].(PSTopic)
	return ret0
}

// TopicInProject indicates an expected call of TopicInProject
func (mr *MockPSMockRecorder) TopicInProject(id, projectID interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TopicInProject", reflect.TypeOf((*MockPS)(nil).TopicInProject), id, projectID)
}

// MockPSTopic is a mock of PSTopic interface
type MockPSTopic struct {
	ctrl     *gomock.Controller
	recorder *MockPSTopicMockRecorder
}

// MockPSTopicMockRecorder is the mock recorder for MockPSTopic
type MockPSTopicMockRecorder struct {
	mock *MockPSTopic
}

// NewMockPSTopic creates a new mock instance
func NewMockPSTopic(ctrl *gomock.Controller) *MockPSTopic {
	mock := &MockPSTopic{ctrl: ctrl}
	mock.recorder = &MockPSTopicMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPSTopic) EXPECT() *MockPSTopicMockRecorder {
	return m.recorder
}

// Publish mocks base method
func (m *MockPSTopic) Publish(ctx context.Context, msg *pubsub.Message) PSPublishResult {
	ret := m.ctrl.Call(m, "Publish", ctx, msg)
	ret0, _ := ret[0].(PSPublishResult)
	return ret0
}

// Publish indicates an expected call of Publish
func (mr *MockPSTopicMockRecorder) Publish(ctx, msg interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Publish", reflect.TypeOf((*MockPSTopic)(nil).Publish), ctx, msg)
}

// Stop mocks base method
func (m *MockPSTopic) Stop() {
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop
func (mr *MockPSTopicMockRecorder) Stop() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockPSTopic)(nil).Stop))
}

// MockPSSubscription is a mock of PSSubscription interface
type MockPSSubscription struct {
	ctrl     *gomock.Controller
	recorder *MockPSSubscriptionMockRecorder
}

// MockPSSubscriptionMockRecorder is the mock recorder for MockPSSubscription
type MockPSSubscriptionMockRecorder struct {
	mock *MockPSSubscription
}

// NewMockPSSubscription creates a new mock instance
func NewMockPSSubscription(ctrl *gomock.Controller) *MockPSSubscription {
	mock := &MockPSSubscription{ctrl: ctrl}
	mock.recorder = &MockPSSubscriptionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPSSubscription) EXPECT() *MockPSSubscriptionMockRecorder {
	return m.recorder
}

// Receive mocks base method
func (m *MockPSSubscription) Receive(ctx context.Context, f func(context.Context, *pubsub.Message)) error {
	ret := m.ctrl.Call(m, "Receive", ctx, f)
	ret0, _ := ret[0].(error)
	return ret0
}

// Receive indicates an expected call of Receive
func (mr *MockPSSubscriptionMockRecorder) Receive(ctx, f interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Receive", reflect.TypeOf((*MockPSSubscription)(nil).Receive), ctx, f)
}

// MockPSPublishResult is a mock of PSPublishResult interface
type MockPSPublishResult struct {
	ctrl     *gomock.Controller
	recorder *MockPSPublishResultMockRecorder
}

// MockPSPublishResultMockRecorder is the mock recorder for MockPSPublishResult
type MockPSPublishResultMockRecorder struct {
	mock *MockPSPublishResult
}

// NewMockPSPublishResult creates a new mock instance
func NewMockPSPublishResult(ctrl *gomock.Controller) *MockPSPublishResult {
	mock := &MockPSPublishResult{ctrl: ctrl}
	mock.recorder = &MockPSPublishResultMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPSPublishResult) EXPECT() *MockPSPublishResultMockRecorder {
	return m.recorder
}

// Get mocks base method
func (m *MockPSPublishResult) Get(ctx context.Context) (string, error) {
	ret := m.ctrl.Call(m, "Get", ctx)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockPSPublishResultMockRecorder) Get(ctx interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockPSPublishResult)(nil).Get), ctx)
}
