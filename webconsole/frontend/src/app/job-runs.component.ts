import { Component, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { JobRun } from './api.resources';
import { JobsService } from './jobs.service';
import { Observable } from 'rxjs/Observable';
import { DataSource } from '@angular/cdk/collections';
import { HttpErrorResponse } from '@angular/common/http';
import 'rxjs/add/observable/of';

@Component({
  selector: 'app-job-configs',
  templateUrl: './job-runs.component.html',
  styleUrls: ['./job-runs.component.css']
})

export class JobRunsComponent implements OnInit {
  displayedColumns = ['runId', 'configId', 'creationTime', 'status'];
  showLoadingSpinner = false;
  showError = false;
  errorTitle: string;
  errorMessage: string;
  jobRunsDataSource: JobRunsDataSource;

  constructor(private readonly jobsService: JobsService) { }

  ngOnInit() {
    this.showLoadingSpinner = true;
    this.jobsService.getJobRuns().subscribe(
      (response: JobRun[]) => {
        this.jobRunsDataSource = new JobRunsDataSource(response);
        this.showLoadingSpinner = false;
      },
      (error: HttpErrorResponse) => {
        this.errorTitle = error.error;
        this.errorMessage = error.message;
        this.showError = true;
        this.showLoadingSpinner = false;
      });
  }
}

class JobRunsDataSource extends DataSource<JobRun> {
  constructor(private initialValue: JobRun[]) {
    super();
  }

  /**
   * TODO(b/65321156): The job runs component should paginate job runs. The
   * connect() method should return job runs according to the page the user
   * is in.
   */
  connect(): Observable<JobRun[]> {
    return Observable.of(this.initialValue);
  }

  disconnect() {}
}
