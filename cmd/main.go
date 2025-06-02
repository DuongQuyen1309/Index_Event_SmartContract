package main

import (
	"fmt"

	"github.com/DuongQuyen1309/indexevent/internal/datastore"
	"github.com/DuongQuyen1309/indexevent/internal/db"
	"github.com/DuongQuyen1309/indexevent/internal/service"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error load env file", err)
	}
}
func main() {
	db.ConnectDB()
	datastore.CreateRequestCreatedEvent(db.DB)
	datastore.CreateResponseCreatedEvent(db.DB)
	if err := service.IndexEvent(); err != nil {
		fmt.Println("Error index event", err)
		return
	}
}
