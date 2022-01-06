package storage

import (
	"context"
	"mime/multipart"
	"os"
	"path/filepath"
	"payment-bridge/blockchain/browsersync/scanlockpayment/polygon"
	"payment-bridge/common"
	"payment-bridge/common/constants"
	"payment-bridge/common/utils"
	"payment-bridge/config"
	"payment-bridge/database"
	"payment-bridge/logs"
	"payment-bridge/models"
	"payment-bridge/on-chain/goBind"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	common2 "github.com/ethereum/go-ethereum/common"

	"github.com/filswan/go-swan-client/command"
	"github.com/filswan/go-swan-lib/client/ipfs"

	libmodel "github.com/filswan/go-swan-lib/model"
	libutils "github.com/filswan/go-swan-lib/utils"
	"github.com/gin-gonic/gin"
)

func SaveFileAndCreateCarAndUploadToIPFSAndSaveDb(c *gin.Context, srcFile *multipart.FileHeader, duration int, walletAddress string) (string, string, int, error) {
	temDirDeal := config.GetConfig().SwanTask.DirDeal

	logs.GetLogger().Info("temp dir is ", temDirDeal)
	needPay := 0
	homedir, err := os.UserHomeDir()
	if err != nil {
		logs.GetLogger().Error("Cannot get home directory.")
		return "", "", needPay, err
	}
	temDirDeal = filepath.Join(homedir, temDirDeal[2:])

	err = libutils.CreateDir(temDirDeal)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	timeStr := time.Now().Format("20060102_150405")
	temDirDeal = filepath.Join(temDirDeal, timeStr)
	srcDir := filepath.Join(temDirDeal, "src")
	carDir := filepath.Join(temDirDeal, "car")
	err = libutils.CreateDir(srcDir)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	err = libutils.CreateDir(carDir)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	srcFilepath := filepath.Join(srcDir, srcFile.Filename)
	logs.GetLogger().Info("save your file to ", srcFilepath)
	err = c.SaveUploadedFile(srcFile, srcFilepath)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}
	logs.GetLogger().Info("car files created in ", carDir)

	cmdCar := &command.CmdCar{
		LotusClientApiUrl:      config.GetConfig().Lotus.ClientApiUrl,
		LotusClientAccessToken: config.GetConfig().Lotus.ClientAccessToken,
		OutputDir:              carDir,
		InputDir:               srcDir,
		GenerateMd5:            false,
	}
	fileList, err := cmdCar.CreateCarFiles()
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}
	logs.GetLogger().Info("car files created in ", carDir, "payload_cid=", fileList[0].PayloadCid)

	uploadUrl := utils.UrlJoin(config.GetConfig().IpfsServer.UploadUrlPrefix, "api/v0/add?stream-channels=true&pin=true")
	ipfsFileHash, err := ipfs.IpfsUploadFileByWebApi(uploadUrl, srcFilepath)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	filePathInIpfs := config.GetConfig().IpfsServer.DownloadUrlPrefix + constants.IPFS_URL_PREFIX_BEFORE_HASH + *ipfsFileHash
	lockPaymentList, err := models.FindEventLockPayment(&models.EventLockPayment{PayloadCid: fileList[0].PayloadCid}, "create_at desc", "10", "0")
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	sourceAndDealFileList, err := GetSourceFileAndDealFileInfoByPayloadCid(fileList[0].PayloadCid)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}
	if len(sourceAndDealFileList) > 0 {
		if sourceAndDealFileList[0].WalletAddress == walletAddress {
			if len(lockPaymentList) > 0 {
				needPay = 1
				return fileList[0].PayloadCid, sourceAndDealFileList[0].IpfsUrl, needPay, nil
			} else {
				needPay = 2
				return fileList[0].PayloadCid, sourceAndDealFileList[0].IpfsUrl, needPay, nil
			}
		} else {
			if len(lockPaymentList) > 0 {
				needPay = 3
				sourceFile, err := saveSourceFileToDB(srcFile, srcFilepath, filePathInIpfs, walletAddress)
				if err != nil {
					logs.GetLogger().Error(err)
					return "", "", needPay, err
				}
				filepMap := new(models.SourceFileDealFileMap)
				filepMap.SourceFileId = sourceFile.ID
				filepMap.DealFileId = sourceAndDealFileList[0].ID
				filepMap.FileIndex = 0
				nowTime := strconv.FormatInt(utils.GetEpochInMillis(), 10)
				filepMap.CreateAt = nowTime
				filepMap.UpdateAt = nowTime
				err = database.SaveOne(filepMap)
				if err != nil {
					logs.GetLogger().Error(err)
					return "", "", needPay, err
				}
				return fileList[0].PayloadCid, sourceAndDealFileList[0].IpfsUrl, needPay, nil
			} else {
				needPay = 4
				sourceFile, err := saveSourceFileToDB(srcFile, srcFilepath, filePathInIpfs, walletAddress)
				if err != nil {
					logs.GetLogger().Error(err)
					return "", "", needPay, err
				}
				err = saveDealFileAndMapRelation(fileList, sourceFile, duration)
				if err != nil {
					logs.GetLogger().Error(err)
					return "", "", needPay, err
				}
				return fileList[0].PayloadCid, filePathInIpfs, needPay, nil
			}
		}
	} else {
		sourceFile, err := saveSourceFileToDB(srcFile, srcFilepath, filePathInIpfs, walletAddress)
		if err != nil {
			logs.GetLogger().Error(err)
			return "", "", needPay, err
		}
		err = saveDealFileAndMapRelation(fileList, sourceFile, duration)
		if err != nil {
			logs.GetLogger().Error(err)
			return "", "", needPay, err
		}
		return fileList[0].PayloadCid, filePathInIpfs, needPay, nil
	}
}

