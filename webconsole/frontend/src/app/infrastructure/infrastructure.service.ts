/**
 * @fileoverview Contains logic to interact with the infrastructure rest APIs.
 */
import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';

import { environment } from '../../environments/environment';
import { InfrastructureStatus } from './infrastructure.resources';
import { ResourceStatus } from '../proto/tasks.js';


function hasAtLeastOneOfStatus(infraStatusList: ResourceStatus.Type[], status: ResourceStatus.Type): boolean {
  for (const value of infraStatusList) {
    if (value === status) {
      return true;
    }
  }
  return false;
}

function hasAllFieldsInStatusList(infraStatusList: ResourceStatus.Type[], statusList: ResourceStatus.Type[]): boolean {
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
  static getOverallStatus(statusList: ResourceStatus.Type[]): ResourceStatus.Type | null {
    if (hasAtLeastOneOfStatus(statusList, ResourceStatus.Type.FAILED)) {
      return ResourceStatus.Type.FAILED;
    } else if (hasAtLeastOneOfStatus(statusList, ResourceStatus.Type.UNKNOWN)) {
      return ResourceStatus.Type.UNKNOWN;
    } else if (hasAllFieldsInStatusList(statusList, [ResourceStatus.Type.NOT_FOUND])) {
      return ResourceStatus.Type.NOT_FOUND;
    } else if (hasAllFieldsInStatusList(statusList, [ResourceStatus.Type.RUNNING])) {
      return ResourceStatus.Type.RUNNING;
    } else if (hasAllFieldsInStatusList(statusList, [ResourceStatus.Type.NOT_FOUND, ResourceStatus.Type.DEPLOYING,
        ResourceStatus.Type.RUNNING])) {
      return ResourceStatus.Type.DEPLOYING;
    } else if (hasAllFieldsInStatusList(statusList, [ResourceStatus.Type.RUNNING, ResourceStatus.Type.DELETING,
        ResourceStatus.Type.NOT_FOUND])) {
      return ResourceStatus.Type.DELETING;
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
