package reflection

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc"
	refv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	refv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Version int

const (
	V1Alpha Version = iota
	Auto
)

type Client struct {
	v1    refv1.ServerReflection_ServerReflectionInfoClient
	v1a   refv1alpha.ServerReflection_ServerReflectionInfoClient
	useV1 bool
}

func NewClient(ctx context.Context, conn grpc.ClientConnInterface, v Version) (*Client, error) {
	if v == V1Alpha {
		st, err := refv1alpha.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
		if err != nil {
			return nil, err
		}
		return &Client{v1a: st, useV1: false}, nil
	}
	st, err := refv1.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err == nil {
		return &Client{v1: st, useV1: true}, nil
	}
	if v == Auto {
		stA, errA := refv1alpha.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
		if errA != nil {
			return nil, err
		}
		return &Client{v1a: stA, useV1: false}, nil
	}
	return nil, err
}

func (c *Client) send(req *refv1.ServerReflectionRequest) error {
	if c.useV1 {
		return c.v1.Send(req)
	}
	// convert to v1alpha
	reqA := &refv1alpha.ServerReflectionRequest{Host: req.Host}
	switch r := req.MessageRequest.(type) {
	case *refv1.ServerReflectionRequest_FileByFilename:
		reqA.MessageRequest = &refv1alpha.ServerReflectionRequest_FileByFilename{FileByFilename: r.FileByFilename}
	case *refv1.ServerReflectionRequest_FileContainingSymbol:
		reqA.MessageRequest = &refv1alpha.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: r.FileContainingSymbol}
	case *refv1.ServerReflectionRequest_ListServices:
		reqA.MessageRequest = &refv1alpha.ServerReflectionRequest_ListServices{ListServices: r.ListServices}
	default:
		return errors.New("unsupported request")
	}
	return c.v1a.Send(reqA)
}

func (c *Client) recv() (*refv1.ServerReflectionResponse, error) {
	if c.useV1 {
		return c.v1.Recv()
	}
	respA, err := c.v1a.Recv()
	if err != nil {
		return nil, err
	}
	resp := &refv1.ServerReflectionResponse{ValidHost: respA.ValidHost}
	switch r := respA.MessageResponse.(type) {
	case *refv1alpha.ServerReflectionResponse_FileDescriptorResponse:
		resp.MessageResponse = &refv1.ServerReflectionResponse_FileDescriptorResponse{
			FileDescriptorResponse: &refv1.FileDescriptorResponse{FileDescriptorProto: r.FileDescriptorResponse.FileDescriptorProto},
		}
	case *refv1alpha.ServerReflectionResponse_ListServicesResponse:
		svcs := make([]*refv1.ServiceResponse, len(r.ListServicesResponse.Service))
		for i, s := range r.ListServicesResponse.Service {
			svcs[i] = &refv1.ServiceResponse{Name: s.Name}
		}
		resp.MessageResponse = &refv1.ServerReflectionResponse_ListServicesResponse{
			ListServicesResponse: &refv1.ListServiceResponse{Service: svcs},
		}
	case *refv1alpha.ServerReflectionResponse_ErrorResponse:
		resp.MessageResponse = &refv1.ServerReflectionResponse_ErrorResponse{
			ErrorResponse: &refv1.ErrorResponse{ErrorCode: r.ErrorResponse.ErrorCode, ErrorMessage: r.ErrorResponse.ErrorMessage},
		}
	default:
		return nil, errors.New("unsupported response")
	}
	return resp, nil
}

func (c *Client) ListServices() ([]string, error) {
	req := &refv1.ServerReflectionRequest{MessageRequest: &refv1.ServerReflectionRequest_ListServices{ListServices: "*"}}
	if err := c.send(req); err != nil {
		return nil, err
	}
	resp, err := c.recv()
	if err != nil {
		return nil, err
	}
	ls := resp.GetListServicesResponse()
	if ls == nil {
		return nil, errors.New("no services")
	}
	names := make([]string, len(ls.Service))
	for i, s := range ls.Service {
		names[i] = s.Name
	}
	return names, nil
}

func (c *Client) FileContainingSymbol(symbol string) ([]protoreflect.FileDescriptor, error) {
	req := &refv1.ServerReflectionRequest{MessageRequest: &refv1.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: symbol}}
	if err := c.send(req); err != nil {
		return nil, err
	}
	resp, err := c.recv()
	if err != nil {
		return nil, err
	}
	fdresp := resp.GetFileDescriptorResponse()
	if fdresp == nil {
		return nil, errors.New("no descriptor")
	}
	res := make([]protoreflect.FileDescriptor, len(fdresp.FileDescriptorProto))
	for i, b := range fdresp.FileDescriptorProto {
		var fdProto descriptorpb.FileDescriptorProto
		if err := proto.Unmarshal(b, &fdProto); err != nil {
			return nil, err
		}
		fd, err := protodesc.NewFile(&fdProto, nil)
		if err != nil {
			return nil, err
		}
		res[i] = fd
	}
	return res, nil
}

func (c *Client) ResolveService(name string) (protoreflect.ServiceDescriptor, error) {
	fds, err := c.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	for _, fd := range fds {
		if svc := fd.Services().ByName(protoreflect.Name(name[strings.LastIndex(name, ".")+1:])); svc != nil {
			return svc, nil
		}
	}
	return nil, errors.New("service not found")
}

func (c *Client) Reset() {
	if c.useV1 {
		c.v1.CloseSend()
	} else {
		c.v1a.CloseSend()
	}
}
