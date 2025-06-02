package model

import (
	"time"

	"github.com/uptrace/bun"
)

type RequestCreatedEvent struct {
	bun.BaseModel `bun:"request_created_events"`

	Id              int       `bun:"id,pk,autoincrement"`
	TransactionHash string    `bun:"transaction_hash"`
	LogIndex        int       `bun:"log_index"`
	RequestId       string    `bun:"request_id"`
	RequestOwner    string    `bun:"request_owner"`
	Amount          int       `bun:"amount"`
	CreatedAt       time.Time `bun:"created_at"`
}

type ResponseCreatedEvent struct {
	bun.BaseModel `bun:"response_created_events"`

	Id              int       `bun:"id,pk,autoincrement"`
	TransactionHash string    `bun:"transaction_hash"`
	LogIndex        int       `bun:"log_index"`
	RequestId       string    `bun:"request_id"`
	PrizesId        []int64   `bun:"prizes_id,array"`
	CreatedAt       time.Time `bun:"created_at"`
}

type TempRequestCreatedEvent struct {
	bun.BaseModel `bun:"table=temp_request_created_events"`

	Id              int       `bun:"id,pk,autoincrement"`
	BlockNumber     int       `bun:"block_number"`
	BlockHash       string    `bun:"block_hash"`
	TransactionHash string    `bun:"transaction_hash"`
	LogIndex        int       `bun:"log_index"`
	RequestId       string    `bun:"request_id"`
	RequestOwner    string    `bun:"request_owner"`
	Amount          int       `bun:"amount"`
	CreatedAt       time.Time `bun:"created_at"`
}
