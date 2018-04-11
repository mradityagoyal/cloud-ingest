import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit } from '@angular/core';
import { MatCheckboxChange, MatDialog } from '@angular/material';

import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { ErrorDialogContent } from '../../util/error-dialog/error-dialog.resources';
import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { JobConfigAddDialogComponent } from '../job-config-add-dialog/job-config-add-dialog.component';
import { TransferJob, SimpleDataSource, OPERATION_STATUS_TO_STRING_MAP, TransferJobResponse } from '../jobs.resources';
import { JobsService } from '../jobs.service';

@Component({
  selector: 'app-job-configs',
  templateUrl: './job-configs.component.html',
  styleUrls: ['./job-configs.component.css']
})

export class JobConfigsComponent implements OnInit {
  showLoadingSpinner = true;
  errorMessage: string;
  errorTitle: string;
  displayErrorMessage = false;
  jobs: TransferJob[];

  // Need to declare this variable here to use it in the template.
  OPERATION_STATUS_TO_STRING_MAP = OPERATION_STATUS_TO_STRING_MAP;

  /**
   * Passed to the add job configuration dialog.
   */
  job = new TransferJob();

  displayedColumns = ['JobDescription', 'onPremSrcDirectory', 'gcsBucket',
      'Status'];

  dataSource: SimpleDataSource<TransferJob>;

  constructor(
      private readonly jobsService: JobsService,
      public dialog: MatDialog
  ) { }

  ngOnInit() {
    this.updateJobs();
  }

  private updateJobs(): void {
    this.showLoadingSpinner = true;
    this.jobsService.getJobs().subscribe(
      (response: TransferJobResponse) => {
        if (!response.transferJobs) {
          this.jobs = [];
        } else {
          this.jobs = response.transferJobs;
        }
        this.showLoadingSpinner = false;
        if (this.jobs.length === 0) {
          this.openAddJobConfigDialog();
        } else {
          this.dataSource = new SimpleDataSource(this.jobs);
        }
      },
      (error: HttpErrorResponse) => {
        this.errorTitle = HttpErrorResponseFormatter.getTitle(error);
        this.errorMessage = HttpErrorResponseFormatter.getMessage(error);
        this.displayErrorMessage = true;
        this.showLoadingSpinner = false;
      });
  }

  getKeys(jsonObject: Object): String[] {
    return Object.keys(jsonObject);
  }

  openAddJobConfigDialog(): void {
    const jobConfigDialogReference = this.dialog.open(JobConfigAddDialogComponent, {
      width: '500px',
      data: this.job
    });

    jobConfigDialogReference.afterClosed().subscribe(configSuccessfullyPosted => {
      if (configSuccessfullyPosted === true) {
        this.updateJobs();
      }
    });
  }

}
