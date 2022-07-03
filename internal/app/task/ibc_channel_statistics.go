package task

import (
	"time"

	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/dto"
	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/entity"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type ChannelStatisticsTask struct {
}

var channelStatisticsTask ChannelStatisticsTask

func (t *ChannelStatisticsTask) Name() string {
	return "ibc_channel_statistics"
}

func (t *ChannelStatisticsTask) Run() int {
	historySegments, err := getHistorySegment()
	if err != nil {
		logrus.Errorf("task %s getHistorySegment err, %v", t.Name(), err)
		return -1
	}
	logrus.Infof("task %s deal history segment total: %d", t.Name(), len(historySegments))
	if err = t.dealHistory(historySegments); err != nil {
		logrus.Errorf("task %s dealHistory err, %v", t.Name(), err)
		return -1
	}

	segments, err := getSegment()
	if err != nil {
		logrus.Errorf("task %s getSegment err, %v", t.Name(), err)
		return -1
	}
	logrus.Infof("task %s deal segment total: %d", t.Name(), len(segments))
	if err = t.deal(segments, opInsert); err != nil {
		logrus.Errorf("task %s deal err, %v", t.Name(), err)
		return -1
	}

	return 1
}

// dealHistory 处理历史记录，针对ex_ibc_tx
func (t *ChannelStatisticsTask) dealHistory(segments []*segment) error {
	for _, v := range segments {
		txs, err := ibcTxRepo.AggrIBCChannelHistoryTxs(v.StartTime, v.EndTime)
		if err != nil {
			logrus.Errorf("task %s AggrIBCChannelHistoryTxs err, %v", t.Name(), err)
			return err
		}

		if len(txs) == 0 {
			continue
		}

		aggr := t.aggr(txs)
		if err = t.saveData(aggr, v.StartTime, v.EndTime, opInsert); err != nil {
			return err
		}
	}
	return nil
}

// deal 处理最新的记录，针对ex_ibc_tx_latest
func (t *ChannelStatisticsTask) deal(segments []*segment, op int) error {
	for _, v := range segments {
		txs, err := ibcTxRepo.AggrIBCChannelTxs(v.StartTime, v.EndTime)
		if err != nil {
			logrus.Errorf("task %s AggrIBCChannelTxs err, %v", t.Name(), err)
			return err
		}

		if len(txs) == 0 {
			continue
		}

		aggr := t.aggr(txs)
		if err = t.saveData(aggr, v.StartTime, v.EndTime, op); err != nil {
			return err
		}
	}
	return nil
}

func (t *ChannelStatisticsTask) aggr(txs []*dto.AggrIBCChannelTxsDTO) []*dto.ChannelStatisticsDTO {
	var cl []*dto.ChannelStatisticsDTO
	for _, v := range txs {
		isExisted := false
		ChannelId := generateChannelId(v.ScChainId, v.ScChannel, v.DcChainId, v.DcChannel)
		for _, c := range cl {
			if c.ChannelId == ChannelId && v.BaseDenom == c.BaseDenom { // 同一个channel
				c.TxsCount += v.Count
				c.TxsAmount = c.TxsAmount.Add(decimal.NewFromFloat(v.Amount))
				isExisted = true
				break
			}
		}

		if !isExisted {
			cl = append(cl, &dto.ChannelStatisticsDTO{
				ChannelId: ChannelId,
				BaseDenom: v.BaseDenom,
				TxsCount:  v.Count,
				TxsAmount: decimal.NewFromFloat(v.Amount),
			})
		}
	}

	return cl
}

func (t *ChannelStatisticsTask) saveData(dtoList []*dto.ChannelStatisticsDTO, segmentStart, segmentEnd int64, op int) error {
	var statistics = make([]*entity.IBCChannelStatistics, 0, len(dtoList))
	for _, v := range dtoList {
		statistics = append(statistics, &entity.IBCChannelStatistics{
			ChannelId:         v.ChannelId,
			TransferBaseDenom: v.BaseDenom,
			TransferTxs:       v.TxsCount,
			TransferAmount:    v.TxsAmount.String(),
			SegmentStartTime:  segmentStart,
			SegmentEndTime:    segmentEnd,
			CreateAt:          time.Now().Unix(),
			UpdateAt:          time.Now().Unix(),
		})
	}

	var err error
	if op == opInsert {
		if err = channelStatisticsRepo.BatchInsert(statistics); err != nil {
			logrus.Errorf("task %s channelStatisticsRepo.BatchInsert err, %v", t.Name(), err)
		}
	} else {
		if err = channelStatisticsRepo.BatchSwap(segmentStart, segmentEnd, statistics); err != nil {
			logrus.Errorf("task %s channelStatisticsRepo.BatchSwap err, %v", t.Name(), err)
		}
	}

	return err
}