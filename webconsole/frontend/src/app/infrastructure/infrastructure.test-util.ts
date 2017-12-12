/**
 * This file contains fakes for the tests in the infrastructure directory.
 */
import { InfrastructureStatus, PubsubStatus } from './infrastructure.resources';
import { ResourceStatus } from '../proto/tasks.js';

const FAKE_PUBSUB_STATUS_RUNNING: PubsubStatus = {
 list:  ResourceStatus.Type.RUNNING,
 listProgress: ResourceStatus.Type.RUNNING,
 uploadGCS: ResourceStatus.Type.RUNNING,
 uploadGCSProgress: ResourceStatus.Type.RUNNING
};

export const FAKE_INFRA_STATUS_RUNNING: InfrastructureStatus = {
  spannerStatus: ResourceStatus.Type.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_RUNNING,
  dcpStatus: ResourceStatus.Type.RUNNING
};

const FAKE_PUBSUB_STATUS_NOT_FOUND: PubsubStatus = {
 list:  ResourceStatus.Type.NOT_FOUND,
 listProgress: ResourceStatus.Type.NOT_FOUND,
 uploadGCS: ResourceStatus.Type.NOT_FOUND,
 uploadGCSProgress: ResourceStatus.Type.NOT_FOUND
};

export const FAKE_INFRA_STATUS_NOT_FOUND: InfrastructureStatus = {
  spannerStatus: ResourceStatus.Type.NOT_FOUND,
  pubsubStatus: FAKE_PUBSUB_STATUS_NOT_FOUND,
  dcpStatus: ResourceStatus.Type.NOT_FOUND
};

const FAKE_PUBSUB_STATUS_UNKNOWN: PubsubStatus = {
 list:  ResourceStatus.Type.RUNNING,
 listProgress: ResourceStatus.Type.RUNNING,
 uploadGCS: ResourceStatus.Type.RUNNING,
 uploadGCSProgress: ResourceStatus.Type.UNKNOWN
};

export const FAKE_INFRA_STATUS_UNKNOWN: InfrastructureStatus = {
  spannerStatus: ResourceStatus.Type.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_UNKNOWN,
  dcpStatus: ResourceStatus.Type.UNKNOWN
};

const FAKE_PUBSUB_STATUS_FAILED: PubsubStatus = {
 list:  ResourceStatus.Type.RUNNING,
 listProgress: ResourceStatus.Type.FAILED,
 uploadGCS: ResourceStatus.Type.RUNNING,
 uploadGCSProgress: ResourceStatus.Type.RUNNING
};

export const FAKE_INFRA_STATUS_FAILED: InfrastructureStatus = {
  spannerStatus: ResourceStatus.Type.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_UNKNOWN,
  dcpStatus: ResourceStatus.Type.FAILED
};

const FAKE_PUBSUB_STATUS_DEPLOYING: PubsubStatus = {
 list:  ResourceStatus.Type.DEPLOYING,
 listProgress: ResourceStatus.Type.NOT_FOUND,
 uploadGCS: ResourceStatus.Type.DEPLOYING,
 uploadGCSProgress: ResourceStatus.Type.DEPLOYING
};

export const FAKE_INFRA_STATUS_DEPLOYING: InfrastructureStatus = {
  spannerStatus: ResourceStatus.Type.NOT_FOUND,
  pubsubStatus: FAKE_PUBSUB_STATUS_DEPLOYING,
  dcpStatus: ResourceStatus.Type.NOT_FOUND
};

const FAKE_PUBSUB_STATUS_DELETING: PubsubStatus = {
 list:  ResourceStatus.Type.DELETING,
 listProgress: ResourceStatus.Type.RUNNING,
 uploadGCS: ResourceStatus.Type.DELETING,
 uploadGCSProgress: ResourceStatus.Type.DELETING
};

export const FAKE_INFRA_STATUS_DELETING: InfrastructureStatus = {
  spannerStatus: ResourceStatus.Type.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_DELETING,
  dcpStatus: ResourceStatus.Type.RUNNING
};

const FAKE_PUBSUB_STATUS_NOT_DETERMINED: PubsubStatus = {
 list:  ResourceStatus.Type.DEPLOYING,
 listProgress: ResourceStatus.Type.RUNNING,
 uploadGCS: ResourceStatus.Type.RUNNING,
 uploadGCSProgress: ResourceStatus.Type.DELETING
};

export const FAKE_INFRA_STATUS_NOT_DETERMINED: InfrastructureStatus = {
  spannerStatus: ResourceStatus.Type.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_NOT_DETERMINED,
  dcpStatus: ResourceStatus.Type.DELETING
};

export class InfrastructureServiceStub {
  public getInfrastructureStatus = jasmine.createSpy('getInfrastructureStatus');
  public postCreateInfrastructure = jasmine.createSpy('postCreateInfrastructure');
  public postTearDownInfrastructure = jasmine.createSpy('postTearDownInfrastructure');
}
