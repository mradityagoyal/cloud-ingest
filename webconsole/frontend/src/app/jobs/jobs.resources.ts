import { DataSource } from '@angular/cdk/collections';
import { Observable } from 'rxjs/Rx';

import { TaskFailureType } from '../proto/tasks.js';

export class JobConfigRequest {
  /**
   * The job configuration id is a string that uniquely identified the job configuration. This is
   * also referred to as a job "description" to the user.
   */
  jobConfigId: string;
  gcsBucket: string;
  fileSystemDirectory: string;
  constructor(
    jobConfigId: string,
    gcsBucket: string,
    fileSystemDirectory: string
  ) { }
}

/**
 * A JobConfigResponse is an object returned by the backend that represents a job configuration.
 */
export interface JobConfigResponse {
  JobConfigId: string;
  JobSpec: JobSpec;
}

/**
 * A JobSpec contains the information that describes the job.
 */
export interface JobSpec {
  gcsBucket: string;
  onPremSrcDirectory: string;
}

export interface JobRun {
  JobConfigId: string;
  JobRunId: string;
  JobCreationTime: string;
  Status: number;
  Counters: Counters;
  JobSpec: JobSpec;
}

export interface Counters {
  totalTasks: number;
  tasksCompleted: number;
  tasksFailed: number;

  totalTasksList: number;
  tasksCompletedList: number;
  tasksFailedList: number;

  totalTasksCopy: number;
  tasksCompletedCopy: number;
  tasksFailedCopy: number;

  totalTasksLoad: number;
  tasksCompletedLoad: number;
  tasksFailedLoad: number;

  listFilesFound: number;
  listBytesFound: number;
  bytesCopied: number;
}

export interface Task {
  JobConfigId: string;
  JobRunId: string;
  TaskId: string;
  TaskSpec: string;
  TaskType: number;
  FailureType?: TaskFailureType.Type;
  Status: number;
  CreationTime: number;
  WorkerId: string;
  LastModificationTime: number;
  FailureMessage: string;
}

export const TASK_STATUS = {
  UNQUEUED: 0,
  QUEUED: 1,
  FAILED: 2,
  SUCCESS: 3
};

/**
 * Maps task type integers to string representations.
 */
export const TASK_TYPE_TO_STRING_MAP = {
  1: 'List task',
  2: 'GCS Upload',
  3: 'Load to BigQuery',
};

export const FAILURE_TYPE_TO_STRING_MAP = {
  0: 'Unknown failure',
  1: 'Unexpected failure',
  2: 'File Modified failure',
  3: 'MD5 mismatch failure',
  4: 'Precondition failure',
  5: 'File not found failure',
  6: 'Permission failure'
};

export const DEFAULT_BACKEND_PAGESIZE = 25;

export class SimpleDataSource<T> extends DataSource<T> {
  items: T[];

  constructor(items: T[]) {
    super();
    this.items = items;
  }

  connect(): Observable<T[]> {
    return Observable.of(this.items);
  }

  disconnect() {}
}
