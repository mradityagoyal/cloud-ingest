import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/operator/finally';
import { JobConfig } from './api.resources';
import { JobsService } from './jobs.service';


@Component({
  selector: 'app-create-config',
  templateUrl: './create-config.component.html'
})

export class CreateConfigComponent {
  submittingForm = false;
  formError = false; // The user submitted bad data
  appError = false; // The application is broken (bug, back-end error, etc.)
  model: JobConfig = {
        JobConfigId: '',
        JobSpec: '{\'gcs_directory\': \'<gcs directory>\', ' +
        '\'oprem_src_directory\': \'<on-premise source directory>\', ' +
        '\'gcs_bucket\': \'<GCS bucket>\', \'bigquery_table\': ' +
        '\'<BigQuery Table>\', \'bigquery_dataset\': ' +
        '\'<BigQuery Dataset>\'}'
    };

  constructor(private readonly jobsService: JobsService,
              private readonly router: Router) { }

  onSubmit() {
    this.submittingForm = true;

    // Reset previously set error flags
    this.formError = false;
    this.appError = false;

    this.jobsService.postJobConfig(this.model).finally(() => {
        this.submittingForm = false;
      }).subscribe(
        data => this.router.navigate(['jobconfigs'],
                                     { queryParamsHandling: 'merge' }),
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
