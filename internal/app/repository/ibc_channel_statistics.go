package repository

import (
	"context"

	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/dto"
	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/entity"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

type IChannelStatisticsRepo interface {
	BatchSwap(segmentStartTime, segmentEndTime int64, batch []*entity.IBCChannelStatistics) error
	BatchInsert(batch []*entity.IBCChannelStatistics) error
	Aggr() ([]*dto.ChannelStatisticsAggrDTO, error)
}

var _ IChannelStatisticsRepo = new(ChannelStatisticsRepo)

type ChannelStatisticsRepo struct {
}

func (repo *ChannelStatisticsRepo) coll() *qmgo.Collection {
	return mgo.Database(ibcDatabase).Collection(entity.IBCChannelStatistics{}.CollectionName())
}

func (repo *ChannelStatisticsRepo) BatchSwap(segmentStartTime, segmentEndTime int64, batch []*entity.IBCChannelStatistics) error {
	callback := func(sessCtx context.Context) (interface{}, error) {
		query := bson.M{
			"segment_start_time": segmentStartTime,
			"segment_end_time":   segmentEndTime,
		}
		if _, err := repo.coll().RemoveAll(sessCtx, query); err != nil {
			return nil, err
		}

		if len(batch) == 0 {
			return nil, nil
		}

		if _, err := repo.coll().InsertMany(sessCtx, batch); err != nil {
			return nil, err
		}

		return nil, nil
	}
	_, err := mgo.DoTransaction(context.Background(), callback)
	return err
}

func (repo *ChannelStatisticsRepo) BatchInsert(batch []*entity.IBCChannelStatistics) error {
	if len(batch) == 0 {
		return nil
	}

	_, err := repo.coll().InsertMany(context.Background(), batch)
	return err
}

func (repo *ChannelStatisticsRepo) Aggr() ([]*dto.ChannelStatisticsAggrDTO, error) {
	group := bson.M{
		"$group": bson.M{
			"_id": bson.M{
				"channel_id":          "$channel_id",
				"base_denom":          "$base_denom",
				"base_denom_chain_id": "$base_denom_chain_id",
			},
			"count": bson.M{
				"$sum": "$transfer_txs",
			},
			"amount": bson.M{
				"$sum": bson.M{
					"$toDouble": "$transfer_amount",
				},
			},
		},
	}
	project := bson.M{
		"$project": bson.M{
			"_id":                 0,
			"channel_id":          "$_id.channel_id",
			"base_denom":          "$_id.base_denom",
			"base_denom_chain_id": "$_id.base_denom_chain_id",
			"count":               "$count",
			"amount":              "$amount",
		},
	}

	var pipe []bson.M
	pipe = append(pipe, group, project)
	var res []*dto.ChannelStatisticsAggrDTO
	err := repo.coll().Aggregate(context.Background(), pipe).All(&res)
	return res, err
}