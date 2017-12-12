import { ResourceStatus } from '../proto/tasks.js';

export interface InfrastructureStatus {
  spannerStatus: ResourceStatus.Type;
  pubsubStatus: PubsubStatus;
  dcpStatus: ResourceStatus.Type;
}

export interface PubsubStatus {
  list: ResourceStatus.Type;
  listProgress: ResourceStatus.Type;
  uploadGCS: ResourceStatus.Type;
  uploadGCSProgress: ResourceStatus.Type;
}
