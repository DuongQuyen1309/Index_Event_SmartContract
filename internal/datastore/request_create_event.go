package datastore

import (
	"context"
	"time"

	token "github.com/DuongQuyen1309/indexevent"
	"github.com/DuongQuyen1309/indexevent/internal/db"
	"github.com/DuongQuyen1309/indexevent/internal/model"
	"github.com/uptrace/bun"
)

func CreateRequestCreatedEvent(DB *bun.DB) error {
	ctx := context.Background()
	_, err := DB.NewCreateTable().Model((*model.RequestCreatedEvent)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}
	_, err = DB.NewCreateIndex().Model((*model.RequestCreatedEvent)(nil)).
		Index("idx_transaction_hash").Column("transaction_hash").Unique().IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}
	_, err = DB.NewCreateIndex().Model((*model.RequestCreatedEvent)(nil)).
		Index("idx_request_owner").Column("request_owner").IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func InsertResquestCreatedDB(log *token.WheelRequestCreated, requestOwner string, timestamp time.Time) error {
	_, err := db.DB.NewInsert().Model(&model.RequestCreatedEvent{
		TransactionHash: log.Raw.TxHash.String(),
		LogIndex:        int(log.Raw.Index),
		RequestId:       log.RequestId.String(),
		RequestOwner:    requestOwner,
		Amount:          int(log.Amount.Int64()),
		CreatedAt:       timestamp,
	}).Exec(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func GetTotalTurnAmountOfUser(address string, c context.Context) (int, error) {
	var amountSum int
	err := db.DB.NewSelect().Model((*model.RequestCreatedEvent)(nil)).
		ColumnExpr("SUM(amount)").Where("request_owner = ?", address).Scan(c, &amountSum)
	if err != nil {
		return -1, err
	}
	return amountSum, nil
}

func GetTurnsRequestsOfUser(address string, limit int, offset int, c context.Context) ([]model.RequestCreatedEvent, error) {
	var turns []model.RequestCreatedEvent
	err := db.DB.NewSelect().Model(&turns).
		Where("request_owner = ?", address).
		Offset(offset).Limit(limit).
		Scan(c)
	if err != nil {
		return nil, err
	}
	return turns, nil
}

func GetTurnById(hash string, c context.Context) (model.RequestCreatedEvent, error) {
	var turn model.RequestCreatedEvent
	err := db.DB.NewSelect().Model(&turn).
		Where("transaction_hash = ?", hash).
		Scan(c)
	if err != nil {
		return model.RequestCreatedEvent{}, err
	}
	return turn, nil
}

func GetRequestIDByHash(hash string, c context.Context) (string, error) {
	var requestId string
	err := db.DB.NewSelect().Model((*model.RequestCreatedEvent)(nil)).
		Column("request_id").
		Where("transaction_hash = ?", hash).
		Scan(c, &requestId)
	if err != nil {
		return "", err
	}
	return requestId, nil
}
