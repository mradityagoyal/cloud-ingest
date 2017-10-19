import { HttpErrorResponseFormatter } from '../../util/error.resources';
import 'rxjs/add/operator/finally';

import { HttpErrorResponse } from '@angular/common/http';
import { Component } from '@angular/core';
import { MatDialogRef, MatSnackBar } from '@angular/material';

import { JobsService } from '../jobs.service';
import { JobConfigFormModel } from './job-config-add-dialog.resources';

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
              private readonly dialogRef: MatDialogRef<JobConfigAddDialogComponent>,
              private readonly snackBar: MatSnackBar) { }

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
        (errorResponse: HttpErrorResponse) => {
          const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
          console.error(errorTitle + '\n' + HttpErrorResponseFormatter.getMessage(errorResponse));
          this.snackBar.open(`There submitting your job configuration: ${errorTitle}`, 'Dismiss');
        }
      );
  }
}