func saveSourceFileToDB(srcFile *multipart.FileHeader, srcFilepath string, filePathInIpfs, walletAddress string) (*models.SourceFile, error) {
	sourceFile := new(models.SourceFile)
	sourceFile.FileName = srcFile.Filename
	sourceFile.FileSize = strconv.FormatInt(srcFile.Size, 10)
	sourceFile.ResourceUri = srcFilepath
	sourceFile.CreateAt = strconv.FormatInt(utils.GetEpochInMillis(), 10)
	sourceFile.IpfsUrl = filePathInIpfs
	sourceFile.PinStatus = constants.IPFS_File_PINNED_STATUS
	sourceFile.WalletAddress = walletAddress
	err := database.SaveOne(sourceFile)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return sourceFile, nil
}

func GetSourceFileAndDealFileInfoByPayloadCid(payloadCid string) ([]*SourceFileAndDealFileInfo, error) {
	sql := "select s.wallet_address,s.ipfs_url,s.file_name,d.id,d.payload_cid,d.deal_cid,d.deal_id,d.lock_payment_status,s.create_at from source_file s,source_file_deal_file_map m,deal_file d " +
		" where s.id = m.source_file_id and m.deal_file_id = d.id and d.payload_cid='" + payloadCid + "'"
	var results []*SourceFileAndDealFileInfo
	err := database.GetDB().Raw(sql).Order("create_at desc").Limit(10).Offset(0).Order("create_at desc").Scan(&results).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return results, nil
}

func saveDealFileAndMapRelation(fileInfoList []*libmodel.FileDesc, sourceFile *models.SourceFile, duration int) error {
	currentTime := utils.GetEpochInMillis()
	dealFile := new(models.DealFile)
	dealFile.CarFileName = fileInfoList[0].CarFileName
	dealFile.CarFilePath = fileInfoList[0].CarFilePath
	dealFile.CarFileSize = fileInfoList[0].CarFileSize
	dealFile.CarMd5 = fileInfoList[0].CarFileMd5
	dealFile.PayloadCid = fileInfoList[0].PayloadCid
	dealFile.PieceCid = fileInfoList[0].PieceCid
	dealFile.SourceFilePath = sourceFile.ResourceUri
	dealFile.DealCid = fileInfoList[0].PayloadCid
	dealFile.CreateAt = strconv.FormatInt(currentTime, 10)
	dealFile.UpdateAt = strconv.FormatInt(currentTime, 10)
	dealFile.Duration = duration
	dealFile.LockPaymentStatus = constants.LOCK_PAYMENT_STATUS_WAITING
	dealFile.IsDeleted = utils.GetBoolPointer(false)
	err := database.SaveOne(dealFile)
	if err != nil {
		logs.GetLogger().Error(err)
		return err
	}

	filepMap := new(models.SourceFileDealFileMap)
	filepMap.SourceFileId = sourceFile.ID
	filepMap.DealFileId = dealFile.ID
	filepMap.FileIndex = 0
	filepMap.CreateAt = strconv.FormatInt(currentTime, 10)
	filepMap.UpdateAt = strconv.FormatInt(currentTime, 10)
	err = database.SaveOne(filepMap)
	if err != nil {
		logs.GetLogger().Error(err)
		return err
	}
	return nil
}

func GetSourceFileAndDealFileInfo(limit, offset string, walletAddress string, payloadCid, fileName string) ([]*SourceFileAndDealFileInfo, error) {
	sql := "select s.file_name,s.file_size,s.pin_status,s.create_at,df.miner_fid,df.payload_cid,df.deal_cid,df.deal_id,df.piece_cid,df.deal_status,df.lock_payment_status as status,df.duration from  source_file s " +
		" inner join source_file_deal_file_map sfdfm on s.id = sfdfm.source_file_id" +
		" inner join deal_file df on sfdfm.deal_file_id = df.id and wallet_address='" + walletAddress + "' "
	if strings.Trim(payloadCid, " ") != "" {
		sql = sql + " and df.payload_cid='" + payloadCid + "'"
	}
	if strings.Trim(fileName, " ") != "" {
		sql = sql + " and s.file_name like '%" + fileName + "%'"
	}
	var results []*SourceFileAndDealFileInfo
	err := database.GetDB().Raw(sql).Order("create_at desc").Limit(limit).Offset(offset).Scan(&results).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return results, nil
}

