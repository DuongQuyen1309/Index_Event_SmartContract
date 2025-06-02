package datastore

import (
	"context"
	"time"

	token "github.com/DuongQuyen1309/indexevent"
	"github.com/DuongQuyen1309/indexevent/internal/db"
	"github.com/DuongQuyen1309/indexevent/internal/model"
	"github.com/uptrace/bun"
)

func CreateResponseCreatedEvent(DB *bun.DB) error {
	ctx := context.Background()
	_, err := DB.NewCreateTable().Model((*model.ResponseCreatedEvent)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}
	_, err = DB.NewCreateIndex().Model((*model.ResponseCreatedEvent)(nil)).
		Index("idx_transaction_hash").Column("transaction_hash").Unique().IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func InsertResponseCreatedDB(log *token.WheelResponseCreated, prizeIds []int64, requestOwner string, timestamp time.Time) error {
	_, err := db.DB.NewInsert().Model(&model.ResponseCreatedEvent{
		TransactionHash: log.Raw.TxHash.String(),
		LogIndex:        int(log.Raw.Index),
		RequestId:       log.RequestId.String(),
		PrizesId:        prizeIds,
		CreatedAt:       timestamp,
	}).Exec(context.Background())
	if err != nil {
		return err
	}
	return nil
}
