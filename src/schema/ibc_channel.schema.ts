import * as mongoose from 'mongoose';

export const IbcChannelSchema = new mongoose.Schema(
  {
    channel_id: String,
    record_id: String,
    update_at: String,
    create_at: String,
  },
  { versionKey: false },
);

IbcChannelSchema.index({ record_id: 1, denom: 1 }, { unique: true });
IbcChannelSchema.index({ update_at: -1 }, { background: true });

IbcChannelSchema.statics = {
  // 查
  async findCount(query) {
    return await this.count(query);
  },

  async findChannelRecord(record_id, cb) {
    const result = await this.findOne({ record_id }, { _id: 0 }, cb);
    return result;
  },

  // 改
  async updateChannelRecord(channelRecord, cb) {
    const { record_id } = channelRecord;
    const options = { upsert: true, new: false, setDefaultsOnInsert: true };
    return await this.findOneAndUpdate(
      { record_id },
      channelRecord,
      options,
      cb,
    );
  },

  // 增
  async insertManyChannel(ibcChannel, cb) {
    return await this.insertMany(ibcChannel, { ordered: false }, cb);
  },
};
