export interface JobConfig {
  JobConfigId: string;
  JobSpec: string;
}

export interface JobRun {
  JobConfigId: string;
  JobRunId: string;
  JobCreationTime: string;
  Status: number;
}

export interface JobRunParams {
  JobConfigId: string;
  JobRunId: string;
}
