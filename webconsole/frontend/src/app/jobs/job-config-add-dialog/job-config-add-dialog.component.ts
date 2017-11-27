import 'rxjs/add/operator/finally';

import { HttpErrorResponse } from '@angular/common/http';
import { Component, Inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';

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
        /** fileSystemDirectory **/ '');
  showError = false;
  errorTitle: string;

  /**
   * Makes the JobConfig add dialog component.
   *
   * @param data An input JobConfigRequest to use as start configuration for the dialog.
   */
  constructor(private readonly jobsService: JobsService,
              private readonly dialogRef: MatDialogRef<JobConfigAddDialogComponent>,
              @Inject(MAT_DIALOG_DATA) public data: JobConfigRequest) {
                this.model = data;
              }

  onSubmit() {
    this.submittingForm = true;

    this.jobsService.postJobConfig(this.model).finally(() => {
        this.submittingForm = false;
      }).subscribe(
        (response) => {
          this.dialogRef.close(/**configSuccessfullyPosted**/ true);
        },
        (errorResponse: HttpErrorResponse) => {
          this.showError = true;
          this.errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
          console.error(`${this.errorTitle} \n` + HttpErrorResponseFormatter.getMessage(errorResponse));
        }
      );
  }
}
