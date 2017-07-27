import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs/Observable';
import { JobRun, JobConfig, JobRunParams } from './api.resources';
import { environment } from './../environments/environment';

@Injectable()
export class JobsService {
  constructor(private http: HttpClient) {}

  getJobConfigs(): Observable<JobConfig[]> {
    return this.http.get<JobConfig[]>(environment.apiUrl + '/jobconfigs');
  }

  getJobRuns(): Observable<JobRun[]> {
    return this.http.get<JobRun[]>(environment.apiUrl + '/jobruns');
  }

  postJobConfig(config: JobConfig): Observable<JobConfig> {
    return this.http.post<JobConfig>(environment.apiUrl + '/jobconfigs', config,
        { headers: new HttpHeaders().set('Content-Type', 'application/json')});
  }

  postJobRun(jobParams: JobRunParams): Observable<JobRun> {
    return this.http.post<JobRun>(environment.apiUrl + '/jobruns', jobParams,
        { headers: new HttpHeaders().set('Content-Type', 'application/json')});
  }
}
