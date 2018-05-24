import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit } from '@angular/core';
import { MatCheckboxChange, MatDialog, MatTooltipModule } from '@angular/material';

import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { ErrorDialogContent } from '../../util/error-dialog/error-dialog.resources';
import { Response } from '@angular/http';
import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { JobConfigAddDialogComponent } from '../job-config-add-dialog/job-config-add-dialog.component';
import { TransferJob, SimpleDataSource, OPERATION_STATUS_TO_STRING_MAP, TransferJobResponse } from '../jobs.resources';
import { JobsService } from '../jobs.service';
import { IfObservable } from 'rxjs/observable/IfObservable';
import { Observable } from 'rxjs/Observable';

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
  /**
   * Indicates if the jobs that are currently checked can be deleted or not.
   */
  checkedJobsCanBeDeleted = false;

  /**
   * Holds a map from a job name to its corresponding transfer job.
   */
  jobNameToJobMap: Map<string, TransferJob>;

  /**
   * If the number of requested jobs is more than 0 this boolean is set to true,
   * otherwise, false.
   */
  hasJobs = false;

  /**
   * A map of jobConfigId -> isChecked. Indicates if the box for a particular config id has been
   * checked. If the key does not exist, it hasn't been checked.
   * Passed to the add job configuration dialog.
   */
  checkedCheckboxes: { [key: string]: boolean; } = {};

  /**
   * The number of checkboxes that have been checked.
   */
  numChecked = 0;

  // Need to declare this variable here to use it in the template.
  OPERATION_STATUS_TO_STRING_MAP = OPERATION_STATUS_TO_STRING_MAP;

  /**
   * A list of the statuses when the job is safe to delete.
   */
  readonly canDeleteStatuses = ['PAUSED', 'FAILED', 'SUCCESS', 'ABORTED'];

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

  updateJobs(): void {
    this.showLoadingSpinner = true;
    this.jobsService.getJobs().subscribe(
      (response: TransferJobResponse) => {
        if (!response.transferJobs) {
          this.hasJobs = false;
          this.openAddJobConfigDialog();
        } else {
          this.hasJobs = true;
          this.jobNameToJobMap = new Map<string, TransferJob>(
            response.transferJobs.map(x => [x.name, x] as [string, TransferJob])
          );
          this.dataSource = new SimpleDataSource(response.transferJobs);
        }
        this.showLoadingSpinner = false;
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

  onCheckboxClick(event: MatCheckboxChange) {
      let count = 0;
      let checkedCanBeDeleted = true;
      for (const key in this.checkedCheckboxes) {
        if (this.checkedCheckboxes[key] === true) {
          count++;
          if (this.jobNameToJobMap.get(key).latestOperation &&
            !this.canDeleteStatuses.includes(this.jobNameToJobMap.get(key).latestOperation.status)) {
            checkedCanBeDeleted = false;
          }
        }
      }
      this.numChecked = count;
      this.checkedJobsCanBeDeleted = checkedCanBeDeleted && (this.numChecked > 0);
  }

  private getSelectedJobConfigs(): string[] {
    const selectedJobConfigs = [];
    for (const key in this.checkedCheckboxes) {
      if (this.checkedCheckboxes[key] === true) {
        selectedJobConfigs.push(key);
       }
     }
    return selectedJobConfigs;
  }

  resumeSelectedJobs() {
    const selectedJobConfigs = this.getSelectedJobConfigs();
     this.jobsService.resumeJobs(selectedJobConfigs).subscribe(
       (response: TransferJob[]) => {
          this.updateJobs();
       }, (errorResponse: HttpErrorResponse) => {
            this.updateJobs();
            const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
            const errorMessage = HttpErrorResponseFormatter.getMessage(errorResponse);
            const errorContent: ErrorDialogContent = {
              errorTitle: errorTitle,
              errorMessage: errorMessage
            };
            this.dialog.open(ErrorDialogComponent, {
              data: errorContent
            });
          },
        () => {
          console.log('complete');
        });
  }

  pauseSelectedJobs() {
    const selectedJobConfigs = this.getSelectedJobConfigs();
    const pausedJobs: Observable<TransferJob[]> = this.jobsService.pauseJobs(selectedJobConfigs);
    pausedJobs.subscribe(
       (response: TransferJob[]) => {
          this.updateJobs();
       }, (errorResponse: HttpErrorResponse) => {
            this.updateJobs();
            const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
            const errorMessage = HttpErrorResponseFormatter.getMessage(errorResponse);
            const errorContent: ErrorDialogContent = {
              errorTitle: errorTitle,
              errorMessage: errorMessage
            };
            this.dialog.open(ErrorDialogComponent, {
              data: errorContent
            });
          },
          () => {
            console.log('complete');
          });
    }

    deleteSelectedJobs() {
      const selectedJobConfigs = this.getSelectedJobConfigs();
      this.jobsService.deleteJobs(selectedJobConfigs).subscribe(
         (response: Response[]) => {
            this.updateJobs();
         }, (errorResponse: HttpErrorResponse) => {
              this.updateJobs();
              const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
              const errorMessage = HttpErrorResponseFormatter.getMessage(errorResponse);
              const errorContent: ErrorDialogContent = {
                errorTitle: errorTitle,
                errorMessage: errorMessage
              };
              this.dialog.open(ErrorDialogComponent, {
                data: errorContent
              });
            },
            () => {
              console.log('complete');
            });
      }

}
