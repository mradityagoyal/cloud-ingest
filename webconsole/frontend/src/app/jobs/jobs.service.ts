import 'rxjs/add/operator/map';
import 'rxjs/add/operator/switchMap';

import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';

import { environment } from '../../environments/environment';
import { TaskFailureType } from '../proto/tasks.js';
import { TransferJob, Schedule, TransferJobResponse } from './jobs.resources';

const POST_HEADERS = {
    headers: new HttpHeaders().set('Content-Type', 'application/json')
};

@Injectable()
export class JobsService {
  private project: Observable<string>;

  constructor(private http: HttpClient, private route: ActivatedRoute) {
    this.project = route.queryParams.map(p => p.project);
  }

  /**
   * Gets a list of jobs.
   */
  getJobs(): Observable<TransferJobResponse> {
    return this.project.switchMap(projectId => {
        // Query all transfers.
        const paramString = JSON.stringify({project_id : projectId});
        return this.http.get<TransferJobResponse>(
            `${environment.apiUrl}/v1/transferJobs`,
             {params: {filter: paramString }});
    });
  }

  /**
   * Get the information of the latest job.
   */
  getJob(jobId: string): Observable<TransferJob> {
    return this.project.switchMap(projectId => {
        return this.http.get<TransferJob>(
            `${environment.apiUrl}/v1/${jobId}`,
            {params: {projectId}});
    });
  }

  /**
   * Creates a TransferJob from the input job parameter.
   */
  postJob(job: TransferJob): Observable<TransferJob> {
    return this.project.switchMap(projectId => {
        job.projectId = projectId;
        job.schedule = new Schedule();
        return this.http.post<TransferJob>(
            `${environment.apiUrl}/v1/transferJobs`,
            job, POST_HEADERS);
    });
  }
}
