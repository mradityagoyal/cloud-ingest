/**
 * @fileoverview Contains logic to interact with the infrastructure rest APIs.
 */
import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';

import { environment } from '../../environments/environment';
import { INFRA_STATUS, InfrastructureStatus } from './infrastructure.resources';

function hasAtLeastOneOfStatus(infraStatusList: string[], status: string): boolean {
  for (const value of infraStatusList) {
    if (value === status) {
      return true;
    }
  }
  return false;
}

function hasAllFieldsInStatusList(infraStatusList: string[], statusList: string[]): boolean {
  for (const value of infraStatusList) {
    if (!statusList.includes(value)) {
      return false;
    }
  }
  return true;
}

@Injectable()
export class InfrastructureService {

  private projectId: Observable<string>;

  constructor(private http: HttpClient, private route: ActivatedRoute) {
    this.projectId = route.queryParams.map(p => p.project);
  }

  /**
   * Infers the overall status from a list of status.
   *
   * @param statusList The list of infrastructure status to infer the overall status from.
   */
  static getOverallStatus(statusList: string[]): string | null {
    if (hasAtLeastOneOfStatus(statusList, INFRA_STATUS.FAILED)) {
      return INFRA_STATUS.FAILED;
    } else if (hasAtLeastOneOfStatus(statusList, INFRA_STATUS.UNKNOWN)) {
      return INFRA_STATUS.UNKNOWN;
    } else if (hasAllFieldsInStatusList(statusList, [INFRA_STATUS.NOT_FOUND])) {
      return INFRA_STATUS.NOT_FOUND;
    } else if (hasAllFieldsInStatusList(statusList, [INFRA_STATUS.RUNNING])) {
      return INFRA_STATUS.RUNNING;
    } else if (hasAllFieldsInStatusList(statusList, [INFRA_STATUS.NOT_FOUND, INFRA_STATUS.DEPLOYING, INFRA_STATUS.RUNNING])) {
      return INFRA_STATUS.DEPLOYING;
    } else if (hasAllFieldsInStatusList(statusList, [INFRA_STATUS.RUNNING, INFRA_STATUS.DELETING, INFRA_STATUS.NOT_FOUND])) {
      return INFRA_STATUS.DELETING;
    } else {
      // The overall status could not be determined.
      return null;
    }
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
