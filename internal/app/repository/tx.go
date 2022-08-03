package repository

import (
	"context"

	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/constant"
	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/entity"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

type ITxRepo interface {
	GetRelayerScChainAddr(packetId, chainId string) (string, error)
	GetTimePeriodByUpdateClient(chainId, address string, startTime int64) (int64, int64, error)
	GetLatestRecvPacketTime(chainId, address string, startTime int64) (int64, error)
	GetChannelOpenConfirmTime(chainId, channelId string) (int64, error)
	GetTransferTx(chainId string, height, limit int64) ([]*entity.Tx, error)
	FindByTypeAndHeight(chainId, txType string, height int64) ([]*entity.Tx, error)
}

var _ ITxRepo = new(TxRepo)

type TxRepo struct {
}

func (repo *TxRepo) coll(chainId string) *qmgo.Collection {
	return mgo.Database(ibcDatabase).Collection(entity.Tx{}.CollectionName(chainId))
}

func (repo *TxRepo) GetRelayerScChainAddr(packetId, chainId string) (string, error) {
	var res entity.Tx
	//get relayer address by packet_id and acknowledge_packet or timeout_packet
	err := repo.coll(chainId).Find(context.Background(), bson.M{
		"msgs.msg.packet_id": packetId,
		"msgs.type": bson.M{ //filter ibc transfer
			"$in": []string{constant.MsgTypeAcknowledgement, constant.MsgTypeTimeoutPacket},
		},
	}).Sort("-height").Limit(1).One(&res)
	if len(res.DocTxMsgs) > 0 {
		for _, msg := range res.DocTxMsgs {
			cmsg := msg.CommonMsg()
			if cmsg.PacketId == packetId {
				return cmsg.Signer, nil
			}
		}
	}
	return "", err
}

// return value description
//1: latest update_client tx_time
//2: time_period
//3: error
func (repo *TxRepo) GetTimePeriodByUpdateClient(chainId, address string, startTime int64) (int64, int64, error) {
	var res []*entity.Tx
	query := bson.M{
		"msgs.type":       constant.MsgTypeUpdateClient,
		"msgs.msg.signer": address,
		"time": bson.M{
			"$gte": startTime,
		},
	}
	err := repo.coll(chainId).Find(context.Background(), query).
		Select(bson.M{"time": 1}).Sort("-time").Hint("msgs.msg.signer_1_msgs.type_1_time_1").Limit(2).All(&res)
	if err != nil {
		return 0, 0, err
	}
	if len(res) == 2 {
		return res[0].Time, res[0].Time - res[1].Time, nil
	}
	if len(res) == 1 {
		return res[0].Time, -1, nil
	}
	return 0, -1, nil
}

func (repo *TxRepo) GetLatestRecvPacketTime(chainId, address string, startTime int64) (int64, error) {
	var res []*entity.Tx
	query := bson.M{
		"msgs.type":       constant.MsgTypeRecvPacket,
		"msgs.msg.signer": address,
		"time": bson.M{
			"$gte": startTime,
		},
	}
	err := repo.coll(chainId).Find(context.Background(), query).
		Select(bson.M{"time": 1}).Sort("-time").Hint("msgs.msg.signer_1_msgs.type_1_time_1").Limit(1).All(&res)
	if err != nil {
		return 0, err
	}

	if len(res) == 1 {
		return res[0].Time, nil
	}
	return 0, nil
}

func (repo *TxRepo) GetChannelOpenConfirmTime(chainId, channelId string) (int64, error) {
	var res entity.Tx
	query := bson.M{
		"msgs.type":           constant.MsgTypeChannelOpenConfirm,
		"msgs.msg.channel_id": channelId,
	}
	err := repo.coll(chainId).Find(context.Background(), query).
		Select(bson.M{"time": 1}).Sort("-time").Limit(1).One(&res)

	if err != nil {
		return 0, err
	}
	return res.Time, nil
}

func (repo *TxRepo) GetTransferTx(chainId string, height, limit int64) ([]*entity.Tx, error) {
	var res []*entity.Tx
	query := bson.M{
		"types": constant.MsgTypeTransfer,
		"height": bson.M{
			"$gt": height,
		},
	}

	err := repo.coll(chainId).Find(context.Background(), query).Sort("height").Limit(limit).All(&res)
	return res, err
}

func (repo *TxRepo) FindByTypeAndHeight(chainId, txType string, height int64) ([]*entity.Tx, error) {
	var res []*entity.Tx
	query := bson.M{
		"types":  txType,
		"height": height,
	}

	err := repo.coll(chainId).Find(context.Background(), query).All(&res)
	return res, err
}