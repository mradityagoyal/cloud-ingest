/**
 * This file contains fakes for the tests in the jobs directory.
 */
import { TaskFailureType } from '../proto/tasks.js';
import { JobConfigResponse, Job, JobSpec, Task } from './jobs.resources';


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
    LastModificationTime: '1507402800000000000',
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
    LastModificationTime: '1444244400000000000',
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
    LastModificationTime: '1507402800000000000',
    FailureMessage: 'Fake failure message 3'
  }
];

export const FAKE_JOBSPEC1: JobSpec = {'onPremSrcDirectory': 'fakeSrcDir1', 'gcsBucket' : 'fakeBucket1'};
export const FAKE_JOBSPEC2: JobSpec = {'onPremSrcDirectory': 'fakeSrcDir2', 'gcsBucket' : 'fakeBucket2'};
export const FAKE_JOBSPEC3: JobSpec = {'onPremSrcDirectory': 'fakeSrcDir3', 'gcsBucket' : 'fakeBucket3'};
export const FAKE_JOBSPEC4: JobSpec = {'onPremSrcDirectory': 'fakeSrcDir4', 'gcsBucket' : 'fakeBucket4'};

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

export const FAKE_JOB_RUNS: Job[] = [
  {
    JobConfigId: 'fakeJobConfigId0',
    JobRunId: 'fakeJobRunId0',
    JobCreationTime: '1504833274371000000',
    Status: 0,
    JobSpec: FAKE_JOBSPEC1,
    Counters: {
      totalTasks: 0,
      tasksCompleted: 0,
      tasksFailed: 0,

      totalTasksList: 0,
      tasksCompletedList: 0,
      tasksFailedList: 0,

      totalTasksCopy: 0,
      tasksCompletedCopy: 0,
      tasksFailedCopy: 0,

      totalTasksLoad: 0,
      tasksCompletedLoad: 0,
      tasksFailedLoad: 0,

      listFilesFound: 0,
      listBytesFound: 0,
      bytesCopied: 0
    }
  },
  {
    JobConfigId: 'fakeJobConfigId1',
    JobRunId: 'fakeJobRunId1',
    JobCreationTime: '1504833274371000000',
    Status: 1,
    JobSpec: FAKE_JOBSPEC2,
    Counters: {
      totalTasks: 1,
      tasksCompleted: 0,
      tasksFailed: 0,

      totalTasksList: 1,
      tasksCompletedList: 0,
      tasksFailedList: 0,

      totalTasksCopy: 0,
      tasksCompletedCopy: 0,
      tasksFailedCopy: 0,

      totalTasksLoad: 0,
      tasksCompletedLoad: 0,
      tasksFailedLoad: 0,

      listFilesFound: 0,
      listBytesFound: 0,
      bytesCopied: 0
    }
  },
  {
    JobConfigId: 'fakeJobConfigId2',
    JobRunId: 'fakeJobRunId2',
    JobCreationTime: '1504833274371000000',
    Status: 2,
    JobSpec: FAKE_JOBSPEC3,
    Counters: {
      totalTasks: 5,
      tasksCompleted: 4,
      tasksFailed: 1,

      totalTasksList: 1,
      tasksCompletedList: 1,
      tasksFailedList: 0,

      totalTasksCopy: 4,
      tasksCompletedCopy: 3,
      tasksFailedCopy: 1,

      totalTasksLoad: 0,
      tasksCompletedLoad: 0,
      tasksFailedLoad: 0,

      listFilesFound: 4,
      listBytesFound: 11223344,
      bytesCopied: 11220000
    }
  },
  {
    JobConfigId: 'fakeJobConfigId3',
    JobRunId: 'fakeJobRunId3',
    JobCreationTime: '1504833274371000000',
    Status: 3,
    JobSpec: FAKE_JOBSPEC4,
    Counters: {
      totalTasks: 9,
      tasksCompleted: 9,
      tasksFailed: 0,

      totalTasksList: 1,
      tasksCompletedList: 1,
      tasksFailedList: 0,

      totalTasksCopy: 4,
      tasksCompletedCopy: 4,
      tasksFailedCopy: 0,

      totalTasksLoad: 4,
      tasksCompletedLoad: 4,
      tasksFailedLoad: 0,

      listFilesFound: 4,
      listBytesFound: 11223344,
      bytesCopied: 11223344
    }
  }
];

export const EMPTY_TASK_ARRAY: Task[] = [];

export const FAKE_JOB_CONFIG_LIST = ['fakeconfigid1', 'fakeconfigid2', 'fakeconfigid3'];

export class JobsServiceStub {
  public getJobConfigs = jasmine.createSpy('getJobConfigs');
  public deleteJobConfigs = jasmine.createSpy('deleteJobConfigs');
  public postJobConfig = jasmine.createSpy('postJobConfig');
  public getJobRun = jasmine.createSpy('getJobRun');
  public getTasksOfStatus = jasmine.createSpy('getTasksOfStatus');
  public getTasksOfFailureType = jasmine.createSpy('getTasksOfFailureType');
}
