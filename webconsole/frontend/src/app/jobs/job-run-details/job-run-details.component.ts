import 'rxjs/add/operator/takeWhile';

import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnDestroy, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material';
import { ActivatedRoute, Router } from '@angular/router';
import { IntervalObservable } from 'rxjs/observable/IntervalObservable';
import { Observable } from 'rxjs/Rx';

import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { ErrorDialogContent } from '../../util/error-dialog/error-dialog.resources';
import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { JobConfigResponse, JobRun } from '../jobs.resources';
import { JobsService } from '../jobs.service';

const UPDATE_JOB_RUN_POLLING_INTERVAL_MILLISECONDS = 10000;

@Component({
  selector: 'app-job-run-details',
  templateUrl: './job-run-details.component.html',
  styleUrls: ['./job-run-details.component.css']
})
export class JobRunDetailsComponent implements OnInit, OnDestroy {
  jobRun: JobRun;
  errorTitle: string;
  errorMessage: string;
  showError: boolean;
  jobConfigId: string;
  jobRunId: string;
  alive: boolean; // Used to control when the component should poll the job run info.

  // The job config object that corresponds to this job run.
  jobConfig: JobConfigResponse;

  showLoadingSpinner: boolean;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private jobService: JobsService,
    private dialog: MatDialog
  ) {
    this.jobConfigId = route.snapshot.paramMap.get('configId');
    this.jobRunId = this.route.snapshot.paramMap.get('runId');
  }

  ngOnInit() {
    this.showLoadingSpinner = true;
    this.alive = true;
    // this.getJobConfig();
    this.initialJobLoad();
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

  private initialJobLoad() {
    // Combine the observables that get the job run and job config.
    this.jobService.getJobRun(this.jobConfigId)
    .subscribe((response: JobRun) => {
      this.jobRun = response;
      this.showLoadingSpinner = false;
    }, (error: HttpErrorResponse) => {
      this.errorTitle = HttpErrorResponseFormatter.getTitle(error);
      this.errorMessage = HttpErrorResponseFormatter.getMessage(error);
      this.showError = true;
      this.showLoadingSpinner = false;
    });
  }

  private updateJobRun() {
    this.jobService.getJobRun(this.jobConfigId)
    .subscribe(
      (response: JobRun) => {
        this.jobRun = response;
      },
      (error: HttpErrorResponse) => {
        const errorContent: ErrorDialogContent = {
          errorTitle: HttpErrorResponseFormatter.getTitle(error),
          errorMessage: HttpErrorResponseFormatter.getMessage(error)
        };
        this.dialog.open(ErrorDialogComponent, {
          data: errorContent
        });
      });
  }
}
