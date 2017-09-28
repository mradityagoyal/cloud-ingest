import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable } from 'rxjs/Observable';
import { JobRun } from '../jobs.resources';
import { JobsService } from '../jobs.service';
import { JobStatusPipe } from '../job-status/job-status.pipe';
import { IntervalObservable } from 'rxjs/observable/IntervalObservable';
import 'rxjs/add/operator/takeWhile';

const UPDATE_JOB_RUN_POLLING_INTERVAL_MILLISECONDS = 3000;

@Component({
  selector: 'app-job-run-details',
  templateUrl: './job-run-details.component.html',
  styleUrls: ['./job-run-details.component.css']
})
export class JobRunDetailsComponent implements OnInit, OnDestroy {
  private jobRun: JobRun;
  private showLoadingSpinner: boolean;
  private errorTitle: string;
  private errorMessage: string;
  private showError: boolean;
  private jobConfigId: string;
  private jobRunId: string;
  private alive: boolean; // Used to control when the component should poll the job run info.

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private jobService: JobsService
  ) {
    this.jobConfigId = route.snapshot.paramMap.get('configId');
    this.jobRunId = this.route.snapshot.paramMap.get('runId');
  }

  ngOnInit() {
    this.showLoadingSpinner = true;
    this.alive = true;
    this.updateJobRun();
    IntervalObservable.create(UPDATE_JOB_RUN_POLLING_INTERVAL_MILLISECONDS)
    /**
     * TODO(b/66414686): This observable should not emit if the job has completed.
     */
    .takeWhile(() => this.alive)
    .subscribe(() => {
      this.updateJobRun();
    });
  }

  ngOnDestroy() {
    this.alive = false;
  }

  private updateJobRun() {
    this.jobService.getJobRun(this.jobConfigId, this.jobRunId)
    .subscribe(
      (response: JobRun) => {
        this.handleGetJobRunResponse(response);
      },
      (error: HttpErrorResponse) => {
        this.handleGetJobRunErrorResponse(error);
      }
    );
  }

  /**
   * Updates the state of the Job Run details after a successful call to getJobRun.
   *
   * @param{response} The http response from getJobRun.
   */
  private handleGetJobRunResponse(response: JobRun) {
    this.jobRun = response;
    this.showLoadingSpinner = false;
    this.showError = false;
  }
  /**
   * Updates the state of the Job Run details after an error.
   *
   * @param{response} The http error response from getJobRun.
   */
  private handleGetJobRunErrorResponse(error: HttpErrorResponse) {
    /**
     * TODO(b/66416361): Job run details page should not retry to get the job run if the error is
     * not retry-able.
     */
    if (error.error instanceof Error) {
      // A client-side or network error occurred
      console.error('An error occurred requesting the job run details:', error.error.message);
    } else {
      // Back-end returned an unsuccessful response code
      this.errorTitle = error.error.error;
      this.errorMessage = error.error.message;
      if (!this.errorMessage) {
        this.errorTitle = error.statusText;
        this.errorMessage = error.message;
      }
      this.showError = true;
      this.showLoadingSpinner = false;
    }
  }
}
