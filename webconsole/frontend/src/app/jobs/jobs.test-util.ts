/**
 * This file contains fakes for the tests in the jobs directory.
 */
import { TaskFailureType } from '../proto/tasks.js';
import { JobConfigResponse, JobSpec, Task } from './jobs.resources';


export const FAKE_TASKS: Task[] = [
  {
    JobConfigId: 'fakeJobConfigId1',
    JobRunId: 'fakeJobRunId1',
    TaskId: 'fakeTaskId1',
    TaskSpec: '{ fakeField: "fakeTaskSpec1" }',
    TaskType: 1,
    FailureType: TaskFailureType.Type.UNKNOWN,
    Status: 3,
    // September 7, 2016 12:00:00 PM
    CreationTime: 1473274800000000000,
    WorkerId: 'fakeWorkerId1',
    // October 7, 2017, 12:00:00 PM
    LastModificationTime: 1507402800000000000,
    FailureMessage: 'Fake failure message 1'
  },
  {
    JobConfigId: 'fakeJobConfigId2',
    JobRunId: 'fakeJobRunId2',
    TaskId: 'fakeTaskId2',
    TaskSpec: '{ fakeField: "fakeTaskSpec2" }',
    TaskType: 2,
    FailureType: TaskFailureType.Type.UNKNOWN,
    Status: 3,
    // October 7, 2014 12:00:00 PM
    CreationTime: 1412708400000000000,
    WorkerId: 'fakeWorkerId2',
    // October 7, 2015 12:00:00 PM
    LastModificationTime: 1444244400000000000,
    FailureMessage: 'Fake failure message 2'
  }
];

export const FAKE_TASKS2: Task[] = [
  {
    JobConfigId: 'fakeJobConfigId3',
    JobRunId: 'fakeJobRunId3',
    TaskId: 'fakeTaskId3',
    TaskSpec: '{ fakeField: "fakeTaskSpec3" }',
    TaskType: 1,
    FailureType: TaskFailureType.Type.UNKNOWN,
    Status: 3,
    // September 7, 2016 12:00:00 PM
    CreationTime: 1473274800000000000,
    WorkerId: 'fakeWorkerId3',
    // October 7, 2017, 12:00:00 PM
    LastModificationTime: 1507402800000000000,
    FailureMessage: 'Fake failure message 3'
  }
];

export const FAKE_JOBSPEC1: JobSpec = {'onPremSrcDirectory': 'fakeSrcDir1', 'gcsBucket' : 'fakeBucket1'};
export const FAKE_JOBSPEC2: JobSpec = {'onPremSrcDirectory': 'fakeSrcDir2', 'gcsBucket' : 'fakeBucket2'};
export const FAKE_JOBSPEC3: JobSpec = {'onPremSrcDirectory': 'fakeSrcDir3', 'gcsBucket' : 'fakeBucket3'};

export const FAKE_JOB_CONFIGS: JobConfigResponse[] = [
  {
    JobConfigId: 'fakeJobConfigId1',
    JobSpec: FAKE_JOBSPEC1
  },
  {
    JobConfigId: 'fakeJobConfigId2',
    JobSpec: FAKE_JOBSPEC2
  },
  {
    JobConfigId: 'fakeJobConfigId3',
    JobSpec: FAKE_JOBSPEC3
  }
];

export const EMPTY_TASK_ARRAY: Task[] = [];

export const FAKE_HTTP_ERROR = {error : {error: 'FakeError', message: 'Fake Error Message.'}};
