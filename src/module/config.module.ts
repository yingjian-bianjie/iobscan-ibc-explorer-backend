import { Module } from '@nestjs/common';
import { ConfigController } from 'src/controller/config.controller';
import { ConfigService } from '../service/config.service';
@Module({
  providers: [ConfigService],
  controllers: [ConfigController],
})
export class ConfigModule {}
