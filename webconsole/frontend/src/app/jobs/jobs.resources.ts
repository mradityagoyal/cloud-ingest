import { DataSource } from '@angular/cdk/collections';
import { Observable, of } from 'rxjs';


export class GcsData {
  constructor() {
    this.bucketName = '';
    this.objectPrefix = '';
  }
  bucketName: string;
  objectPrefix: string;
}

export class OnPremisesFiler {
  constructor() {
    this.directoryPath = '';
  }
  directoryPath: string;
}

export class ScheduleDate {
  year: number;
  month: number;
  day: number;
  constructor() {
    const currentDate = new Date();
    this.year = currentDate.getUTCFullYear();
    this.month = currentDate.getUTCMonth() + 1;
    this.day = currentDate.getUTCDate();
  }
}

export class Schedule {
  scheduleStartDate: ScheduleDate;
  scheduleEndDate: ScheduleDate;
  constructor() {
    this.scheduleStartDate = new ScheduleDate();
    this.scheduleEndDate = new ScheduleDate();
  }
}

export class TransferSpec {
  gcsDataSink: GcsData;
  onPremFiler: OnPremisesFiler;
  constructor() {
    this.gcsDataSink = new GcsData();
    this.onPremFiler = new OnPremisesFiler();
  }
}

export class TransferOperation {
  name: string;
  projectId: string;
  startTime: string;
  endTime: string;
  status: string;
  transferJobName: string;
  counters: TransferCounters;
}

export class TransferJob {
  name: string;
  description: string;
  projectId: string;
  transferSpec: TransferSpec;
  status: string;
  schedule: Schedule;
  creationTime: string;
  lastModificationTime: string;
  latestOperation: TransferOperation;

  constructor() {
    this.schedule = new Schedule();
    this.status = 'ENABLED';
    this.transferSpec = new TransferSpec();
  }

}

export interface PauseTransferJobRequest {
  name: string;
  projectId: string;
}

export interface ResumeTransferJobRequest {
  name: string;
  projectId: string;
}

export interface DeleteTransferJobRequest {
  jobName: string;
  projectId: string;
}

/**
 * Holds information about a checked item in the job configurations page.
 */
export interface CheckedItemInfo {
  isChecked: boolean;
  isSafeToDelete: boolean;
}

export class TransferCounters {
  objectsFoundFromSource: number;
  objectsFromSourceFailed: number;
  objectsCopiedToSink: number;
  bytesCopiedToSink: number;
  directoriesFoundFromSource: number;
  directoriesFailedToListFromSource: number;
  directoriesSuccessfullyListedFromSource: number;
}

export interface TransferJobResponse {
  transferJobs: TransferJob[];
}

/**
 * Maps Operation Status and their representations to readable strings.
 */
export const OPERATION_STATUS_TO_STRING_MAP = {};
OPERATION_STATUS_TO_STRING_MAP['STATUS_UNSPECIFIED'] = 'Unspecified';
OPERATION_STATUS_TO_STRING_MAP['IN_PROGRESS'] = 'In Progress',
OPERATION_STATUS_TO_STRING_MAP['PAUSED'] = 'Paused';
OPERATION_STATUS_TO_STRING_MAP['PAUSING'] = 'Pausing',
OPERATION_STATUS_TO_STRING_MAP['SUCCESS'] = 'Success';
OPERATION_STATUS_TO_STRING_MAP['FAILED'] = 'Failed';
OPERATION_STATUS_TO_STRING_MAP['ABORTED'] = 'Aborted';

export const DEFAULT_BACKEND_PAGESIZE = 25;

export class SimpleDataSource<T> extends DataSource<T> {
  items: T[];

  constructor(items: T[]) {
    super();
    this.items = items;
  }

  connect(): Observable<T[]> {
    return of(this.items);
  }

  disconnect() {}
}
