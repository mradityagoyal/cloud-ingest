import { Component, Inject } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/operator/finally';
import { JobConfig } from '../jobs.resources';
import { JobsService } from '../jobs.service';
import { JobConfigFormModel } from './job-config-add-dialog.resources';
import { MatDialogRef } from '@angular/material';

@Component({
  selector: 'app-job-config-add-dialog',
  templateUrl: './job-config-add-dialog.component.html',
  styleUrls: ['./job-config-add-dialog.component.css']
})
/**
 * Contains the Job Config Add Dialog component. It opens as a dialog in the job configurations
 * page.
 */
export class JobConfigAddDialogComponent {
  submittingForm = false;
  bigQueryTransferChecked = false;
  formError = false; // The user submitted bad data
  appError = false; // The application is broken (bug, back-end error, etc.)
  model = new JobConfigFormModel(
    /** jobConfigId **/ '',
    /** gcsBucket **/ '',
    /** fileSystemDirectory **/ '',
    /** bigqueryDataset **/ '',
    /** bigqueryTable **/ '');

  constructor(private readonly jobsService: JobsService,
              private readonly dialogRef: MatDialogRef<JobConfigAddDialogComponent>) { }

  onSubmit() {
    this.submittingForm = true;

    // Reset previously set error flags
    this.formError = false;
    this.appError = false;

    this.jobsService.postJobConfig(this.model.toApiJobConfig()).finally(() => {
        this.submittingForm = false;
      }).subscribe(
        (response) => {
          this.dialogRef.close(/**configSuccessfullyPosted**/ true);
        },
        (err: HttpErrorResponse) => {
          if (err.error instanceof Error) {
            // Client-side or network error occurred
            console.log('An error occurred: ', err.error.message);
            this.appError = true;
          } else if (err.status === 400) {
            this.formError = true;
          } else {
            // Back-end returned an unsuccessful response code.
            console.log(`Back-end returned code ${err.status}, ` +
                         `body was: ${JSON.stringify(err.error)}`);
            this.appError = true;
          }
        }
      );
  }
}
