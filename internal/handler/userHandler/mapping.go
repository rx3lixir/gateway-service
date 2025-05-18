package userhandler

import (
	pbUser "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/user"
)

func toPBUserReq(u UserReq) *pbUser.UserReq {
	return &pbUser.UserReq{
		Name:     u.Name,
		Email:    u.Email,
		Password: u.Password,
		IsAdmin:  u.IsAdmin,
	}
}

func toUserRes(u *pbUser.UserRes) UserRes {
	return UserRes{
		Name:    u.Name,
		Email:   u.Email,
		IsAdmin: u.IsAdmin,
	}
}
