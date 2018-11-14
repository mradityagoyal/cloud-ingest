import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit } from '@angular/core';
import { Response } from '@angular/http';
import { MatCheckboxChange, MatDialog } from '@angular/material';
import { Observable, of } from 'rxjs';

import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { ErrorDialogContent } from '../../util/error-dialog/error-dialog.resources';
import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { JobConfigAddDialogComponent } from '../job-config-add-dialog/job-config-add-dialog.component';
import { OPERATION_STATUS_TO_STRING_MAP, SimpleDataSource, TransferJob, TransferJobResponse } from '../jobs.resources';
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
        // Reload the checkboxes
        this.updateNumCheckedCheckboxes();
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

  private updateNumCheckedCheckboxes() {
    let count = 0;
      for (const key in this.checkedCheckboxes) {
        if (this.checkedCheckboxes[key] === true) {
          count++;
        }
      }
      this.numChecked = count;
  }

  onCheckboxClick(event: MatCheckboxChange) {
    this.updateNumCheckedCheckboxes();
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
          console.log('resume jobs completed');
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
            console.log('pause jobs flow completed');
          });
    }

    deleteSelectedJobs() {
      const selectedJobConfigs = this.getSelectedJobConfigs();
      this.jobsService.deleteJobs(selectedJobConfigs).subscribe(
         (response: Response[]) => {
            // If the message is successful, delete the jobs from the internal
            // map as well.
            for (const jobConfig in selectedJobConfigs) {
              if (selectedJobConfigs.hasOwnProperty(jobConfig)) {
                delete this.checkedCheckboxes[selectedJobConfigs[jobConfig]];
              }
            }
            console.log('about to update');
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
              console.log('deletion flow completed');
            });
      }

}
