package webapi

import (
	"encoding/json"
	"net/http"

	userTxPb "github.com/Sugar-pack/users-manager/pkg/generated/distributedtx"
	userPb "github.com/Sugar-pack/users-manager/pkg/generated/users"
	"github.com/Sugar-pack/users-manager/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderPb "github.com/Sugar-pack/orders-manager/pkg/pb"
)

type Handler struct {
	UserClient    userPb.UsersClient
	UserTxClient  userTxPb.DistributedTxServiceClient
	OrderClient   orderPb.OrdersManagerServiceClient
	OrderTxClient orderPb.TnxConfirmingServiceClient
	BgResponses   map[string][]byte
}

func NewHandler(userConn *grpc.ClientConn, orderConn *grpc.ClientConn) *Handler {
	userClient := userPb.NewUsersClient(userConn)
	userTxClient := userTxPb.NewDistributedTxServiceClient(userConn)
	orderClient := orderPb.NewOrdersManagerServiceClient(orderConn)
	orderTxClient := orderPb.NewTnxConfirmingServiceClient(orderConn)
	bgResponses := make(map[string][]byte)

	return &Handler{
		UserClient:    userClient,
		UserTxClient:  userTxClient,
		OrderClient:   orderClient,
		OrderTxClient: orderTxClient,
		BgResponses:   bgResponses,
	}
}

type Message struct {
	Name  string `json:"name" example:"John"`
	Label string `json:"label" example:"Bag"`
}

// SendMessage godoc
// @Summary      Send message
// @Description  Put message with name and label to DB by 2pc transactions
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        message body Message true "Message"
// @Success      200  {string} string	"ok"
// @Failure      400  {string} string	"message decode error"
// @Failure      500  {string} string	"server error"
// @Router       /send [post].
func (h *Handler) SendMessage(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	logger := logging.FromContext(ctx)
	logger.Info("SendMessage")

	messageFromReq := &Message{}
	err := json.NewDecoder(request.Body).Decode(messageFromReq)
	if err != nil {
		logger.WithError(err).Error("Error while decoding user from request")
		BadRequest(ctx, writer, "Error while decoding user from request")

		return
	}
	user := &userPb.NewUser{
		Name: messageFromReq.Name,
	}
	createUser, err := h.UserClient.CreateUser(ctx, user)
	if err != nil {
		logger.WithError(err).Error("Error while creating user")
		InternalError(ctx, writer, "Error while creating user")

		return
	}
	userTx := createUser.GetTxId()

	order := &orderPb.Order{
		UserId:    createUser.GetId(),
		Label:     messageFromReq.Label,
		CreatedAt: timestamppb.Now(),
	}
	insertOrder, err := h.OrderClient.InsertOrder(ctx, order)
	if err != nil {
		logger.WithError(err).Error("Error while creating order")
		userRollback := &userTxPb.TxToRollback{TxId: userTx}
		_, errRollback := h.UserTxClient.Rollback(ctx, userRollback)
		if errRollback != nil {
			logger.WithError(errRollback).Error("Error while rollback user")
			InternalError(ctx, writer, "Error while rollback user")

			return
		}
		InternalError(ctx, writer, "Error while creating order. User rollback success")

		return
	}
	orderTx := insertOrder.GetTnx()

	userCommit := &userTxPb.TxToCommit{TxId: userTx}
	_, errCommit := h.UserTxClient.Commit(ctx, userCommit)
	if errCommit != nil {
		logger.WithError(errCommit).Error("Error while commit user")
		orderRollback := &orderPb.Confirmation{
			Tnx:    orderTx,
			Commit: false,
		}
		_, errRollback := h.OrderTxClient.SendConfirmation(ctx, orderRollback)
		if errRollback != nil {
			logger.WithError(errRollback).Error("Error while rollback order. User commit success")
			InternalError(ctx, writer, "Error while rollback order. User commit success")

			return
		}
		InternalError(ctx, writer, "Error while commit user")

		return
	}

	orderCommit := &orderPb.Confirmation{
		Tnx:    orderTx,
		Commit: true,
	}
	_, errCommitOrder := h.OrderTxClient.SendConfirmation(ctx, orderCommit)
	if errCommitOrder != nil {
		logger.WithError(errCommitOrder).Error("Error while commit order")
		InternalError(ctx, writer, "Error while commit order. User commit success")

		return
	}

	StatusOk(ctx, writer, "User and order created")
}
