import { Component, OnInit } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable } from 'rxjs/Observable';
import { JobRun } from '../../api.resources';
import { JobsService } from '../../jobs.service';
import { JobStatusPipe } from '../job-status.pipe';
import 'rxjs/add/operator/switchMap';

@Component({
  selector: 'app-job-run-details',
  templateUrl: './job-run-details.component.html',
  styleUrls: ['./job-run-details.component.css']
})
export class JobRunDetailsComponent implements OnInit {
  private jobRun: JobRun;
  private jobRunLoading: boolean;
  private errorTitle: string;
  private errorMessage: string;
  private showError: boolean;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private jobService: JobsService
  ) { }

  ngOnInit() {
    this.jobRunLoading = true;
    const configId = this.route.snapshot.paramMap.get('configId');
    const runId = this.route.snapshot.paramMap.get('runId');

    this.jobService.getJobRun(configId, runId)
    .subscribe(
      (data) => {
        this.jobRun = data;
        this.jobRunLoading = false;
      },
      (err: HttpErrorResponse) => {
        if (err.error instanceof Error) {
          // A client-side or network error occurred
          console.log('An error occurred: ', err.error.message);
          this.errorTitle = 'Client-side or Network Error';
          this.errorMessage = err.error.message;
        } else {
          // Back-end returned an unsuccessful response code
          this.errorTitle = err.error.error;
          this.errorMessage = err.error.message;
          if (!this.errorMessage) {
            this.errorTitle = err.statusText;
            this.errorMessage = err.message;
          }
          this.showError = true;
          this.jobRunLoading = false;
         }
      }
    );
  }
}
