import { Component, OnInit } from '@angular/core';
import { JobConfig } from './api.resources';
import { JobsService } from './jobs.service';

@Component({
  selector: 'app-job-configs',
  templateUrl: './job-configs.component.html'
})

export class JobConfigsComponent implements OnInit {
  jobConfigs: JobConfig[];

  constructor(private readonly jobsService: JobsService) { }

  ngOnInit() {
    this.jobsService.getJobConfigs().subscribe(data => {
      this.jobConfigs = data;
    });
  }
}
