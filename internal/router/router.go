package router

import (
	"github.com/DuongQuyen1309/indexevent/internal/handler"
	"github.com/gin-gonic/gin"
)

func SetupRouer() *gin.Engine {
	router := gin.Default()
	router.GET("/user/:address/turn-amount", handler.GetTotalTurnAmountOfUser)
	router.GET("/user/:address/turn-requests", handler.GetTurnsRequestsOfUser)
	router.GET("/turn-request/:hash", handler.GetTurnRequestByHash)
	router.GET("/turn-request/:hash/prizes", handler.GetPrizesOfHash)
	return router

}
