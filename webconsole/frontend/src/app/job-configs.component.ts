import { Component, OnInit } from '@angular/core';
import { JobConfig } from './api.resources';
import { JobsService } from './jobs.service';
import { MdDialog } from '@angular/material';
import { HttpErrorResponse } from '@angular/common/http';
import { JobConfigAddDialogComponent } from './job-config-add-dialog.component';

@Component({
  selector: 'app-job-configs',
  templateUrl: './job-configs.component.html',
  styleUrls: ['./job-configs.component.css']
})

export class JobConfigsComponent implements OnInit {
  jobConfigs: JobConfig[];
  showLoadingSpinner = false;
  errorMessage: string;
  errorTitle: string;
  displayErrorMessage = false;

  constructor(
      private readonly jobsService: JobsService,
      public dialog: MdDialog
  ) { }

  ngOnInit() {
    this.updateJobConfigs();
  }

  private updateJobConfigs(): void {
    this.showLoadingSpinner = true;
    this.jobsService.getJobConfigs().subscribe(
      (response) => {
        this.jobConfigs = response;
        this.showLoadingSpinner = false;
        if (this.jobConfigs.length === 0) {
          this.openAddJobConfigDialog();
        }
      },
      (error: HttpErrorResponse) => {
        if (typeof error.error === 'string') {
          this.errorTitle = error.error;
        } else {
          this.errorTitle = error.statusText;
        }
        this.errorMessage = error.message;
        this.displayErrorMessage = true;
        this.showLoadingSpinner = false;
      });
  }

  private getKeys(jsonObject: Object): String[] {
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
