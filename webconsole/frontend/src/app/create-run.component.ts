import { Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/operator/finally';
import { JobConfig, JobRunParams } from './api.resources';
import { JobsService } from './jobs.service';

@Component({
  selector: 'app-create-config',
  templateUrl: './create-run.component.html'
})

export class CreateRunComponent implements OnInit {
  jobConfigs: JobConfig[];
  formSubmitting = false;
  formError = false;
  appError = false;
  model: JobRunParams = {JobConfigId: '', JobRunId: ''};

  constructor(private readonly jobsService: JobsService,
              private readonly router: Router) { }

  ngOnInit() {
    this.jobsService.getJobConfigs().subscribe(
      data => this.jobConfigs = data
    );
  }

  onSubmit() {
    this.formSubmitting = true;

    // Reset previously set error flags
    this.formError = false;
    this.appError = false;

    this.jobsService.postJobRun(this.model).finally(() => {
        this.formSubmitting = false;
      }).subscribe(
        data => this.router.navigate(['jobruns'],
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
