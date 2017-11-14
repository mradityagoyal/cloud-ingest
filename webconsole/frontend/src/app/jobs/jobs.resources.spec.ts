import { async } from '@angular/core/testing';
import { FAKE_TASKS, FAKE_JOB_CONFIGS } from './jobs.test-util';
import { JobConfigResponse, SimpleDataSource, Task } from './jobs.resources';

let simpleTaskDataSource: SimpleDataSource<Task>;
let simpleJobConfigResponseDataSource: SimpleDataSource<JobConfigResponse>;


describe('SimpleDataSource', () => {

      beforeEach(() => {
        simpleTaskDataSource = new SimpleDataSource(FAKE_TASKS);
        simpleJobConfigResponseDataSource = new SimpleDataSource(FAKE_JOB_CONFIGS);
      });

      it('connect() should return the sample tasks', async(() => {
        simpleTaskDataSource.connect().subscribe((tasks: Task[]) => {
          expect(tasks).toEqual(FAKE_TASKS);
        }, () => {
          fail('Should return the tasks.');
        });
      }));

      it('connect() should return the sample job configs', async(() => {
        simpleJobConfigResponseDataSource.connect()
        .subscribe((jobConfigs: JobConfigResponse[]) => {
          expect(jobConfigs).toEqual(FAKE_JOB_CONFIGS);
        }, () => {
          fail('Should return job configs');
        });
      }));
  });
