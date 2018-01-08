import { ResourceStatus } from '../proto/tasks.js';

export interface InfrastructureStatus {
  pubsubStatus: PubsubStatus;
}

export interface PubsubStatus {
  list: ResourceStatus.Type;
  listProgress: ResourceStatus.Type;
  copy: ResourceStatus.Type;
  copyProgress: ResourceStatus.Type;
}
