import 'rxjs/add/operator/map';
import 'rxjs/add/operator/switchMap';

import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';

import { environment } from '../../environments/environment';
import { TaskFailureType } from '../proto/tasks.js';
import { JobConfigRequest, JobConfigResponse, JobRun, JobRunParams, Task } from './jobs.resources';

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

  getJobRuns(): Observable<JobRun[]> {
    return this.project.switchMap(projectId => {
        return this.http.get<JobRun[]>(
            `${environment.apiUrl}/projects/${projectId}/jobruns`);
    });
  }

  getJobRun(configId: string, runId: string): Observable<JobRun> {
    return this.project.switchMap(projectId => {
        return this.http.get<JobRun>(
            `${environment.apiUrl}/projects/${projectId}/jobruns/${configId}/${runId}`
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

  postJobRun(jobParams: JobRunParams): Observable<JobRun> {
    return this.project.switchMap(projectId => {
        return this.http.post<JobRun>(
            `${environment.apiUrl}/projects/${projectId}/jobruns`,
            jobParams, POST_HEADERS);
    });
  }

  getTasksOfStatus(configId: string, runId: string, status: number, lastModifiedBefore?: number): Observable<Task[]> {
    let requestParameters = new HttpParams();
    if (lastModifiedBefore != null) {
        requestParameters = requestParameters.set('lastModifiedBefore', String(lastModifiedBefore));
    }
    return this.project.switchMap(projectId => {
        return this.http.get<Task[]>(
            `${environment.apiUrl}/projects/${projectId}/tasks/${configId}/${runId}/status/${status}`,
            {params: requestParameters}
        );
    });
  }

  getTasksOfFailureType(
    configId: string,
    runId: string,
    failureType: TaskFailureType.Type): Observable<Task[]> {
    return this.project.switchMap(projectId => {
        return this.http.get<Task[]>(
            `${environment.apiUrl}/projects/${projectId}/tasks/${configId}/${runId}/failuretype/${failureType}`
        );
    });
  }
}
