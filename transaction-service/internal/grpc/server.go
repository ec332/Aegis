package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/aegis/proto/gen/market"
	"github.com/aegis/proto/gen/transaction"
	"transaction-service/internal/model"
	"transaction-service/internal/service"
)

type TransactionGRPCServer struct {
	transaction.UnimplementedTransactionServiceServer
	svc          *service.TransactionService
	marketClient market.MarketServiceClient
	logger       *zap.Logger
}

func NewTransactionGRPCServer(svc *service.TransactionService, logger *zap.Logger, marketClient market.MarketServiceClient) *TransactionGRPCServer {
	return &TransactionGRPCServer{
		svc:          svc,
		logger:       logger,
		marketClient: marketClient,
	}
}

func (s *TransactionGRPCServer) CreateTransaction(ctx context.Context, req *transaction.TransactionRequest) (*transaction.TransactionResponse, error) {
	// Input validation
	if req.MarketId == "" || req.OptionId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id, option_id, and user_id are required")
	}
	if req.NumberOfShares <= 0 {
		return nil, status.Error(codes.InvalidArgument, "number_of_shares must be positive")
	}
	if req.PricePerShare < 0 || req.PricePerShare > 1 {
		return nil, status.Error(codes.InvalidArgument, "price_per_share must be between 0 and 1")
	}

	// Validate market status before allowing trade
	marketResp, err := s.marketClient.GetMarket(ctx, &market.GetMarketRequest{Id: req.MarketId})
	if err != nil {
		s.logger.Error("failed to get market status", zap.String("market_id", req.MarketId), zap.Error(err))
		return nil, status.Error(codes.Unavailable, "market service unavailable")
	}
	if marketResp.Market.Status != "active" {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("market is not active (current status: %s)", marketResp.Market.Status))
	}

	// Validate option belongs to market
	optsResp, err := s.marketClient.GetMarketOptions(ctx, &market.GetMarketOptionsRequest{MarketId: req.MarketId})
	if err != nil {
		s.logger.Error("failed to get market options", zap.String("market_id", req.MarketId), zap.Error(err))
		return nil, status.Error(codes.Unavailable, "market service unavailable")
	}
	found := false
	var currentPrice float64
	for _, opt := range optsResp.Options {
		if opt.Id == req.OptionId {
			found = true
			currentPrice = opt.CurrentPrice
			break
		}
	}
	if !found {
		return nil, status.Error(codes.InvalidArgument, "option does not belong to market")
	}

	// Validate price is close to current LMSR price (±2% tolerance)
	tol := 0.02
	if currentPrice > 0 && (req.PricePerShare < currentPrice*(1-tol) || req.PricePerShare > currentPrice*(1+tol)) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("price_per_share deviates from current price (%.4f vs %.4f)", req.PricePerShare, currentPrice))
	}

	// Create transaction model
	transactionModel := model.Transaction{
		ID:              uuid.New(),
		UserID:          uuid.MustParse(req.UserId),
		MarketID:        uuid.MustParse(req.MarketId),
		OptionID:        uuid.MustParse(req.OptionId),
		TransactionType: req.TransactionType.String(),
		NumberOfShares:  decimal.NewFromInt(int64(req.NumberOfShares)),
		PricePerShare:   decimal.NewFromFloat(req.PricePerShare),
		CreatedAt:       time.Now().UTC(),
	}

	// Insert into database
	created, err := s.svc.Create(ctx, transactionModel)
	if err != nil {
		s.logger.Error("failed to create transaction", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create transaction")
	}

	return &transaction.TransactionResponse{
		Id:        created.ID.String(),
		CreatedAt: timestamppb.New(created.CreatedAt),
	}, nil
}

func (s *TransactionGRPCServer) GetTransactions(ctx context.Context, req *transaction.GetTransactionsRequest) (*transaction.GetTransactionsResponse, error) {
	var transactions []model.Transaction
	var err error

	// Apply filters if provided
	if req.UserId != nil && *req.UserId != "" {
		transactions, err = s.svc.FindByUserID(ctx, uuid.MustParse(*req.UserId))
	} else if req.MarketId != nil && *req.MarketId != "" {
		transactions, err = s.svc.FindByMarketID(ctx, uuid.MustParse(*req.MarketId))
	} else {
		transactions, err = s.svc.FindAll(ctx)
	}

	if err != nil {
		s.logger.Error("failed to get transactions", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get transactions")
	}

	// Convert to protobuf response
	protoTransactions := make([]*transaction.TransactionResponse, len(transactions))
	for i, t := range transactions {
		protoTransactions[i] = &transaction.TransactionResponse{
			Id:              t.ID.String(),
			MarketId:        t.MarketID.String(),
			OptionId:        t.OptionID.String(),
			UserId:          t.UserID.String(),
			TransactionType: mapTransactionType(t.TransactionType),
			NumberOfShares:  int32(t.NumberOfShares.IntPart()),
			PricePerShare:   t.PricePerShare.InexactFloat64(),
			CreatedAt:       timestamppb.New(t.CreatedAt),
		}
	}

	return &transaction.GetTransactionsResponse{
		Transactions: protoTransactions,
	}, nil
}

func mapTransactionType(transactionType string) transaction.TransactionType {
	switch transactionType {
	case "BUY":
		return transaction.TransactionType_BUY
	case "SELL":
		return transaction.TransactionType_SELL
	default:
		return transaction.TransactionType_BUY // Default to BUY
	}
}