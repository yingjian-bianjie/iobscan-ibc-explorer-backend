package repository

import (
	"context"

	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/dto"
	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/entity"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

type IIBCTxFailLogRepo interface {
	BatchInsert(batch []*entity.IBCTxFailLog) error
	BatchSwap(segmentStartTime, segmentEndTime int64, batch []*entity.IBCTxFailLog) error
	FailureStatistics(chain string, startTime, endTime int64) ([]*dto.FailureStatisticsSDTO, error)
}

var _ IIBCTxFailLogRepo = new(IBCTxFailLogRepo)

type IBCTxFailLogRepo struct {
}

func (repo *IBCTxFailLogRepo) coll() *qmgo.Collection {
	return mgo.Database(ibcDatabase).Collection(entity.IBCTxFailLogCollName)
}

func (repo *IBCTxFailLogRepo) BatchInsert(batch []*entity.IBCTxFailLog) error {
	if len(batch) == 0 {
		return nil
	}

	_, err := repo.coll().InsertMany(context.Background(), batch)
	return err
}

func (repo *IBCTxFailLogRepo) BatchSwap(segmentStartTime, segmentEndTime int64, batch []*entity.IBCTxFailLog) error {
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

func (repo *IBCTxFailLogRepo) parseFailureStatisticsQuery(chain string, startTime, endTime int64) bson.M {
	query := bson.M{}
	query["chain"] = chain
	if startTime > 0 {
		query["segment_start_time"] = bson.M{
			"$gte": startTime,
		}
	}

	if endTime > 0 {
		query["segment_end_time"] = bson.M{
			"$lte": endTime,
		}
	}
	return query
}

func (repo *IBCTxFailLogRepo) FailureStatistics(chain string, startTime, endTime int64) ([]*dto.FailureStatisticsSDTO, error) {
	cond := repo.parseFailureStatisticsQuery(chain, startTime, endTime)
	match := bson.M{
		"$match": cond,
	}

	group := bson.M{
		"$group": bson.M{
			"_id": "$code",
			"txs_num": bson.M{
				"$sum": "$txs_number",
			},
		},
	}

	var pipe []bson.M
	pipe = append(pipe, match, group)
	var res []*dto.FailureStatisticsSDTO
	err := repo.coll().Aggregate(context.Background(), pipe).All(&res)
	return res, err
}