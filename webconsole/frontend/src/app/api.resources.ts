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

export interface InfrastructureStatus {
  spannerStatus: string;
  pubsubStatus: PubsubStatus;
  dcpStatus: string;
  cloudFunctionsStatus: string;
}

export interface PubsubStatus {
  list: string;
  listProgress: string;
  uploadGCS: string;
  uploadGCSProgress: string;
  loadBigQuery: string;
  loadBigQueryProgress: string;
}
