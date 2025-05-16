package authhandler

import (
	"time"

	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HTTPSessionToProtoSession преборазует Session из HTTP в формат Proto
func HTTPSessionToProtoSession(session *Session) *pbAuth.SessionReq {
	if session == nil {
		return nil
	}

	var expiresAt *timestamppb.Timestamp

	if !session.ExpiresAt.IsZero() {
		expiresAt = timestamppb.New(session.ExpiresAt)
	}

	return &pbAuth.SessionReq{
		Id:           session.ID,
		UserEmail:    session.UserEmail,
		RefreshToken: session.RefreshToken,
		IsRevoked:    session.IsRevoked,
		ExpiresAt:    expiresAt,
	}
}

// ProtoSessionToHTTPSession преобразует Session из Proto в формат HTTP
func ProtoSessionToHTTPSession(protoSession *pbAuth.SessionRes) *Session {
	if protoSession == nil {
		return nil
	}

	var expiresAt time.Time
	if protoSession.ExpiresAt != nil {
		expiresAt = protoSession.ExpiresAt.AsTime()
	}

	return &Session{
		ID:           protoSession.Id,
		UserEmail:    protoSession.UserEmail,
		RefreshToken: protoSession.RefreshToken,
		IsRevoked:    protoSession.IsRevoked,
		ExpiresAt:    expiresAt,
	}
}
