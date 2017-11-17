/**
 * This file contains fakes for the tests in the infrastructure directory.
 */
import { InfrastructureStatus, PubsubStatus, INFRA_STATUS } from './infrastructure.resources';

const FAKE_PUBSUB_STATUS_RUNNING: PubsubStatus = {
 list:  INFRA_STATUS.RUNNING,
 listProgress: INFRA_STATUS.RUNNING,
 uploadGCS: INFRA_STATUS.RUNNING,
 uploadGCSProgress: INFRA_STATUS.RUNNING
};

export const FAKE_INFRA_STATUS_RUNNING: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_RUNNING,
  dcpStatus: INFRA_STATUS.RUNNING
};

const FAKE_PUBSUB_STATUS_NOT_FOUND: PubsubStatus = {
 list:  INFRA_STATUS.NOT_FOUND,
 listProgress: INFRA_STATUS.NOT_FOUND,
 uploadGCS: INFRA_STATUS.NOT_FOUND,
 uploadGCSProgress: INFRA_STATUS.NOT_FOUND
};

export const FAKE_INFRA_STATUS_NOT_FOUND: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.NOT_FOUND,
  pubsubStatus: FAKE_PUBSUB_STATUS_NOT_FOUND,
  dcpStatus: INFRA_STATUS.NOT_FOUND
};

const FAKE_PUBSUB_STATUS_UNKNOWN: PubsubStatus = {
 list:  INFRA_STATUS.RUNNING,
 listProgress: INFRA_STATUS.RUNNING,
 uploadGCS: INFRA_STATUS.RUNNING,
 uploadGCSProgress: INFRA_STATUS.UNKNOWN
};

export const FAKE_INFRA_STATUS_UNKNOWN: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_UNKNOWN,
  dcpStatus: INFRA_STATUS.UNKNOWN
};

const FAKE_PUBSUB_STATUS_FAILED: PubsubStatus = {
 list:  INFRA_STATUS.RUNNING,
 listProgress: INFRA_STATUS.FAILED,
 uploadGCS: INFRA_STATUS.RUNNING,
 uploadGCSProgress: INFRA_STATUS.RUNNING
};

export const FAKE_INFRA_STATUS_FAILED: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_UNKNOWN,
  dcpStatus: INFRA_STATUS.FAILED
};

const FAKE_PUBSUB_STATUS_DEPLOYING: PubsubStatus = {
 list:  INFRA_STATUS.DEPLOYING,
 listProgress: INFRA_STATUS.NOT_FOUND,
 uploadGCS: INFRA_STATUS.DEPLOYING,
 uploadGCSProgress: INFRA_STATUS.DEPLOYING
};

export const FAKE_INFRA_STATUS_DEPLOYING: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.NOT_FOUND,
  pubsubStatus: FAKE_PUBSUB_STATUS_DEPLOYING,
  dcpStatus: INFRA_STATUS.NOT_FOUND
};

const FAKE_PUBSUB_STATUS_DELETING: PubsubStatus = {
 list:  INFRA_STATUS.DELETING,
 listProgress: INFRA_STATUS.RUNNING,
 uploadGCS: INFRA_STATUS.DELETING,
 uploadGCSProgress: INFRA_STATUS.DELETING
};

export const FAKE_INFRA_STATUS_DELETING: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_DELETING,
  dcpStatus: INFRA_STATUS.RUNNING
};

const FAKE_PUBSUB_STATUS_NOT_DETERMINED: PubsubStatus = {
 list:  INFRA_STATUS.DEPLOYING,
 listProgress: INFRA_STATUS.RUNNING,
 uploadGCS: INFRA_STATUS.RUNNING,
 uploadGCSProgress: INFRA_STATUS.DELETING
};

export const FAKE_INFRA_STATUS_NOT_DETERMINED: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_NOT_DETERMINED,
  dcpStatus: INFRA_STATUS.DELETING
};
