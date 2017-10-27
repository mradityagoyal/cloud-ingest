import 'rxjs/add/operator/finally';

import { HttpErrorResponse } from '@angular/common/http';
import { Component } from '@angular/core';
import { MatDialogRef, MatSnackBar } from '@angular/material';

import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { JobConfigRequest } from '../jobs.resources';
import { JobsService } from '../jobs.service';

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
  model = new JobConfigRequest(
        /** jobConfigId **/ '',
        /** gcsBucket **/ '',
        /** fileSystemDirectory **/ '',
        /** bigQueryDataset **/'',
        /** bigqueryTable **/ '');

  constructor(private readonly jobsService: JobsService,
              private readonly dialogRef: MatDialogRef<JobConfigAddDialogComponent>,
              private readonly snackBar: MatSnackBar) { }

  onSubmit() {
    this.submittingForm = true;

    this.jobsService.postJobConfig(this.model).finally(() => {
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
