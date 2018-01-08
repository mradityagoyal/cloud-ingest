/**
 * This file contains fakes for the tests in the infrastructure directory.
 */
import { InfrastructureStatus, PubsubStatus } from './infrastructure.resources';
import { ResourceStatus } from '../proto/tasks.js';

const FAKE_PUBSUB_STATUS_RUNNING: PubsubStatus = {
 list:  ResourceStatus.Type.RUNNING,
 listProgress: ResourceStatus.Type.RUNNING,
 copy: ResourceStatus.Type.RUNNING,
 copyProgress: ResourceStatus.Type.RUNNING
};

export const FAKE_INFRA_STATUS_RUNNING: InfrastructureStatus = {
  pubsubStatus: FAKE_PUBSUB_STATUS_RUNNING
};

const FAKE_PUBSUB_STATUS_NOT_FOUND: PubsubStatus = {
 list:  ResourceStatus.Type.NOT_FOUND,
 listProgress: ResourceStatus.Type.NOT_FOUND,
 copy: ResourceStatus.Type.NOT_FOUND,
 copyProgress: ResourceStatus.Type.NOT_FOUND
};

export const FAKE_INFRA_STATUS_NOT_FOUND: InfrastructureStatus = {
  pubsubStatus: FAKE_PUBSUB_STATUS_NOT_FOUND
};

const FAKE_PUBSUB_STATUS_UNKNOWN: PubsubStatus = {
 list:  ResourceStatus.Type.RUNNING,
 listProgress: ResourceStatus.Type.RUNNING,
 copy: ResourceStatus.Type.RUNNING,
 copyProgress: ResourceStatus.Type.UNKNOWN
};

export const FAKE_INFRA_STATUS_UNKNOWN: InfrastructureStatus = {
  pubsubStatus: FAKE_PUBSUB_STATUS_UNKNOWN
};

const FAKE_PUBSUB_STATUS_FAILED: PubsubStatus = {
 list:  ResourceStatus.Type.RUNNING,
 listProgress: ResourceStatus.Type.FAILED,
 copy: ResourceStatus.Type.RUNNING,
 copyProgress: ResourceStatus.Type.RUNNING
};

export const FAKE_INFRA_STATUS_FAILED: InfrastructureStatus = {
  pubsubStatus: FAKE_PUBSUB_STATUS_FAILED
};

const FAKE_PUBSUB_STATUS_DEPLOYING: PubsubStatus = {
 list:  ResourceStatus.Type.DEPLOYING,
 listProgress: ResourceStatus.Type.NOT_FOUND,
 copy: ResourceStatus.Type.DEPLOYING,
 copyProgress: ResourceStatus.Type.DEPLOYING
};

export const FAKE_INFRA_STATUS_DEPLOYING: InfrastructureStatus = {
  pubsubStatus: FAKE_PUBSUB_STATUS_DEPLOYING
};

const FAKE_PUBSUB_STATUS_DELETING: PubsubStatus = {
 list:  ResourceStatus.Type.DELETING,
 listProgress: ResourceStatus.Type.RUNNING,
 copy: ResourceStatus.Type.DELETING,
 copyProgress: ResourceStatus.Type.DELETING
};

export const FAKE_INFRA_STATUS_DELETING: InfrastructureStatus = {
  pubsubStatus: FAKE_PUBSUB_STATUS_DELETING
};

const FAKE_PUBSUB_STATUS_NOT_DETERMINED: PubsubStatus = {
 list:  ResourceStatus.Type.DEPLOYING,
 listProgress: ResourceStatus.Type.RUNNING,
 copy: ResourceStatus.Type.RUNNING,
 copyProgress: ResourceStatus.Type.DELETING
};

export const FAKE_INFRA_STATUS_NOT_DETERMINED: InfrastructureStatus = {
  pubsubStatus: FAKE_PUBSUB_STATUS_NOT_DETERMINED
};

export class InfrastructureServiceStub {
  public getInfrastructureStatus = jasmine.createSpy('getInfrastructureStatus');
  public postCreateInfrastructure = jasmine.createSpy('postCreateInfrastructure');
  public postTearDownInfrastructure = jasmine.createSpy('postTearDownInfrastructure');
}
