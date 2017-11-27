import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit } from '@angular/core';
import { MatCheckboxChange, MatDialog } from '@angular/material';

import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { ErrorDialogContent } from '../../util/error-dialog/error-dialog.resources';
import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { JobConfigAddDialogComponent } from '../job-config-add-dialog/job-config-add-dialog.component';
import { JobConfigRequest, JobConfigResponse, SimpleDataSource } from '../jobs.resources';
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

  /**
   * A map of jobConfigId -> isChecked. Indicates if the box for a particular config id has been
   * checked. If the key does not exist, it hasn't been checked.
   */
  checkedCheckboxes: { [key: string]: boolean; } = {};

  /**
   * The number of checkboxes that have been checked.
   */
  numChecked = 0;

  /**
   * An object to pass to the add job configuration dialog with starting information for the
   * dialog.
   */
  startingJobConfig: JobConfigRequest = new JobConfigRequest(/*jobConfigId*/'',
      /*gcsBucket*/'', /*fileSystemDirectory*/'');

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
      width: '500px',
      data: this.startingJobConfig
    });

    jobConfigDialogReference.afterClosed().subscribe(configSuccessfullyPosted => {
      if (configSuccessfullyPosted === true) {
        this.updateJobConfigs();
      }
    });
  }

  onCheckboxClick(event: MatCheckboxChange) {
    let count = 0;
    for (const key in this.checkedCheckboxes) {
      if (this.checkedCheckboxes[key] === true) {
        count++;
      }
    }
    this.numChecked = count;
  }

  deleteJobConfigs() {
    const selectedJobConfigs = [];
    for (const key in this.checkedCheckboxes) {
      if (this.checkedCheckboxes[key] === true) {
        selectedJobConfigs.push(key);
      }
    }
    this.jobsService.deleteJobConfigs(selectedJobConfigs).subscribe(
      (response) => {
        this.updateJobConfigs();
      }, (errorResponse: HttpErrorResponse) => {
        this.updateJobConfigs();
        const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
        const errorMessage = HttpErrorResponseFormatter.getMessage(errorResponse);
        const errorContent: ErrorDialogContent = {
          errorTitle: errorTitle,
          errorMessage: errorMessage
        };
        this.dialog.open(ErrorDialogComponent, {
          data: errorContent
        });
      });
  }

  /**
   * Handles the user click on the clone existing configuration button.
   */
  cloneExistingConfig() {
    let selectedJobConfigId: string;
    for (const key in this.checkedCheckboxes) {
      if (this.checkedCheckboxes[key] === true) {
        selectedJobConfigId = key;
        break;
      }
    }
    for (const jobConfig of this.jobConfigs) {
      if (jobConfig.JobConfigId === selectedJobConfigId) {
        this.startingJobConfig.jobConfigId = jobConfig.JobConfigId + ' copy';
        this.startingJobConfig.gcsBucket = jobConfig.JobSpec.gcsBucket;
        this.startingJobConfig.fileSystemDirectory = jobConfig.JobSpec.onPremSrcDirectory;
        break;
      }
    }
    this.openAddJobConfigDialog();
  }
}
