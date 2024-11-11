package handlers

import (
	"context"
	"fmt"
	"gateway/external"
	pb "gateway/protos/user_service"
)

type HandlerInterface interface {
	GetUsers(ctx context.Context, userID string) (interface{}, error)
}

type userHandler struct {
	us external.UserServiceInterface
}

func UserHandler() HandlerInterface {
	us := external.UserService()
	return &userHandler{us}
}

func (h *userHandler) GetUsers(ctx context.Context, userID string) (interface{}, error) {
	fmt.Println("Get Users handler called for: ")

	users, userErr := h.us.GetUsers(ctx, &pb.GetUsersRequest{UserIds: []string{}, TenantId: ""})
	if userErr != nil {
		return nil, userErr
	}

	return users.Data, nil
}
