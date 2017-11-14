import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material';

import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { JobConfigAddDialogComponent } from '../job-config-add-dialog/job-config-add-dialog.component';
import { JobConfigResponse, SimpleDataSource } from '../jobs.resources';
import { JobsService } from '../jobs.service';

@Component({
  selector: 'app-job-configs',
  templateUrl: './job-configs.component.html',
  styleUrls: ['./job-configs.component.css']
})

export class JobConfigsComponent implements OnInit {
  showLoadingSpinner = false;
  errorMessage: string;
  errorTitle: string;
  displayErrorMessage = false;
  jobConfigs: JobConfigResponse[];

  displayedColumns = ['JobConfigId', 'onPremSrcDirectory', 'gcsBucket'];

  dataSource: SimpleDataSource<JobConfigResponse>;

  constructor(
      private readonly jobsService: JobsService,
      public dialog: MatDialog
  ) { }

  ngOnInit() {
    this.updateJobConfigs();
  }

  private updateJobConfigs(): void {
    this.showLoadingSpinner = true;
    this.jobsService.getJobConfigs().subscribe(
      (response: JobConfigResponse[]) => {
        this.jobConfigs = response;
        this.showLoadingSpinner = false;
        if (response.length === 0) {
          this.openAddJobConfigDialog();
        } else {
          this.dataSource = new SimpleDataSource(response);
          console.log(response);
        }
      },
      (error: HttpErrorResponse) => {
        this.errorTitle = HttpErrorResponseFormatter.getTitle(error);
        this.errorMessage = HttpErrorResponseFormatter.getMessage(error);
        if (error.status === 404) {
          // If error is not found, add a title that reminds the user to deploy their configs.
          this.errorTitle = `${this.errorTitle} (did you deploy your infrastructure yet?)`;
        }
        this.displayErrorMessage = true;
        this.showLoadingSpinner = false;
      });
  }

  getKeys(jsonObject: Object): String[] {
    return Object.keys(jsonObject);
  }

  openAddJobConfigDialog(): void {
    const jobConfigDialogReference = this.dialog.open(JobConfigAddDialogComponent, {
      width: '500px'
    });

    jobConfigDialogReference.afterClosed().subscribe(configSuccessfullyPosted => {
      if (configSuccessfullyPosted === true) {
        this.updateJobConfigs();
      }
    });
  }
}
