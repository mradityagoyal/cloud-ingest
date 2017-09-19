import { Component, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { DataSource } from '@angular/cdk/collections';
import { Router, NavigationExtras } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable } from 'rxjs/Observable';

import { JobRun } from '../../api.resources';
import { JobsService } from '../../jobs.service';
import { JobStatusPipe } from '../job-status.pipe';

import 'rxjs/add/observable/of';

@Component({
  selector: 'app-job-configs',
  templateUrl: './job-run-list.component.html',
  styleUrls: ['./job-run-list.component.css']
})

export class JobRunListComponent implements OnInit {
  displayedColumns = ['runId', 'configId', 'creationTime', 'status'];
  showLoadingSpinner = false;
  showError = false;
  errorTitle: string;
  errorMessage: string;
  jobRunsDataSource: JobRunsDataSource;

  constructor(
    private readonly jobsService: JobsService,
    private readonly router: Router
  ) { }

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

  handleRowClick(row: JobRun) {
    const navExtras: NavigationExtras = {
      queryParamsHandling: 'merge'
    };
    this.router.navigate(
      ['/jobruns', row.JobConfigId, row.JobRunId], navExtras);
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
