// Code generated by MockGen. DO NOT EDIT.
// Source: ./proto/auth/auth_grpc.pb.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	auth "crmSystem/proto/auth"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	grpc "google.golang.org/grpc"
)

// MockAuthServiceClient is a mock of AuthServiceClient interface.
type MockAuthServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockAuthServiceClientMockRecorder
}

// MockAuthServiceClientMockRecorder is the mock recorder for MockAuthServiceClient.
type MockAuthServiceClientMockRecorder struct {
	mock *MockAuthServiceClient
}

// NewMockAuthServiceClient creates a new mock instance.
func NewMockAuthServiceClient(ctrl *gomock.Controller) *MockAuthServiceClient {
	mock := &MockAuthServiceClient{ctrl: ctrl}
	mock.recorder = &MockAuthServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAuthServiceClient) EXPECT() *MockAuthServiceClientMockRecorder {
	return m.recorder
}

// Login mocks base method.
func (m *MockAuthServiceClient) Login(ctx context.Context, in *auth.LoginAuthRequest, opts ...grpc.CallOption) (*auth.LoginAuthResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Login", varargs...)
	ret0, _ := ret[0].(*auth.LoginAuthResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Login indicates an expected call of Login.
func (mr *MockAuthServiceClientMockRecorder) Login(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Login", reflect.TypeOf((*MockAuthServiceClient)(nil).Login), varargs...)
}

// Register mocks base method.
func (m *MockAuthServiceClient) Register(ctx context.Context, in *auth.RegisterAuthRequest, opts ...grpc.CallOption) (*auth.RegisterAuthResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Register", varargs...)
	ret0, _ := ret[0].(*auth.RegisterAuthResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Register indicates an expected call of Register.
func (mr *MockAuthServiceClientMockRecorder) Register(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Register", reflect.TypeOf((*MockAuthServiceClient)(nil).Register), varargs...)
}

// MockAuthServiceServer is a mock of AuthServiceServer interface.
type MockAuthServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockAuthServiceServerMockRecorder
}

// MockAuthServiceServerMockRecorder is the mock recorder for MockAuthServiceServer.
type MockAuthServiceServerMockRecorder struct {
	mock *MockAuthServiceServer
}

// NewMockAuthServiceServer creates a new mock instance.
func NewMockAuthServiceServer(ctrl *gomock.Controller) *MockAuthServiceServer {
	mock := &MockAuthServiceServer{ctrl: ctrl}
	mock.recorder = &MockAuthServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAuthServiceServer) EXPECT() *MockAuthServiceServerMockRecorder {
	return m.recorder
}

// Login mocks base method.
func (m *MockAuthServiceServer) Login(arg0 context.Context, arg1 *auth.LoginAuthRequest) (*auth.LoginAuthResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Login", arg0, arg1)
	ret0, _ := ret[0].(*auth.LoginAuthResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Login indicates an expected call of Login.
func (mr *MockAuthServiceServerMockRecorder) Login(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Login", reflect.TypeOf((*MockAuthServiceServer)(nil).Login), arg0, arg1)
}

// Register mocks base method.
func (m *MockAuthServiceServer) Register(arg0 context.Context, arg1 *auth.RegisterAuthRequest) (*auth.RegisterAuthResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Register", arg0, arg1)
	ret0, _ := ret[0].(*auth.RegisterAuthResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Register indicates an expected call of Register.
func (mr *MockAuthServiceServerMockRecorder) Register(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Register", reflect.TypeOf((*MockAuthServiceServer)(nil).Register), arg0, arg1)
}

// mustEmbedUnimplementedAuthServiceServer mocks base method.
func (m *MockAuthServiceServer) mustEmbedUnimplementedAuthServiceServer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "mustEmbedUnimplementedAuthServiceServer")
}

// mustEmbedUnimplementedAuthServiceServer indicates an expected call of mustEmbedUnimplementedAuthServiceServer.
func (mr *MockAuthServiceServerMockRecorder) mustEmbedUnimplementedAuthServiceServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "mustEmbedUnimplementedAuthServiceServer", reflect.TypeOf((*MockAuthServiceServer)(nil).mustEmbedUnimplementedAuthServiceServer))
}

// MockUnsafeAuthServiceServer is a mock of UnsafeAuthServiceServer interface.
type MockUnsafeAuthServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockUnsafeAuthServiceServerMockRecorder
}

// MockUnsafeAuthServiceServerMockRecorder is the mock recorder for MockUnsafeAuthServiceServer.
type MockUnsafeAuthServiceServerMockRecorder struct {
	mock *MockUnsafeAuthServiceServer
}

// NewMockUnsafeAuthServiceServer creates a new mock instance.
func NewMockUnsafeAuthServiceServer(ctrl *gomock.Controller) *MockUnsafeAuthServiceServer {
	mock := &MockUnsafeAuthServiceServer{ctrl: ctrl}
	mock.recorder = &MockUnsafeAuthServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnsafeAuthServiceServer) EXPECT() *MockUnsafeAuthServiceServerMockRecorder {
	return m.recorder
}

// mustEmbedUnimplementedAuthServiceServer mocks base method.
func (m *MockUnsafeAuthServiceServer) mustEmbedUnimplementedAuthServiceServer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "mustEmbedUnimplementedAuthServiceServer")
}

// mustEmbedUnimplementedAuthServiceServer indicates an expected call of mustEmbedUnimplementedAuthServiceServer.
func (mr *MockUnsafeAuthServiceServerMockRecorder) mustEmbedUnimplementedAuthServiceServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "mustEmbedUnimplementedAuthServiceServer", reflect.TypeOf((*MockUnsafeAuthServiceServer)(nil).mustEmbedUnimplementedAuthServiceServer))
}
