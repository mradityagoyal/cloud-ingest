// Code generated by MockGen. DO NOT EDIT.
// Source: dcp/listresultreader.go

// Package dcp is a generated GoMock package.
package dcp

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockListingResultReader is a mock of ListingResultReader interface
type MockListingResultReader struct {
	ctrl     *gomock.Controller
	recorder *MockListingResultReaderMockRecorder
}

// MockListingResultReaderMockRecorder is the mock recorder for MockListingResultReader
type MockListingResultReaderMockRecorder struct {
	mock *MockListingResultReader
}

// NewMockListingResultReader creates a new mock instance
func NewMockListingResultReader(ctrl *gomock.Controller) *MockListingResultReader {
	mock := &MockListingResultReader{ctrl: ctrl}
	mock.recorder = &MockListingResultReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockListingResultReader) EXPECT() *MockListingResultReaderMockRecorder {
	return m.recorder
}

// ReadEntries mocks base method
func (m *MockListingResultReader) ReadEntries(ctx context.Context, bucket, object string, offset int64, maxEntries int) ([]ListFileEntry, int64, error) {
	ret := m.ctrl.Call(m, "ReadEntries", ctx, bucket, object, offset, maxEntries)
	ret0, _ := ret[0].([]ListFileEntry)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ReadEntries indicates an expected call of ReadEntries
func (mr *MockListingResultReaderMockRecorder) ReadEntries(ctx, bucket, object, offset, maxEntries interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadEntries", reflect.TypeOf((*MockListingResultReader)(nil).ReadEntries), ctx, bucket, object, offset, maxEntries)
}
