// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import context "context"
import mock "github.com/stretchr/testify/mock"
import p2p "fileshare/p2p"

// DownloadMethod is an autogenerated mock type for the DownloadMethod type
type DownloadMethod struct {
	mock.Mock
}

// NextFragment provides a mock function with given fields: ctx, peers, fm
func (_m *DownloadMethod) NextFragment(ctx context.Context, peers map[string]p2p.Client, fm p2p.FileMetaData) (p2p.Client, int, error) {
	ret := _m.Called(ctx, peers, fm)

	var r0 p2p.Client
	if rf, ok := ret.Get(0).(func(context.Context, map[string]p2p.Client, p2p.FileMetaData) p2p.Client); ok {
		r0 = rf(ctx, peers, fm)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(p2p.Client)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(context.Context, map[string]p2p.Client, p2p.FileMetaData) int); ok {
		r1 = rf(ctx, peers, fm)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, map[string]p2p.Client, p2p.FileMetaData) error); ok {
		r2 = rf(ctx, peers, fm)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