func GetSourceFileAndDealFileInfoCount(walletAddress string) (int64, error) {
	sql := "select count(1) as total_record from  source_file s " +
		" inner join source_file_deal_file_map sfdfm on s.id = sfdfm.source_file_id" +
		" inner join deal_file df on sfdfm.deal_file_id = df.id" +
		" where s.wallet_address='" + walletAddress + "'"
	var recordCount common.RecordCount
	err := database.GetDB().Raw(sql).Scan(&recordCount).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return 0, err
	}
	return recordCount.TotalRecord, nil
}

func GetDealListThanGreaterDealID(dealId int64, offset, limit int) ([]*DaoDealResult, error) {
	whereCondition := "deal_id > " + strconv.FormatInt(dealId, 10)
	var results []*DaoDealResult
	err := database.GetDB().Table("deal_file").Where(whereCondition).Offset(offset).Limit(limit).Order("create_at").Scan(&results).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return results, nil
}

func GetDaoSignatureInfoByDealId(dealId int64) ([]*DaoSignResult, error) {
	whereCondition := "deal_id = " + strconv.FormatInt(dealId, 10)
	var results []*DaoSignResult
	err := database.GetDB().Table("event_dao_signature").Where(whereCondition).Offset(0).Limit(constants.DEFAULT_SELECT_LIMIT).Order("block_time desc").Scan(&results).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return results, nil
}

func GetLockFoundInfoByPayloadCid(payloadCid string) (*LockFound, error) {
	lockEventList, err := models.FindEventLockPayment(&models.EventLockPayment{PayloadCid: payloadCid}, "", "10", "0")
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err

	}
	lockFound := new(LockFound)
	if len(lockEventList) > 0 {
		lockFound.PayloadCid = lockEventList[0].PayloadCid
		lockFound.LockedFee = lockEventList[0].LockedFee
		lockFound.CreateAt = lockEventList[0].CreateAt
	}
	return lockFound, nil
}

func GetShoulBeSignDealListFromDB() ([]*DealForDaoSignResult, error) {
	finalSql := "select deal_id,deal_cid,piece_cid,payload_cid,cost,verified,miner_fid,duration,client_wallet_address,create_at from deal_file where deal_id not in  ( " +
		"     select  deal_id from dao_fetched_deal ) " +
		" and deal_id > 0 order by create_at desc"
	var dealForDaoSignResultList []*DealForDaoSignResult
	err := database.GetDB().Raw(finalSql).Scan(&dealForDaoSignResultList).Limit(0).Offset(constants.DEFAULT_SELECT_LIMIT).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err

	}
	return dealForDaoSignResultList, nil
}

func GetDaoSignEventByDealId(dealId int64) ([]*DaoInfoResult, error) {
	if dealId == 0 {
		dealId = constants.FILE_BLOCK_NUMBER_MAX
	}
	finalSql := " select * from( " +
		" (select dao_name, dao_address,order_index from dao_info order by order_index asc)) as d  left  join " +
		" (select deal_id,tx_hash,dao_pass_time,if(deal_id > 0,1,2) as status,dao_address as dao_address_event,payload_cid  from event_dao_signature where deal_id = " + strconv.FormatInt(dealId, 10) + " ) as a " +
		" on d.dao_address=a.dao_address_event"

	var daoInfoResult []*DaoInfoResult
	err := database.GetDB().Raw(finalSql).Scan(&daoInfoResult).Limit(0).Offset(constants.DEFAULT_SELECT_LIMIT).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return daoInfoResult, nil
}

func GetThreshHold() (uint8, error) {
	daoAddress := common2.HexToAddress(polygon.GetConfig().PolygonMainnetNode.DaoSwanOracleAddress)
	client := polygon.WebConn.ConnWeb

	pk := os.Getenv("privateKeyOnPolygon")
	if strings.HasPrefix(strings.ToLower(pk), "0x") {
		pk = pk[2:]
	}

	callOpts := new(bind.CallOpts)
	callOpts.From = daoAddress
	callOpts.Context = context.Background()

	daoOracleContractInstance, err := goBind.NewFilswanOracle(daoAddress, client)
	if err != nil {
		logs.GetLogger().Error(err)
		return 0, err
	}

	threshHold, err := daoOracleContractInstance.GetThreshold(callOpts)
	if err != nil {
		logs.GetLogger().Error(err)
		return 0, err
	}
	logs.GetLogger().Info("dao threshHold is : ", threshHold)
	return threshHold, nil
}

func GetDealFileInfoByPayloadCid(payloadCid string) ([]*models.DealFile, error) {
	dealFileList, err := models.FindDealFileList(&models.DealFile{PayloadCid: payloadCid}, "", "10", "0")
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err

	}
	return dealFileList, nil
}
