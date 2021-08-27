package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	cors "github.com/itsjamie/gin-cors"
	"github.com/joho/godotenv"
	"os"
	"payment-bridge/blockchain/browsersync/nbai"
	"payment-bridge/blockchain/initclient/bscclient"
	"payment-bridge/blockchain/initclient/goerliclient"
	"payment-bridge/blockchain/initclient/nbaiclient"
	"payment-bridge/blockchain/initclient/polygonclient"
	"payment-bridge/common/constants"
	"payment-bridge/config"
	"payment-bridge/database"
	"payment-bridge/logs"
	"payment-bridge/routers"
	"time"
)

func main() {
	initMethod()

	LoadEnv()

	// init database
	db := database.Init()
	//polygon.ScanPolygonEventFromChainAndSaveEventLogData(1,17785986)
	//nbai.ListenForReceiptLogTillExit2()
	nbai.NbaiBlockBrowserSyncAndEventLogsSync()
	//nbai.ScanNbaiEventFromChainAndSaveEventLogData(6914904, 6915904)
	//nbai.ChangeNbaiToBnb(nil)
	//go polygon.PolygonBlockBrowserSyncAndEventLogsSync()

	//go goerli.GoerliBlockBrowserSyncAndEventLogsSync()

	defer func() {
		err := db.Close()
		if err != nil {
			logs.GetLogger().Error(err)
		}
	}()

	r := gin.Default()
	r.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, PUT, POST, DELETE",
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          50 * time.Second,
		Credentials:     true,
		ValidateHeaders: false,
	}))

	v1 := r.Group("/api/v1")
	routers.EventLogManager(v1.Group(constants.URL_EVENT_PREFIX))

	err := r.Run(":" + config.GetConfig().Port)
	if err != nil {
		logs.GetLogger().Fatal(err)
	}

}

func initMethod() string {
	config.InitConfig("")
	goerliclient.ClientInit()
	polygonclient.ClientInit()
	nbaiclient.ClientInit()
	bscclient.ClientInit()
	return ""
}

func LoadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		logs.GetLogger().Error(err)
	}
	fmt.Println("name: ", os.Getenv("privateKey"))
}
