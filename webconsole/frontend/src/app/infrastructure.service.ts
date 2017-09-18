/**
 * @fileoverview Contains logic to interact with the infrastructure rest APIs.
 */
import { Injectable } from '@angular/core';
import { environment } from './../environments/environment';
import { Observable } from 'rxjs/Observable';
import { ActivatedRoute } from '@angular/router';
import { HttpResponse } from '@angular/common/http';
import { InfrastructureStatus } from './api.resources';
import { HttpClient, HttpHeaders } from '@angular/common/http';

/**
 * Returns true if all of the fields on the infrastructure status are in the statusList.
 */
function hasAllFieldsInStatusList(infraStructureStatus: InfrastructureStatus, statusList: string[]): boolean {
  let key: string;
  for (key of ['spannerStatus', 'dcpStatus', 'cloudFunctionsStatus']) {
    /**
     * Accessing properties by square bracket notation potentially violates the style guide. But,
     * this is more readable than accessing each property individually with the dot notation.
     */
    if (!statusList.includes(infraStructureStatus[key])) {
      return false;
    }
  }
  for (key of ['list', 'listProgress', 'uploadGCS', 'uploadGCSProgress', 'loadBigQuery', 'loadBigQueryProgress']) {
    if (!statusList.includes(infraStructureStatus.pubsubStatus[key])) {
      return false;
    }
  }
  return true;
}

/**
 * Checks if at least one field in the infrastructure status is of the input status.
 */
function hasAtLeastOneOfStatus(infraStructureStatus: InfrastructureStatus, status: string): boolean {
  let key: string;
  for (key of ['spannerStatus', 'dcpStatus', 'cloudFunctionsStatus']) {
    if (infraStructureStatus[key] === status) {
      return true;
    }
  }
  for (key of ['list', 'listProgress', 'uploadGCS', 'uploadGCSProgress', 'loadBigQuery', 'loadBigQueryProgress']) {
    if (infraStructureStatus.pubsubStatus[key] === status) {
      return true;
    }
  }
  return false;
}

export const INFRA_STATUS = {
  RUNNING : 'RUNNING',
  NOT_FOUND: 'NOT_FOUND',
  DEPLOYING: 'DEPLOYING',
  DELETING: 'DELETING',
  FAILED: 'FAILED',
  UNKNOWN: 'UNKNOWN'
};

@Injectable()
export class InfrastructureService {

  private projectId: Observable<string>;

  constructor(private http: HttpClient, private route: ActivatedRoute) {
    this.projectId = route.queryParams.map(p => p.project);
  }

  /**
   * Checks if everything in the infrastructure is of status RUNNING.
   */
  static isInfrastructureOk(infraStructureStatus: InfrastructureStatus): boolean {
    return hasAllFieldsInStatusList(infraStructureStatus, [INFRA_STATUS.RUNNING]);
  }

  /**
   * Checks if everything in the infrastructure is of status NOT_FOUND.
   */
  static isInfrastructureNotFound(infraStructureStatus: InfrastructureStatus): boolean {
    return hasAllFieldsInStatusList(infraStructureStatus, [INFRA_STATUS.NOT_FOUND]);
  }

  /**
   * Returns true if everything in the infrastructure is either DEPLOYING or NOT_FOUND and there
   * is at least one field that is DEPLOYING. Else, returns false.
   */
  static isInfrastructureDeploying(infraStructureStatus: InfrastructureStatus): boolean {
    return hasAllFieldsInStatusList(infraStructureStatus, [INFRA_STATUS.DEPLOYING, INFRA_STATUS.NOT_FOUND, INFRA_STATUS.RUNNING]) &&
        hasAtLeastOneOfStatus(infraStructureStatus, INFRA_STATUS.DEPLOYING);
  }

  /**
   * Returns true if everything in the infrastructure is either DELETING or RUNNING and there is
   * at least one field that is DELETING.
   */
  static isInfrastructureDeleting(infraStructureStatus: InfrastructureStatus): boolean {
    return hasAllFieldsInStatusList(infraStructureStatus, [INFRA_STATUS.DELETING, INFRA_STATUS.RUNNING, INFRA_STATUS.NOT_FOUND]) &&
        hasAtLeastOneOfStatus(infraStructureStatus, INFRA_STATUS.DELETING);
  }

  /**
   * Returns true if at least one of the infrastructure status is FAILED.
   */
  static isInfrastructureFailed(infrastructureStatus: InfrastructureStatus): boolean {
    return hasAtLeastOneOfStatus(infrastructureStatus, INFRA_STATUS.FAILED);
  }

  /**
   * Returns true if at least one of the infrastructure status is UNKNOWN.
   */
  static isInfrastructureUnknown(infrastructureStatus: InfrastructureStatus) {
    return hasAtLeastOneOfStatus(infrastructureStatus, INFRA_STATUS.UNKNOWN);
  }

  /**
   * Gets the infrastructure status from the backend.
   */
  getInfrastructureStatus(): Observable<InfrastructureStatus> {
    return this.projectId.switchMap(projectId => {
        return this.http.get<InfrastructureStatus>(
            `${environment.apiUrl}/projects/${projectId}/infrastructure-status`);
    });
  }

  postCreateInfrastructure(): Observable<{}> {
    return this.projectId.switchMap(projectId => {
        return this.http.post<{}>(
            `${environment.apiUrl}/projects/${projectId}/create-infrastructure`, {});
    });
  }

  postTearDownInfrastructure(): Observable<{}> {
  return this.projectId.switchMap(projectId => {
        return this.http.post<{}>(
            `${environment.apiUrl}/projects/${projectId}/tear-down-infrastructure`, {});
    });
  }

}
