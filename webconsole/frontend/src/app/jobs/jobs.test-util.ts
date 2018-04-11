/**
 * This file contains fakes for the tests in the jobs directory.
 */
import { TaskFailureType } from '../proto/tasks.js';
import { TransferJobResponse, Schedule } from './jobs.resources';


export const EMPTY_TRANSFER_JOB_RESPONSE = {};

export const FAKE_TRANSFER_JOB_RESPONSE: TransferJobResponse = {
  transferJobs: [
    {
      name: 'transferJobs/OPI1',
      description: '',
      projectId: 'testProjectId',
      status: 'ENABLED',
      schedule: new Schedule(),
      creationTime: '2018-04-10T23:16:26.565769187Z',
      lastModificationTime: '2018-04-10T23:16:26.565769187Z',
      transferSpec: {
       gcsDataSink: {
         bucketName: 'testBucket'
        },
       onPremFiler: {
         directoryPath: '/test/path'
        }
      },
     latestOperation: {
        name: 'transferOperations/OPI1',
        projectId: 'testProjectId',
        status: 'IN_PROGRESS',
        transferJobName: 'transferJobs/OPI1',
        startTime: '2018-04-10T23:15:26.565769187Z',
        endTime: '1970-01-01T00:00:00Z',
     }
     },
     {
      name: 'transferJobs/OPI2',
      description: '',
      projectId: 'testProjectId',
      status: 'ENABLED',
      creationTime: '2018-04-10T23:16:26.565769187Z',
      lastModificationTime: '2018-04-10T23:16:26.565769187Z',
      schedule: new Schedule(),
      transferSpec: {
       gcsDataSink: {
         bucketName: 'testBucket2'
        },
       onPremFiler: {
         directoryPath: '/test/path2'
        }
      },
     latestOperation: {
        name: 'transferOperations/OPI2',
        projectId: 'testProjectId',
        status: 'IN_PROGRESS',
        transferJobName: 'transferJobs/OPI2',
        startTime: '2018-04-10T23:16:26.565769187Z',
        endTime: '1970-01-01T00:00:00Z',
     }
     },
     {
      name: 'transferJobs/OPI3',
      description: '',
      schedule: new Schedule(),
      projectId: 'testProjectId',
      creationTime: '2018-04-10T23:16:26.565769187Z',
      lastModificationTime: '2018-04-10T23:16:26.565769187Z',
      status: 'ENABLED',
      transferSpec: {
       gcsDataSink: {
         bucketName: 'testBucket3'
        },
       onPremFiler: {
         directoryPath: '/test/path3'
        }
      },
     latestOperation: {
        name: 'transferOperations/OPI3',
        projectId: 'testProjectId',
        status: 'IN_PROGRESS',
        transferJobName: 'transferJobs/OPI3',
        startTime: '2018-04-10T23:17:26.565769187Z',
        endTime: '1970-01-01T00:00:00Z',
     }
     }
  ]
};

export class JobsServiceStub {
  public getJobs = jasmine.createSpy('getJobs');
  public postJob = jasmine.createSpy('postJobConfig');
  public getJob = jasmine.createSpy('getJobRun');
}
