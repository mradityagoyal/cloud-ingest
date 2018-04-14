import 'rxjs/add/operator/finally';

import { HttpErrorResponse } from '@angular/common/http';
import { Component, Inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';

import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { TransferJob, TransferSpec } from '../jobs.resources';
import { JobsService } from '../jobs.service';
import { AuthService } from '../../auth/auth.service';

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
  model = new TransferJob();
  showError = false;
  errorTitle: string;

  /**
   * Makes the TransferJob add dialog component.
   *
   * @param data An input TransferJob to use as start configuration for the dialog.
   */
  constructor(private readonly jobsService: JobsService,
              private readonly authService: AuthService,
              private readonly dialogRef: MatDialogRef<JobConfigAddDialogComponent>,
              @Inject(MAT_DIALOG_DATA) public data: TransferJob) {
                this.model = data;
              }

  private postJob() {
    this.jobsService.postJob(this.model).finally(() => {
      this.submittingForm = false;
    }).subscribe(
      () => {
        this.dialogRef.close(/**configSuccessfullyPosted**/ true);
      },
      (errorResponse: HttpErrorResponse) => {
        this.showError = true;
        this.errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
        console.error(`${this.errorTitle} \n` + HttpErrorResponseFormatter.getMessage(errorResponse));
      }
    );
  }

  private grantBucketPermissions() {
    this.authService.grantBucketPermissionsIfNotExist(
      this.model.transferSpec.gcsDataSink.bucketName).then(
        (response) => {
          this.postJob();
        },
        (error: HttpErrorResponse) => {
          this.showError = true;
          this.errorTitle = 'Could not grant permissions to the On-Premises ' +
                            'Transfer Service service account. Do you have ' +
                            'access to bucket ' +
                            this.model.transferSpec.gcsDataSink.bucketName
                            + '?';
          this.submittingForm = false;
          console.error(error);
        });
  }

  onSubmit() {
    this.submittingForm = true;
    this.authService.grantPubsubTopicPermissionsIfNotExists().then(
      (response) => {
        this.grantBucketPermissions();
      },
      (error: HttpErrorResponse) => {
        this.showError = true;
        this.errorTitle = 'Could not grant pubsub editor permissions on the ' +
                          'current project to the On-Premises Transfer ' +
                          'Service account.';
        this.submittingForm = false;
        console.error(error);
      });
  }
}
