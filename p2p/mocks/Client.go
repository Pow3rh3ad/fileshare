// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import context "context"
import mock "github.com/stretchr/testify/mock"
import p2p "fileshare/p2p"

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// Alive provides a mock function with given fields:
func (_m *Client) Alive() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Download provides a mock function with given fields: ctx, fileHash, fragmentID, out
func (_m *Client) Download(ctx context.Context, fileHash string, fragmentID int, out chan p2p.DownloadResult) {
	_m.Called(ctx, fileHash, fragmentID, out)
}

// FragmentsAvailable provides a mock function with given fields: ctx, fileHash
func (_m *Client) FragmentsAvailable(ctx context.Context, fileHash string) []int {
	ret := _m.Called(ctx, fileHash)

	var r0 []int
	if rf, ok := ret.Get(0).(func(context.Context, string) []int); ok {
		r0 = rf(ctx, fileHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]int)
		}
	}

	return r0
}

// List provides a mock function with given fields: ctx
func (_m *Client) List(ctx context.Context) ([]p2p.FileMetaData, error) {
	ret := _m.Called(ctx)

	var r0 []p2p.FileMetaData
	if rf, ok := ret.Get(0).(func(context.Context) []p2p.FileMetaData); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]p2p.FileMetaData)
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

// Name provides a mock function with given fields:
func (_m *Client) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
