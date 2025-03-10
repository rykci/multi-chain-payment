package main

import (
	"os"
	"path/filepath"
	"payment-bridge/blockchain/browsersync"
	"payment-bridge/common/constants"
	"payment-bridge/config"
	"payment-bridge/database"
	"payment-bridge/models"
	"payment-bridge/routers"
	"payment-bridge/routers/billing"
	"payment-bridge/routers/common"
	"payment-bridge/routers/storage"
	"payment-bridge/scheduler"
	"time"

	"github.com/filswan/go-swan-lib/logs"

	"github.com/gin-gonic/gin"
	cors "github.com/itsjamie/gin-cors"
	"github.com/joho/godotenv"
)

func main() {
	//logs.GetLogger().Info(errorinfo.GetErrMsg("abc"))
	//logs.GetLogger().Info(errorinfo.GetErrMsg(errorinfo.GET_EVENT_FROM_DB_ERROR_CODE))
	LoadEnv()

	db := database.Init()
	defer database.CloseDB(db)

	browsersync.Init()

	models.RunAllTheScan()

	scheduler.InitScheduler()
	scheduler.CreateTaskScheduler()
	scheduler.SendDealScheduler()

	createGinServer()
}

func createGinServer() {
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
	common.HostManager(v1.Group(constants.URL_HOST_GET_COMMON))
	routers.EventLogManager(v1.Group(constants.URL_EVENT_PREFIX))
	billing.BillingManager(v1.Group(constants.URL_BILLING_PREFIX))
	storage.SendDealManager(v1.Group(constants.URL_STORAGE_PREFIX))

	err := r.Run(":" + config.GetConfig().Port)
	if err != nil {
		logs.GetLogger().Fatal(err)
	}
}

func LoadEnv() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		logs.GetLogger().Fatal("Cannot get home directory.")
	}

	envFile := filepath.Join(homedir, ".swan/mcp/.env")
	err = godotenv.Load(envFile)
	if err != nil {
		logs.GetLogger().Fatal(err)
	}

	keyName := "privateKeyOnPolygon"
	logs.GetLogger().Info(keyName, ":", os.Getenv(keyName))
}
