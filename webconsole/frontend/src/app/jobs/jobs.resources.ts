export interface JobConfig {
  JobConfigId: string;
  JobSpec: string;
}

export interface JobRun {
  JobConfigId: string;
  JobRunId: string;
  JobCreationTime: string;
  Status: number;
  Progress: Progress;
}

export interface JobRunParams {
  JobConfigId: string;
  JobRunId: string;
}

export interface Progress {
  totalTasks: number;
  tasksCompleted: number;
  tasksFailed: number;
  list?: ListProgress;
  uploadGCS?: UploadGCSProgress;
  loadBigQuery?: LoadBigQueryProgress;
}

export interface ListProgress {
  totalLists: number;
  listsCompleted: number;
  listsFailed: number;
}

export interface UploadGCSProgress {
  totalFiles: number;
  filesCompleted: number;
  filesFailed: number;
}

export interface LoadBigQueryProgress {
  totalObjects: number;
  objectsCompleted: number;
  objectsFailed: number;
}

