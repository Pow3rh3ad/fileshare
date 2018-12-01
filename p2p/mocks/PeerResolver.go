// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import context "context"
import mock "github.com/stretchr/testify/mock"
import p2p "fileshare/p2p"

// PeerResolver is an autogenerated mock type for the PeerResolver type
type PeerResolver struct {
	mock.Mock
}

// Discover provides a mock function with given fields: ctx
func (_m *PeerResolver) Discover(ctx context.Context) ([]p2p.Client, error) {
	ret := _m.Called(ctx)

	var r0 []p2p.Client
	if rf, ok := ret.Get(0).(func(context.Context) []p2p.Client); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]p2p.Client)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Listen provides a mock function with given fields: ctx, addr
func (_m *PeerResolver) Listen(ctx context.Context, addr string) {
	_m.Called(ctx, addr)
}
