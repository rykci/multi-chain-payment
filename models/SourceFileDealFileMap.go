package models

import (
	"payment-bridge/database"

	"github.com/filswan/go-swan-lib/logs"
)

type SourceFileDealFileMap struct {
	SourceFileId int64 `json:"source_file_id"`
	FileIndex    int   `json:"file_index"`
	DealFileId   int64 `json:"deal_file_id"`
	CreateAt     int64 `json:"create_at"`
	UpdateAt     int64 `json:"update_at"`
}

//condition :&models.SourceFileDealFileMap{DealCid: "123"}
//updateFields: map[string]interface{}{"processing_time": taskT.ProcessingTime, "worker_reward": taskT.WorkerReward}
func UpdateSourceFileDealFileMap(whereCondition interface{}, updateFields interface{}) error {
	db := database.GetDB()
	sdfMap := SourceFileDealFileMap{}
	err := db.Model(&sdfMap).Where(whereCondition).Update(updateFields).Error
	if err != nil {
		logs.GetLogger().Error(err)
	}
	return err
}
