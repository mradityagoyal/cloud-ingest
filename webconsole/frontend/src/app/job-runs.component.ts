import { Component, OnInit } from '@angular/core';
import { JobRun } from './api.resources';
import { JobsService } from './jobs.service';

@Component({
  selector: 'app-job-configs',
  templateUrl: './job-runs.component.html'
})

export class JobRunsComponent implements OnInit {
  jobRuns: JobRun[];

  constructor(private readonly jobsService: JobsService) { }

  ngOnInit() {
    this.jobsService.getJobRuns().subscribe(data => {
      this.jobRuns = data;
    });
  }
}
