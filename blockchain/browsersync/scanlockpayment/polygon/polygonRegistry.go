package polygon

import (
	"payment-bridge/models"
)

func Init() {
	models.Register(&models.RegInfo{
		Network:                           "Polygon",
		CoinName:                          "MATIC",
		ScanEventFromChainAndSaveDataToDb: ScanEventFromChainAndSaveDataToDb,
	})
}

func ScanEventFromChainAndSaveDataToDb() {
	ScanEventFromChainAndSaveDataToDbForPolygon()
}
