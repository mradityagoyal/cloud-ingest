import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Router, ActivatedRoute, Params } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { JobRun, JobConfig, JobRunParams, Task } from './jobs.resources';
import { environment } from '../../environments/environment';

import 'rxjs/add/operator/switchMap';
import 'rxjs/add/operator/map';

const POST_HEADERS = {
    headers: new HttpHeaders().set('Content-Type', 'application/json')
};

@Injectable()
export class JobsService {
  private project: Observable<string>;

  constructor(private http: HttpClient, private route: ActivatedRoute) {
    this.project = route.queryParams.map(p => p.project);
  }

  getJobConfigs(): Observable<JobConfig[]> {
    return this.project.switchMap(projectId => {
        return this.http.get<JobConfig[]>(
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

  postJobConfig(config: JobConfig): Observable<JobConfig> {
    return this.project.switchMap(projectId => {
        return this.http.post<JobConfig>(
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

  /**
   * Gets the first 25 tasks of the input status.
   *
   * TODO(b/67581174): The job tasks component should paginate to retrieve all of the tasks.
   */
  getTasksOfStatus(configId: string, runId: string, status: number): Observable<Task[]> {
    const params = new HttpParams().set('status', String(status));
    return this.project.switchMap(projectId => {
        return this.http.get<Task[]>(
            `${environment.apiUrl}/projects/${projectId}/tasks/${configId}/${runId}`,
            {params: params}
        );
    });
  }
}
