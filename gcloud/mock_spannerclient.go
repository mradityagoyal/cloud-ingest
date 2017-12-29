// Code generated by MockGen. DO NOT EDIT.
// Source: gcloud/spannerclient.go

// Package gcloud is a generated GoMock package.
package gcloud

import (
	spanner "cloud.google.com/go/spanner"
	context "context"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
	time "time"
)

// MockSpanner is a mock of Spanner interface
type MockSpanner struct {
	ctrl     *gomock.Controller
	recorder *MockSpannerMockRecorder
}

// MockSpannerMockRecorder is the mock recorder for MockSpanner
type MockSpannerMockRecorder struct {
	mock *MockSpanner
}

// NewMockSpanner creates a new mock instance
func NewMockSpanner(ctrl *gomock.Controller) *MockSpanner {
	mock := &MockSpanner{ctrl: ctrl}
	mock.recorder = &MockSpannerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSpanner) EXPECT() *MockSpannerMockRecorder {
	return m.recorder
}

// Single mocks base method
func (m *MockSpanner) Single() ReadOnlyTransaction {
	ret := m.ctrl.Call(m, "Single")
	ret0, _ := ret[0].(ReadOnlyTransaction)
	return ret0
}

// Single indicates an expected call of Single
func (mr *MockSpannerMockRecorder) Single() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Single", reflect.TypeOf((*MockSpanner)(nil).Single))
}

// ReadWriteTransaction mocks base method
func (m *MockSpanner) ReadWriteTransaction(ctx context.Context, f func(context.Context, ReadWriteTransaction) error) (time.Time, error) {
	ret := m.ctrl.Call(m, "ReadWriteTransaction", ctx, f)
	ret0, _ := ret[0].(time.Time)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadWriteTransaction indicates an expected call of ReadWriteTransaction
func (mr *MockSpannerMockRecorder) ReadWriteTransaction(ctx, f interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadWriteTransaction", reflect.TypeOf((*MockSpanner)(nil).ReadWriteTransaction), ctx, f)
}

// MockReadOnlyTransaction is a mock of ReadOnlyTransaction interface
type MockReadOnlyTransaction struct {
	ctrl     *gomock.Controller
	recorder *MockReadOnlyTransactionMockRecorder
}

// MockReadOnlyTransactionMockRecorder is the mock recorder for MockReadOnlyTransaction
type MockReadOnlyTransactionMockRecorder struct {
	mock *MockReadOnlyTransaction
}

// NewMockReadOnlyTransaction creates a new mock instance
func NewMockReadOnlyTransaction(ctrl *gomock.Controller) *MockReadOnlyTransaction {
	mock := &MockReadOnlyTransaction{ctrl: ctrl}
	mock.recorder = &MockReadOnlyTransactionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockReadOnlyTransaction) EXPECT() *MockReadOnlyTransactionMockRecorder {
	return m.recorder
}

// Query mocks base method
func (m *MockReadOnlyTransaction) Query(ctx context.Context, statement spanner.Statement) RowIterator {
	ret := m.ctrl.Call(m, "Query", ctx, statement)
	ret0, _ := ret[0].(RowIterator)
	return ret0
}

// Query indicates an expected call of Query
func (mr *MockReadOnlyTransactionMockRecorder) Query(ctx, statement interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockReadOnlyTransaction)(nil).Query), ctx, statement)
}

// Read mocks base method
func (m *MockReadOnlyTransaction) Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) RowIterator {
	ret := m.ctrl.Call(m, "Read", ctx, table, keys, columns)
	ret0, _ := ret[0].(RowIterator)
	return ret0
}

// Read indicates an expected call of Read
func (mr *MockReadOnlyTransactionMockRecorder) Read(ctx, table, keys, columns interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockReadOnlyTransaction)(nil).Read), ctx, table, keys, columns)
}

