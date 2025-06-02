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
