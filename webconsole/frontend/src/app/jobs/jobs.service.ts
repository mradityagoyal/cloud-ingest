import 'rxjs/add/operator/map';
import 'rxjs/add/operator/switchMap';

import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Response } from '@angular/http';
import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { forkJoin } from 'rxjs/observable/forkJoin';
import { mergeMap } from 'rxjs/operators';
import { zip } from 'rxjs/observable/zip';
import { combineLatest } from 'rxjs/operators';
import { environment } from '../../environments/environment';
import { TaskFailureType } from '../proto/tasks.js';
import { TransferJob, Schedule, TransferJobResponse, PauseTransferJobRequest,
  ResumeTransferJobRequest, DeleteTransferJobRequest } from './jobs.resources';

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

  /**
   * Returns an observable that pauses an input job.
   */
  private pauseJob(job: string): Observable<TransferJob> {
    return this.project.switchMap(projectId => {
      const pauseTransferJobRequest: PauseTransferJobRequest = {
        name: job,
        projectId: projectId,
      };
      return this.http.post<TransferJob>(`${environment.apiUrl}/v1/${job}:pause`,
      pauseTransferJobRequest, POST_HEADERS);
    });
  }

  /**
   * Returns an observable that resumes an input job.
   */
  private resumeJob(job: string): Observable<TransferJob> {
    return this.project.switchMap(projectId => {
      const resumeTransferJobRequest: ResumeTransferJobRequest = {
        name: job,
        projectId: projectId,
      };
      return this.http.post<TransferJob>(`${environment.apiUrl}/v1/${job}:resume`,
      resumeTransferJobRequest, POST_HEADERS);
    });
  }

  /**
   * Returns an observable that deletes an input job.
   */
  private deleteJob(job: string): Observable<Response> {
    return this.project.switchMap(projectId => {
      const deleteTransferJobRequest: DeleteTransferJobRequest = {
        jobName: job,
        projectId: projectId,
      };
      return this.http.request<Response>('delete', `${environment.apiUrl}/v1/${job}`, {
        body: deleteTransferJobRequest
      });
    });
  }

  pauseJobs(jobs: string[]): Observable<TransferJob[]> {
    const pauseJobRequests = [];
    for (const job of jobs) {
      pauseJobRequests.push(this.pauseJob(job));
    }
    return Observable.combineLatest(pauseJobRequests).take(1);
  }

  resumeJobs(jobs: string[]): Observable<TransferJob[]> {
    const resumeJobRequests = [];
    for (const job of jobs) {
      resumeJobRequests.push(this.resumeJob(job));
    }
    return Observable.combineLatest(resumeJobRequests).take(1);
  }

  deleteJobs(jobs: string[]): Observable<Response[]> {
    const deleteJobRequests = [];
    for (const job of jobs) {
      deleteJobRequests.push(this.deleteJob(job));
    }
    return Observable.combineLatest(deleteJobRequests).take(1);
  }
}
