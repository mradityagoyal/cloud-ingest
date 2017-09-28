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
