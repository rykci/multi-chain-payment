package storage

import (
	"context"
	"mime/multipart"
	"os"
	"path/filepath"
	"payment-bridge/blockchain/browsersync/scanlockpayment/polygon"
	"payment-bridge/common/constants"
	"payment-bridge/common/utils"
	"payment-bridge/config"
	"payment-bridge/database"
	"payment-bridge/models"
	"payment-bridge/on-chain/goBind"
	"payment-bridge/scheduler"
	"strconv"
	"strings"

	"github.com/filswan/go-swan-lib/logs"
	"github.com/shopspring/decimal"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	common2 "github.com/ethereum/go-ethereum/common"

	"github.com/filswan/go-swan-lib/client/ipfs"

	libutils "github.com/filswan/go-swan-lib/utils"
	"github.com/gin-gonic/gin"
)

func UpdateSourceFileMaxPrice(id int64, maxPrice decimal.Decimal) error {
	err := models.UpdateSourceFileMaxPrice(id, maxPrice)
	if err != nil {
		logs.GetLogger().Error(err)
		return err
	}

	return nil
}

func SaveFile(c *gin.Context, srcFile *multipart.FileHeader, duration int, walletAddress string) (*int64, *string, *string, *int, error) {
	srcDir := scheduler.GetSrcDir()

	filename := srcFile.Filename
	if libutils.IsFileExists(srcDir, filename) {
		for i := 0; ; i++ {
			filename = srcFile.Filename + strconv.Itoa(i)
			if !libutils.IsFileExists(srcDir, filename) {
				break
			}
		}
	}

	srcFilepath := filepath.Join(srcDir, filename)
	logs.GetLogger().Info("saving source file to ", srcFilepath)
	err := c.SaveUploadedFile(srcFile, srcFilepath)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, nil, nil, nil, err
	}
	logs.GetLogger().Info("source file saved to ", srcFilepath)

	uploadUrl := utils.UrlJoin(config.GetConfig().IpfsServer.UploadUrlPrefix, "api/v0/add?stream-channels=true&pin=true")
	ipfsFileHash, err := ipfs.IpfsUploadFileByWebApi(uploadUrl, srcFilepath)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, nil, nil, nil, err
	}

	ipfsUrl := libutils.UrlJoin(config.GetConfig().IpfsServer.DownloadUrlPrefix, constants.IPFS_URL_PREFIX_BEFORE_HASH, *ipfsFileHash)

	sourceFiles, err := models.GetSourceFilesByPayloadCid(*ipfsFileHash)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, nil, nil, nil, err
	}

	needPay := 0

	// not uploaded by anyone yet
	if len(sourceFiles) == 0 {
		sourceFile := models.SourceFile{
			FileName:      srcFile.Filename,
			FileSize:      srcFile.Size,
			ResourceUri:   srcFilepath,
			Status:        constants.SOURCE_FILE_STATUS_CREATED,
			CreateAt:      utils.GetCurrentUtcMilliSecond(),
			IpfsUrl:       ipfsUrl,
			PinStatus:     constants.IPFS_File_PINNED_STATUS,
			WalletAddress: walletAddress,
			PayloadCid:    *ipfsFileHash,
		}

		sourceFileCreated, err := models.CreateSourceFile(sourceFile)
		if err != nil {
			logs.GetLogger().Error(err)
			return nil, nil, nil, nil, err
		}

		return &sourceFileCreated.ID, ipfsFileHash, &ipfsUrl, &needPay, nil
	}

	// remove the current copy of file
	err = os.Remove(srcFilepath)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, nil, nil, nil, err
	}

	for _, srcFile := range sourceFiles {
		// uploaded by the same wallet
		if srcFile.PayloadCid == *ipfsFileHash && srcFile.WalletAddress == walletAddress {
			eventLockPayments, err := models.GetEventLockPaymentByPayloadCidWallet(*ipfsFileHash, walletAddress)
			if err != nil {
				logs.GetLogger().Error(err)
				return nil, nil, nil, nil, err
			}

			if len(eventLockPayments) > 0 { // paid by the same wallet
				needPay = 1
			} else { // not paid by the same wallet
				needPay = 2
			}

			return &srcFile.ID, &srcFile.PayloadCid, &srcFile.IpfsUrl, &needPay, nil
		}
	}

	// uploaded by other wallet
	eventLockPayments, err := models.GetEventLockPaymentByPayloadCid(*ipfsFileHash)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, nil, nil, nil, err
	}

	if len(eventLockPayments) > 0 { // uploaded and paid by others
		needPay = 3
	} else { // uploaded by others but not paid
		needPay = 4
	}

	sourceFile := models.SourceFile{
		FileName:      srcFile.Filename,
		FileSize:      srcFile.Size,
		ResourceUri:   sourceFiles[0].ResourceUri,
		Status:        constants.SOURCE_FILE_STATUS_CREATED,
		CreateAt:      utils.GetCurrentUtcMilliSecond(),
		IpfsUrl:       sourceFiles[0].IpfsUrl,
		PinStatus:     constants.IPFS_File_PINNED_STATUS,
		WalletAddress: walletAddress,
		PayloadCid:    sourceFiles[0].PayloadCid,
	}
	sourceFileCreated, err := models.CreateSourceFile(sourceFile)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, nil, nil, nil, err
	}

	return &sourceFileCreated.ID, &sourceFile.PayloadCid, &sourceFile.IpfsUrl, &needPay, nil
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
		lockFound.CreateAt = strconv.FormatInt(lockEventList[0].CreateAt, 10)
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
