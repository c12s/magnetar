// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: magnetar.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// MagnetarClient is the client API for Magnetar service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MagnetarClient interface {
	GetNode(ctx context.Context, in *GetNodeReq, opts ...grpc.CallOption) (*GetNodeResp, error)
	ListNodes(ctx context.Context, in *ListNodesReq, opts ...grpc.CallOption) (*ListNodesResp, error)
	QueryNodes(ctx context.Context, in *QueryNodesReq, opts ...grpc.CallOption) (*QueryNodesResp, error)
	PutLabel(ctx context.Context, in *PutLabelReq, opts ...grpc.CallOption) (*PutLabelResp, error)
	DeleteLabel(ctx context.Context, in *DeleteLabelReq, opts ...grpc.CallOption) (*DeleteLabelResp, error)
}

type magnetarClient struct {
	cc grpc.ClientConnInterface
}

func NewMagnetarClient(cc grpc.ClientConnInterface) MagnetarClient {
	return &magnetarClient{cc}
}

func (c *magnetarClient) GetNode(ctx context.Context, in *GetNodeReq, opts ...grpc.CallOption) (*GetNodeResp, error) {
	out := new(GetNodeResp)
	err := c.cc.Invoke(ctx, "/proto.Magnetar/GetNode", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *magnetarClient) ListNodes(ctx context.Context, in *ListNodesReq, opts ...grpc.CallOption) (*ListNodesResp, error) {
	out := new(ListNodesResp)
	err := c.cc.Invoke(ctx, "/proto.Magnetar/ListNodes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *magnetarClient) QueryNodes(ctx context.Context, in *QueryNodesReq, opts ...grpc.CallOption) (*QueryNodesResp, error) {
	out := new(QueryNodesResp)
	err := c.cc.Invoke(ctx, "/proto.Magnetar/QueryNodes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *magnetarClient) PutLabel(ctx context.Context, in *PutLabelReq, opts ...grpc.CallOption) (*PutLabelResp, error) {
	out := new(PutLabelResp)
	err := c.cc.Invoke(ctx, "/proto.Magnetar/PutLabel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *magnetarClient) DeleteLabel(ctx context.Context, in *DeleteLabelReq, opts ...grpc.CallOption) (*DeleteLabelResp, error) {
	out := new(DeleteLabelResp)
	err := c.cc.Invoke(ctx, "/proto.Magnetar/DeleteLabel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MagnetarServer is the server API for Magnetar service.
// All implementations must embed UnimplementedMagnetarServer
// for forward compatibility
type MagnetarServer interface {
	GetNode(context.Context, *GetNodeReq) (*GetNodeResp, error)
	ListNodes(context.Context, *ListNodesReq) (*ListNodesResp, error)
	QueryNodes(context.Context, *QueryNodesReq) (*QueryNodesResp, error)
	PutLabel(context.Context, *PutLabelReq) (*PutLabelResp, error)
	DeleteLabel(context.Context, *DeleteLabelReq) (*DeleteLabelResp, error)
	mustEmbedUnimplementedMagnetarServer()
}

// UnimplementedMagnetarServer must be embedded to have forward compatible implementations.
type UnimplementedMagnetarServer struct {
}

func (UnimplementedMagnetarServer) GetNode(context.Context, *GetNodeReq) (*GetNodeResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetNode not implemented")
}
func (UnimplementedMagnetarServer) ListNodes(context.Context, *ListNodesReq) (*ListNodesResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListNodes not implemented")
}
func (UnimplementedMagnetarServer) QueryNodes(context.Context, *QueryNodesReq) (*QueryNodesResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryNodes not implemented")
}
func (UnimplementedMagnetarServer) PutLabel(context.Context, *PutLabelReq) (*PutLabelResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PutLabel not implemented")
}
func (UnimplementedMagnetarServer) DeleteLabel(context.Context, *DeleteLabelReq) (*DeleteLabelResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteLabel not implemented")
}
func (UnimplementedMagnetarServer) mustEmbedUnimplementedMagnetarServer() {}

// UnsafeMagnetarServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MagnetarServer will
// result in compilation errors.
type UnsafeMagnetarServer interface {
	mustEmbedUnimplementedMagnetarServer()
}

func RegisterMagnetarServer(s grpc.ServiceRegistrar, srv MagnetarServer) {
	s.RegisterService(&Magnetar_ServiceDesc, srv)
}

func _Magnetar_GetNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetNodeReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MagnetarServer).GetNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.Magnetar/GetNode",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MagnetarServer).GetNode(ctx, req.(*GetNodeReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Magnetar_ListNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListNodesReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MagnetarServer).ListNodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.Magnetar/ListNodes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MagnetarServer).ListNodes(ctx, req.(*ListNodesReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Magnetar_QueryNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryNodesReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MagnetarServer).QueryNodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.Magnetar/QueryNodes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MagnetarServer).QueryNodes(ctx, req.(*QueryNodesReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Magnetar_PutLabel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PutLabelReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MagnetarServer).PutLabel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.Magnetar/PutLabel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MagnetarServer).PutLabel(ctx, req.(*PutLabelReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Magnetar_DeleteLabel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteLabelReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MagnetarServer).DeleteLabel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.Magnetar/DeleteLabel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MagnetarServer).DeleteLabel(ctx, req.(*DeleteLabelReq))
	}
	return interceptor(ctx, in, info, handler)
}

// Magnetar_ServiceDesc is the grpc.ServiceDesc for Magnetar service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Magnetar_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.Magnetar",
	HandlerType: (*MagnetarServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetNode",
			Handler:    _Magnetar_GetNode_Handler,
		},
		{
			MethodName: "ListNodes",
			Handler:    _Magnetar_ListNodes_Handler,
		},
		{
			MethodName: "QueryNodes",
			Handler:    _Magnetar_QueryNodes_Handler,
		},
		{
			MethodName: "PutLabel",
			Handler:    _Magnetar_PutLabel_Handler,
		},
		{
			MethodName: "DeleteLabel",
			Handler:    _Magnetar_DeleteLabel_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "magnetar.proto",
}
