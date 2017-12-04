import 'rxjs/add/operator/map';
import 'rxjs/add/operator/switchMap';

import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';

import { environment } from '../../environments/environment';
import { TaskFailureType } from '../proto/tasks.js';
import { JobConfigRequest, JobConfigResponse, JobRun, Task } from './jobs.resources';

const POST_HEADERS = {
    headers: new HttpHeaders().set('Content-Type', 'application/json')
};

@Injectable()
export class JobsService {
  private project: Observable<string>;

  constructor(private http: HttpClient, private route: ActivatedRoute) {
    this.project = route.queryParams.map(p => p.project);
  }

  getJobConfigs(): Observable<JobConfigResponse[]> {
    return this.project.switchMap(projectId => {
        return this.http.get<JobConfigResponse[]>(
            `${environment.apiUrl}/projects/${projectId}/jobconfigs`);
    });
  }

  getJobRun(configId: string): Observable<JobRun> {
    return this.project.switchMap(projectId => {
        return this.http.get<JobRun>(
            `${environment.apiUrl}/projects/${projectId}/jobrun/${configId}`
        );
    });
  }

  postJobConfig(config: JobConfigRequest): Observable<JobConfigResponse> {
    return this.project.switchMap(projectId => {
        return this.http.post<JobConfigResponse>(
            `${environment.apiUrl}/projects/${projectId}/jobconfigs`,
            config, POST_HEADERS);
    });
  }

  getTasksOfStatus(configId: string, status: number, lastModifiedBefore?: number): Observable<Task[]> {
    let requestParameters = new HttpParams();
    if (lastModifiedBefore != null) {
        requestParameters = requestParameters.set('lastModifiedBefore', String(lastModifiedBefore));
    }
    return this.project.switchMap(projectId => {
        return this.http.get<Task[]>(
            `${environment.apiUrl}/projects/${projectId}/tasks/${configId}/status/${status}`,
            {params: requestParameters}
        );
    });
  }

  getTasksOfFailureType(
    configId: string,
    failureType: TaskFailureType.Type,
    lastModifiedBefore?: number): Observable<Task[]> {
    let requestParameters = new HttpParams();
    if (lastModifiedBefore != null) {
        requestParameters = requestParameters.set('lastModifiedBefore', String(lastModifiedBefore));
    }
    return this.project.switchMap(projectId => {
        return this.http.get<Task[]>(
            `${environment.apiUrl}/projects/${projectId}/tasks/${configId}/failuretype/${failureType}`
        );
    });
  }

  deleteJobConfigs(
      configIdList: string[]
  ): Observable<string[]> {
    return this.project.switchMap(projectId => {
        return this.http.post<string[]>(
            `${environment.apiUrl}/projects/${projectId}/jobconfigs/delete`,
            configIdList, POST_HEADERS);
    });
  }
}
