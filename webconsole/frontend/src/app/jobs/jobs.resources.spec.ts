import { async } from '@angular/core/testing';
import { FAKE_TRANSFER_JOB_RESPONSE } from './jobs.test-util';
import { SimpleDataSource, TransferJob } from './jobs.resources';

let simpleJobConfigResponseDataSource: SimpleDataSource<TransferJob>;


describe('SimpleDataSource', () => {

      beforeEach(() => {
        simpleJobConfigResponseDataSource = new SimpleDataSource(FAKE_TRANSFER_JOB_RESPONSE.transferJobs);
      });

      it('connect() should return the sample job configs', async(() => {
        simpleJobConfigResponseDataSource.connect()
        .subscribe((jobConfigs: TransferJob[]) => {
          expect(jobConfigs).toEqual(FAKE_TRANSFER_JOB_RESPONSE.transferJobs);
        }, () => {
          fail('Should return job configs');
        });
      }));
  });
