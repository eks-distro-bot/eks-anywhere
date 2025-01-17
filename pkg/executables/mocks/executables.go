// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/aws/eks-anywhere/pkg/executables (interfaces: Executable)

// Package mocks is a generated GoMock package.
package mocks

import (
	bytes "bytes"
	context "context"
	reflect "reflect"

	executables "github.com/aws/eks-anywhere/pkg/executables"
	gomock "github.com/golang/mock/gomock"
)

// MockExecutable is a mock of Executable interface.
type MockExecutable struct {
	ctrl     *gomock.Controller
	recorder *MockExecutableMockRecorder
}

// MockExecutableMockRecorder is the mock recorder for MockExecutable.
type MockExecutableMockRecorder struct {
	mock *MockExecutable
}

// NewMockExecutable creates a new mock instance.
func NewMockExecutable(ctrl *gomock.Controller) *MockExecutable {
	mock := &MockExecutable{ctrl: ctrl}
	mock.recorder = &MockExecutableMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutable) EXPECT() *MockExecutableMockRecorder {
	return m.recorder
}

// Command mocks base method.
func (m *MockExecutable) Command(arg0 context.Context, arg1 ...string) *executables.Command {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Command", varargs...)
	ret0, _ := ret[0].(*executables.Command)
	return ret0
}

// Command indicates an expected call of Command.
func (mr *MockExecutableMockRecorder) Command(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Command", reflect.TypeOf((*MockExecutable)(nil).Command), varargs...)
}

// Execute mocks base method.
func (m *MockExecutable) Execute(arg0 context.Context, arg1 ...string) (bytes.Buffer, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Execute", varargs...)
	ret0, _ := ret[0].(bytes.Buffer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Execute indicates an expected call of Execute.
func (mr *MockExecutableMockRecorder) Execute(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Execute", reflect.TypeOf((*MockExecutable)(nil).Execute), varargs...)
}

// ExecuteWithEnv mocks base method.
func (m *MockExecutable) ExecuteWithEnv(arg0 context.Context, arg1 map[string]string, arg2 ...string) (bytes.Buffer, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ExecuteWithEnv", varargs...)
	ret0, _ := ret[0].(bytes.Buffer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecuteWithEnv indicates an expected call of ExecuteWithEnv.
func (mr *MockExecutableMockRecorder) ExecuteWithEnv(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecuteWithEnv", reflect.TypeOf((*MockExecutable)(nil).ExecuteWithEnv), varargs...)
}

// ExecuteWithStdin mocks base method.
func (m *MockExecutable) ExecuteWithStdin(arg0 context.Context, arg1 []byte, arg2 ...string) (bytes.Buffer, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ExecuteWithStdin", varargs...)
	ret0, _ := ret[0].(bytes.Buffer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecuteWithStdin indicates an expected call of ExecuteWithStdin.
func (mr *MockExecutableMockRecorder) ExecuteWithStdin(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecuteWithStdin", reflect.TypeOf((*MockExecutable)(nil).ExecuteWithStdin), varargs...)
}

// Run mocks base method.
func (m *MockExecutable) Run(arg0 *executables.Command) (bytes.Buffer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", arg0)
	ret0, _ := ret[0].(bytes.Buffer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Run indicates an expected call of Run.
func (mr *MockExecutableMockRecorder) Run(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockExecutable)(nil).Run), arg0)
}
