// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             v3.12.4
// source: signer.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.62.0 or later.
const _ = grpc.SupportPackageIsVersion8

const (
	Signer_BatchSign_FullMethodName     = "/signer.Signer/BatchSign"
	Signer_BatchRetrieve_FullMethodName = "/signer.Signer/BatchRetrieve"
	Signer_GetStatus_FullMethodName     = "/signer.Signer/GetStatus"
)

// SignerClient is the client API for Signer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SignerClient interface {
	// This API accepts rows of encoded blobs to sign from clients. The node will verify the blobs existance in `DAEntrance` contract and validate the received rows.
	BatchSign(ctx context.Context, in *BatchSignRequest, opts ...grpc.CallOption) (*BatchSignReply, error)
	// This retrieves the requested encoded rows from the DA node database.
	BatchRetrieve(ctx context.Context, in *BatchRetrieveRequest, opts ...grpc.CallOption) (*BatchRetrieveReply, error)
	GetStatus(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatusReply, error)
}

type signerClient struct {
	cc grpc.ClientConnInterface
}

func NewSignerClient(cc grpc.ClientConnInterface) SignerClient {
	return &signerClient{cc}
}

func (c *signerClient) BatchSign(ctx context.Context, in *BatchSignRequest, opts ...grpc.CallOption) (*BatchSignReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(BatchSignReply)
	err := c.cc.Invoke(ctx, Signer_BatchSign_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) BatchRetrieve(ctx context.Context, in *BatchRetrieveRequest, opts ...grpc.CallOption) (*BatchRetrieveReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(BatchRetrieveReply)
	err := c.cc.Invoke(ctx, Signer_BatchRetrieve_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) GetStatus(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatusReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StatusReply)
	err := c.cc.Invoke(ctx, Signer_GetStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SignerServer is the server API for Signer service.
// All implementations must embed UnimplementedSignerServer
// for forward compatibility
type SignerServer interface {
	// This API accepts rows of encoded blobs to sign from clients. The node will verify the blobs existance in `DAEntrance` contract and validate the received rows.
	BatchSign(context.Context, *BatchSignRequest) (*BatchSignReply, error)
	// This retrieves the requested encoded rows from the DA node database.
	BatchRetrieve(context.Context, *BatchRetrieveRequest) (*BatchRetrieveReply, error)
	GetStatus(context.Context, *Empty) (*StatusReply, error)
	mustEmbedUnimplementedSignerServer()
}

// UnimplementedSignerServer must be embedded to have forward compatible implementations.
type UnimplementedSignerServer struct {
}

func (UnimplementedSignerServer) BatchSign(context.Context, *BatchSignRequest) (*BatchSignReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BatchSign not implemented")
}
func (UnimplementedSignerServer) BatchRetrieve(context.Context, *BatchRetrieveRequest) (*BatchRetrieveReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BatchRetrieve not implemented")
}
func (UnimplementedSignerServer) GetStatus(context.Context, *Empty) (*StatusReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}
func (UnimplementedSignerServer) mustEmbedUnimplementedSignerServer() {}

// UnsafeSignerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SignerServer will
// result in compilation errors.
type UnsafeSignerServer interface {
	mustEmbedUnimplementedSignerServer()
}

func RegisterSignerServer(s grpc.ServiceRegistrar, srv SignerServer) {
	s.RegisterService(&Signer_ServiceDesc, srv)
}

func _Signer_BatchSign_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BatchSignRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).BatchSign(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Signer_BatchSign_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).BatchSign(ctx, req.(*BatchSignRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_BatchRetrieve_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BatchRetrieveRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).BatchRetrieve(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Signer_BatchRetrieve_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).BatchRetrieve(ctx, req.(*BatchRetrieveRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_GetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).GetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Signer_GetStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).GetStatus(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// Signer_ServiceDesc is the grpc.ServiceDesc for Signer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Signer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "signer.Signer",
	HandlerType: (*SignerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "BatchSign",
			Handler:    _Signer_BatchSign_Handler,
		},
		{
			MethodName: "BatchRetrieve",
			Handler:    _Signer_BatchRetrieve_Handler,
		},
		{
			MethodName: "GetStatus",
			Handler:    _Signer_GetStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "signer.proto",
}