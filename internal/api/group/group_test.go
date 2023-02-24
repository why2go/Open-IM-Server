package group

import (
	common "OpenIM/internal/api2rpc"
	"OpenIM/pkg/apistruct"
	"OpenIM/pkg/proto/group"
	"context"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type ApiBind[A, B any] interface {
	OperationID() string
	OpUserID() (string, error)
	Bind(*A) error
	Context() context.Context
	Resp(resp *B, err error)
}

func NewGin[A, B any](c *gin.Context) ApiBind[A, B] {
	return &ginApiBind[A, B]{
		c: c,
	}
}

type ginApiBind[A, B any] struct {
	c *gin.Context
}

func (g *ginApiBind[A, B]) OperationID() string {
	return g.c.GetHeader("operationID")
}

func (g *ginApiBind[A, B]) OpUserID() (string, error) {
	return "", nil
}

func (g *ginApiBind[A, B]) Bind(a *A) error {
	return g.c.BindJSON(a)
}

func (g *ginApiBind[A, B]) Resp(resp *B, err error) {
	if err == nil {
		g.Write(resp)
	} else {
		g.Error(err)
	}
}

func (g *ginApiBind[A, B]) Error(err error) {
	//TODO implement me
}

func (g *ginApiBind[A, B]) Write(b *B) {
	//TODO implement me
}

func (g *ginApiBind[A, B]) Context() context.Context {
	return g.c
}

//func TestName(t *testing.T) {
//	//var bind ApiBind[int, int]
//	//NewRpc(bind, "", group.NewGroupClient, temp)
//
//	var c *gin.Context
//	NewRpc(NewGin[apistruct.KickGroupMemberReq, apistruct.KickGroupMemberResp](c), "", group.NewGroupClient, group.GroupClient.KickGroupMember)
//}

func KickGroupMember(c *gin.Context) {
	// 默认 全部自动
	NewRpc(NewGin[apistruct.KickGroupMemberReq, apistruct.KickGroupMemberResp](c), group.NewGroupClient, group.GroupClient.KickGroupMember).Call()
	// 可以自定义编辑请求和响应
	a := NewRpc(NewGin[apistruct.KickGroupMemberReq, apistruct.KickGroupMemberResp](c), group.NewGroupClient, group.GroupClient.KickGroupMember)
	a.Before(func(apiReq *apistruct.KickGroupMemberReq, rpcReq *group.KickGroupMemberReq, bind func() error) error {
		return bind()
	}).After(func(rpcResp *group.KickGroupMemberResp, apiResp *apistruct.KickGroupMemberResp, bind func() error) error {
		return bind()
	}).Call()
}

// NewRpc A: apiReq B: apiResp  C: rpcReq  D: rpcResp  Z: rpcClient (group.GroupClient)
func NewRpc[A, B any, C, D any, Z any](bind ApiBind[A, B], client func(conn *grpc.ClientConn) Z, rpc func(client Z, ctx context.Context, req *C, options ...grpc.CallOption) (*D, error)) *Rpc[A, B, C, D, Z] {
	return &Rpc[A, B, C, D, Z]{
		bind:   bind,
		client: client,
		rpc:    rpc,
	}
}

type Rpc[A, B any, C, D any, Z any] struct {
	bind   ApiBind[A, B]
	name   string
	client func(conn *grpc.ClientConn) Z
	rpc    func(client Z, ctx context.Context, req *C, options ...grpc.CallOption) (*D, error)
	before func(apiReq *A, rpcReq *C, bind func() error) error
	after  func(rpcResp *D, apiResp *B, bind func() error) error
}

func (a *Rpc[A, B, C, D, Z]) Name(name string) *Rpc[A, B, C, D, Z] {
	a.name = name
	return a
}

func (a *Rpc[A, B, C, D, Z]) Before(fn func(apiReq *A, rpcReq *C, bind func() error) error) *Rpc[A, B, C, D, Z] {
	a.before = fn
	return a
}

func (a *Rpc[A, B, C, D, Z]) After(fn func(rpcResp *D, apiResp *B, bind func() error) error) *Rpc[A, B, C, D, Z] {
	a.after = fn
	return a
}

func (a *Rpc[A, B, C, D, Z]) defaultCopyReq(apiReq *A, rpcReq *C) error {
	common.CopyAny(apiReq, rpcReq)
	return nil
}

func (a *Rpc[A, B, C, D, Z]) defaultCopyResp(rpcResp *D, apiResp *B) error {
	common.CopyAny(rpcResp, apiResp)
	return nil
}

func (a *Rpc[A, B, C, D, Z]) GetGrpcConn() (*grpc.ClientConn, error) {
	return nil, nil // todo
}

func (a *Rpc[A, B, C, D, Z]) execute() (*B, error) {
	var apiReq A
	if err := a.bind.Bind(&apiReq); err != nil {
		return nil, err
	}
	opID := a.bind.OperationID()
	userID, err := a.bind.OpUserID()
	if err != nil {
		return nil, err
	}
	_, _ = opID, userID
	var rpcReq C
	if a.before == nil {
		err = a.defaultCopyReq(&apiReq, &rpcReq)
	} else {
		err = a.before(&apiReq, &rpcReq, func() error { return a.defaultCopyReq(&apiReq, &rpcReq) })
	}
	if err != nil {
		return nil, err
	}
	conn, err := a.GetGrpcConn()
	if err != nil {
		return nil, err
	}
	rpcResp, err := a.rpc(a.client(conn), a.bind.Context(), &rpcReq)
	if err != nil {
		return nil, err
	}
	var apiResp B
	if a.after == nil {
		err = a.defaultCopyResp(rpcResp, &apiResp)
	} else {
		err = a.after(rpcResp, &apiResp, func() error { return a.defaultCopyResp(rpcResp, &apiResp) })
	}
	if err != nil {
		return nil, err
	}
	return &apiResp, nil
}

func (a *Rpc[A, B, C, D, Z]) Call() {
	a.bind.Resp(a.execute())
}
