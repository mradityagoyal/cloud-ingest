export interface InfrastructureStatus {
  spannerStatus: string;
  pubsubStatus: PubsubStatus;
  dcpStatus: string;
}

export interface PubsubStatus {
  list: string;
  listProgress: string;
  uploadGCS: string;
  uploadGCSProgress: string;
  loadBigQuery: string;
  loadBigQueryProgress: string;
}

export const INFRA_STATUS = {
  RUNNING : 'RUNNING',
  NOT_FOUND: 'NOT_FOUND',
  DEPLOYING: 'DEPLOYING',
  DELETING: 'DELETING',
  FAILED: 'FAILED',
  UNKNOWN: 'UNKNOWN'
};
