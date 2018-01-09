import { DataSource } from '@angular/cdk/collections';
import { Observable } from 'rxjs/Rx';

import { TaskFailureType, TaskType, JobRunStatus } from '../proto/tasks.js';

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

/**
 * A Job contains infromation that describes a job configuration and the last
 * job run.
 */
export interface Job {
  // The job configuration id of the job.
  JobConfigId: string;
  // The job spec of the job configuration.
  JobSpec: JobSpec;
  // The job run id of the last job run.
  JobRunId: string;
  // The time that last job run was created.
  JobCreationTime: string;
  // The status of the last job run.
  Status: JobRunStatus.Type;
  // The counters of the last job run.
  Counters: Counters;
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
  LastModificationTime: string;
  FailureMessage: string;
}

/**
 * Maps task type integers to string representations.
 */
export const TASK_TYPE_TO_STRING_MAP = {};
TASK_TYPE_TO_STRING_MAP[TaskType.Type.LIST] = 'List task';
TASK_TYPE_TO_STRING_MAP[TaskType.Type.COPY] = 'Copy task';

/**
 * Maps failure type enums to their string representations.
 */
export const FAILURE_TYPE_TO_STRING_MAP = {};
FAILURE_TYPE_TO_STRING_MAP[TaskFailureType.Type.UNUSED] = 'Unknown failure';
FAILURE_TYPE_TO_STRING_MAP[TaskFailureType.Type.UNKNOWN] = 'Unexpected failure';
FAILURE_TYPE_TO_STRING_MAP[TaskFailureType.Type.FILE_MODIFIED_FAILURE] = 'File Modified failure';
FAILURE_TYPE_TO_STRING_MAP[TaskFailureType.Type.MD5_MISMATCH_FAILURE] = 'MD5 mismatch failure';
FAILURE_TYPE_TO_STRING_MAP[TaskFailureType.Type.PRECONDITION_FAILURE] = 'Precondition failure';
FAILURE_TYPE_TO_STRING_MAP[TaskFailureType.Type.FILE_NOT_FOUND_FAILURE] = 'File not found failure';
FAILURE_TYPE_TO_STRING_MAP[TaskFailureType.Type.PERMISSION_FAILURE] = 'Permission failure';

/**
 * Maps job run status enums to their string representations.
 */
export const JOB_RUN_STATUS_TO_STRING_MAP = {};
JOB_RUN_STATUS_TO_STRING_MAP[JobRunStatus.Type.NOT_STARTED] = 'Not started';
JOB_RUN_STATUS_TO_STRING_MAP[JobRunStatus.Type.IN_PROGRESS] = 'In Progress',
JOB_RUN_STATUS_TO_STRING_MAP[JobRunStatus.Type.FAILED] = 'Failed';
JOB_RUN_STATUS_TO_STRING_MAP[JobRunStatus.Type.SUCCESS] = 'Sucess';

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
