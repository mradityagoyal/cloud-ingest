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
import { TransferJob, OPERATION_STATUS_TO_STRING_MAP } from '../jobs.resources';
import { JobsService } from '../jobs.service';

const UPDATE_JOB_RUN_POLLING_INTERVAL_MILLISECONDS = 10000;

@Component({
  selector: 'app-job-run-details',
  templateUrl: './job-run-details.component.html',
  styleUrls: ['./job-run-details.component.css']
})
export class JobRunDetailsComponent implements OnInit, OnDestroy {
  job: TransferJob;
  errorTitle: string;
  errorMessage: string;
  showError: boolean;
  jobId: string;
  alive: boolean; // Used to control when the component should poll the job.

  showLoadingSpinner: boolean;

  // Needed to export this variable to the template.
  OPERATION_STATUS_TO_STRING_MAP = OPERATION_STATUS_TO_STRING_MAP;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private jobsService: JobsService,
    private dialog: MatDialog
  ) {
    this.jobId = route.snapshot.paramMap.get('jobId');
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
      this.updateJob();
    });
  }

  ngOnDestroy() {
    this.alive = false;
  }

  private initialJobLoad() {
    // Combine the observables that get the job run and job config.
    this.jobsService.getJob('transferJobs/' + this.jobId)
    .subscribe((response: TransferJob) => {
      this.job = response;
      this.showLoadingSpinner = false;
    }, (error: HttpErrorResponse) => {
      this.errorTitle = HttpErrorResponseFormatter.getTitle(error);
      this.errorMessage = HttpErrorResponseFormatter.getMessage(error);
      this.showError = true;
      this.showLoadingSpinner = false;
    });
  }

  private updateJob() {
    this.jobsService.getJob('transferJobs/' + this.jobId).subscribe(
      (response: TransferJob) => {
        this.job = response;
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
