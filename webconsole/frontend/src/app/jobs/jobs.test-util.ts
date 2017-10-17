/**
 * This file contains fakes for the tests in the jobs directory.
 */

import { Task, TASK_STATUS } from './jobs.resources';
import { TaskFailureType } from '../proto/tasks.js';

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

export const FAKE_HTTP_ERROR = {error : {error: 'FakeError', message: 'Fake Error Message.'}};
