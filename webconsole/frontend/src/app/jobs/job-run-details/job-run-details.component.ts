import { HttpErrorResponse } from '@angular/common/http';
import { Component, Inject, InjectionToken, OnDestroy, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material';
import { ActivatedRoute, Router } from '@angular/router';
import { interval } from 'rxjs';
import { takeWhile } from 'rxjs/operators';

import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { ErrorDialogContent } from '../../util/error-dialog/error-dialog.resources';
import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { OPERATION_STATUS_TO_STRING_MAP, TransferJob } from '../jobs.resources';
import { JobsService } from '../jobs.service';

const UPDATE_JOB_RUN_POLLING_INTERVAL_MILLISECONDS = 10000;
export const ENABLE_POLLING = new InjectionToken<boolean>('ENABLE_POLLING');

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
  // Used to control when the component should poll the job. Determines if the
  // component is alive.
  alive: boolean;

  showLoadingSpinner: boolean;

  // Needed to export this variable to the template.
  OPERATION_STATUS_TO_STRING_MAP = OPERATION_STATUS_TO_STRING_MAP;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private jobsService: JobsService,
    private dialog: MatDialog,
    @Inject(ENABLE_POLLING) public enablePolling: boolean,
  ) {
    this.jobId = route.snapshot.paramMap.get('jobId');
  }

  ngOnInit() {
    this.showLoadingSpinner = true;
    this.alive = true;
    this.initialJobLoad();
    if (this.enablePolling) {
      interval(UPDATE_JOB_RUN_POLLING_INTERVAL_MILLISECONDS).pipe(
        /**
         * TODO(b/66414686): This observable should not emit if the job has completed.
         */
        takeWhile(() => this.alive),
        ).subscribe(() => {
          this.updateJob();
        });
    }
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
