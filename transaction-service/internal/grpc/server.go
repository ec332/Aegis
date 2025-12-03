package grpc

import (
    "context"
    "fmt"
    "strings"
    "time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

    "github.com/aegis/proto/gen/market"
    "github.com/aegis/proto/gen/wallet"
    "github.com/aegis/proto/gen/transaction"
    "transaction-service/internal/model"
    "transaction-service/internal/service"
    "google.golang.org/grpc/metadata"
)

type TransactionGRPCServer struct {
    transaction.UnimplementedTransactionServiceServer
    svc          *service.TransactionService
    marketClient market.MarketServiceClient
    walletClient wallet.WalletServiceClient
    defaultCurrency string
    logger       *zap.Logger
}

func NewTransactionGRPCServer(svc *service.TransactionService, logger *zap.Logger, marketClient market.MarketServiceClient, walletClient wallet.WalletServiceClient, defaultCurrency string) *TransactionGRPCServer {
    return &TransactionGRPCServer{
        svc:          svc,
        logger:       logger,
        marketClient: marketClient,
        walletClient: walletClient,
        defaultCurrency: strings.TrimSpace(defaultCurrency),
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

	// Compute amount = price_per_share * number_of_shares
	amount := transactionModel.PricePerShare.Mul(transactionModel.NumberOfShares).InexactFloat64()

	// If SELL, validate user has sufficient shares for this market option
	if req.TransactionType == transaction.TransactionType_SELL {
		userTxs, err := s.svc.FindByUserID(ctx, uuid.MustParse(req.UserId))
		if err != nil {
			s.logger.Error("failed to load user transactions", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to validate holdings")
		}
		var held int64
		for _, t := range userTxs {
			if t.MarketID.String() == req.MarketId && t.OptionID.String() == req.OptionId {
				shares := t.NumberOfShares.IntPart()
				if strings.EqualFold(t.TransactionType, "SELL") {
					held -= shares
				} else {
					held += shares
				}
			}
		}
		if int64(req.NumberOfShares) > held {
			return nil, status.Error(codes.FailedPrecondition, "insufficient shares")
		}
	}

    // Resolve user's wallet account by currency
    currency := s.defaultCurrency
    if currency == "" { currency = "USD" }
    // propagate authorization metadata to wallet service
    var ctxWithAuth = ctx
    if md, ok := metadata.FromIncomingContext(ctx); ok {
        ctxWithAuth = metadata.NewOutgoingContext(ctx, md)
    }
    accResp, err := s.walletClient.GetWalletAccountByUserID(ctxWithAuth, &wallet.GetWalletAccountByUserIDRequest{UserId: req.UserId, Currency: currency})
    if err != nil {
        st, _ := status.FromError(err)
        if st.Code() == codes.NotFound {
            return nil, status.Error(codes.NotFound, "wallet account not found")
        }
        s.logger.Error("failed to get wallet account", zap.String("user_id", req.UserId), zap.Error(err))
        return nil, status.Error(codes.Internal, "wallet service error")
    }
    accountID := accResp.Account.Id

    // Perform wallet update prior to persisting transaction
    var walletErr error
    refID := transactionModel.ID.String()
    switch req.TransactionType {
    case transaction.TransactionType_BUY:
        _, walletErr = s.walletClient.Withdrawal(ctxWithAuth, &wallet.WithdrawalRequest{AccountId: accountID, Amount: amount, ReferenceId: refID})
    case transaction.TransactionType_SELL:
        _, walletErr = s.walletClient.Deposit(ctxWithAuth, &wallet.DepositRequest{AccountId: accountID, Amount: amount, ReferenceId: refID})
    default:
        return nil, status.Error(codes.InvalidArgument, "invalid transaction_type")
    }
    if walletErr != nil {
        st, _ := status.FromError(walletErr)
        if st.Code() == codes.FailedPrecondition || strings.Contains(strings.ToLower(walletErr.Error()), "insufficient funds") {
            return nil, status.Error(codes.FailedPrecondition, "insufficient funds")
        }
        s.logger.Error("wallet operation failed", zap.String("account_id", accountID), zap.Error(walletErr))
        return nil, status.Error(codes.Internal, "wallet operation failed")
    }

    // Persist transaction
    created, err := s.svc.Create(ctx, transactionModel)
    if err != nil {
        s.logger.Error("failed to create transaction after wallet update", zap.Error(err))
        // Compensating reversal
        if req.TransactionType == transaction.TransactionType_BUY {
            if _, rerr := s.walletClient.Deposit(ctxWithAuth, &wallet.DepositRequest{AccountId: accountID, Amount: amount, ReferenceId: refID}); rerr != nil {
                s.logger.Error("compensation deposit failed", zap.Error(rerr))
            }
        } else if req.TransactionType == transaction.TransactionType_SELL {
            if _, rerr := s.walletClient.Withdrawal(ctxWithAuth, &wallet.WithdrawalRequest{AccountId: accountID, Amount: amount, ReferenceId: refID}); rerr != nil {
                s.logger.Error("compensation withdrawal failed", zap.Error(rerr))
            }
        }
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

    total := len(transactions)
    limit := int(req.GetLimit())
    offset := int(req.GetOffset())
    if limit <= 0 { limit = total }
    if offset < 0 { offset = 0 }
    start := offset
    if start > total { start = total }
    end := start + limit
    if end > total { end = total }
    page := transactions[start:end]

    protoTransactions := make([]*transaction.Transaction, len(page))
    for i, t := range page {
        protoTransactions[i] = &transaction.Transaction{
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
        Total:        int32(total),
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
