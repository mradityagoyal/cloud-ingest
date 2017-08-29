import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Router, ActivatedRoute, Params } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { JobRun, JobConfig, JobRunParams } from './api.resources';
import { environment } from './../environments/environment';

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
}