// ReadRow mocks base method
func (m *MockReadOnlyTransaction) ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error) {
	ret := m.ctrl.Call(m, "ReadRow", ctx, table, key, columns)
	ret0, _ := ret[0].(*spanner.Row)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadRow indicates an expected call of ReadRow
func (mr *MockReadOnlyTransactionMockRecorder) ReadRow(ctx, table, key, columns interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadRow", reflect.TypeOf((*MockReadOnlyTransaction)(nil).ReadRow), ctx, table, key, columns)
}

// MockReadWriteTransaction is a mock of ReadWriteTransaction interface
type MockReadWriteTransaction struct {
	ctrl     *gomock.Controller
	recorder *MockReadWriteTransactionMockRecorder
}

// MockReadWriteTransactionMockRecorder is the mock recorder for MockReadWriteTransaction
type MockReadWriteTransactionMockRecorder struct {
	mock *MockReadWriteTransaction
}

// NewMockReadWriteTransaction creates a new mock instance
func NewMockReadWriteTransaction(ctrl *gomock.Controller) *MockReadWriteTransaction {
	mock := &MockReadWriteTransaction{ctrl: ctrl}
	mock.recorder = &MockReadWriteTransactionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockReadWriteTransaction) EXPECT() *MockReadWriteTransactionMockRecorder {
	return m.recorder
}

// Query mocks base method
func (m *MockReadWriteTransaction) Query(ctx context.Context, statement spanner.Statement) RowIterator {
	ret := m.ctrl.Call(m, "Query", ctx, statement)
	ret0, _ := ret[0].(RowIterator)
	return ret0
}

// Query indicates an expected call of Query
func (mr *MockReadWriteTransactionMockRecorder) Query(ctx, statement interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockReadWriteTransaction)(nil).Query), ctx, statement)
}

// Read mocks base method
func (m *MockReadWriteTransaction) Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) RowIterator {
	ret := m.ctrl.Call(m, "Read", ctx, table, keys, columns)
	ret0, _ := ret[0].(RowIterator)
	return ret0
}

// Read indicates an expected call of Read
func (mr *MockReadWriteTransactionMockRecorder) Read(ctx, table, keys, columns interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockReadWriteTransaction)(nil).Read), ctx, table, keys, columns)
}

// ReadRow mocks base method
func (m *MockReadWriteTransaction) ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error) {
	ret := m.ctrl.Call(m, "ReadRow", ctx, table, key, columns)
	ret0, _ := ret[0].(*spanner.Row)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadRow indicates an expected call of ReadRow
func (mr *MockReadWriteTransactionMockRecorder) ReadRow(ctx, table, key, columns interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadRow", reflect.TypeOf((*MockReadWriteTransaction)(nil).ReadRow), ctx, table, key, columns)
}

// BufferWrite mocks base method
func (m *MockReadWriteTransaction) BufferWrite(ms []*spanner.Mutation) error {
	ret := m.ctrl.Call(m, "BufferWrite", ms)
	ret0, _ := ret[0].(error)
	return ret0
}

// BufferWrite indicates an expected call of BufferWrite
func (mr *MockReadWriteTransactionMockRecorder) BufferWrite(ms interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BufferWrite", reflect.TypeOf((*MockReadWriteTransaction)(nil).BufferWrite), ms)
}

// MockRowIterator is a mock of RowIterator interface
type MockRowIterator struct {
	ctrl     *gomock.Controller
	recorder *MockRowIteratorMockRecorder
}

// MockRowIteratorMockRecorder is the mock recorder for MockRowIterator
type MockRowIteratorMockRecorder struct {
	mock *MockRowIterator
}

// NewMockRowIterator creates a new mock instance
func NewMockRowIterator(ctrl *gomock.Controller) *MockRowIterator {
	mock := &MockRowIterator{ctrl: ctrl}
	mock.recorder = &MockRowIteratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRowIterator) EXPECT() *MockRowIteratorMockRecorder {
	return m.recorder
}

// Do mocks base method
func (m *MockRowIterator) Do(f func(*spanner.Row) error) error {
	ret := m.ctrl.Call(m, "Do", f)
	ret0, _ := ret[0].(error)
	return ret0
}

// Do indicates an expected call of Do
func (mr *MockRowIteratorMockRecorder) Do(f interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*MockRowIterator)(nil).Do), f)
}

// Next mocks base method
func (m *MockRowIterator) Next() (*spanner.Row, error) {
	ret := m.ctrl.Call(m, "Next")
	ret0, _ := ret[0].(*spanner.Row)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Next indicates an expected call of Next
func (mr *MockRowIteratorMockRecorder) Next() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockRowIterator)(nil).Next))
}

// Stop mocks base method
func (m *MockRowIterator) Stop() {
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop
func (mr *MockRowIteratorMockRecorder) Stop() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockRowIterator)(nil).Stop))
}