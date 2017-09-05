import { Component, OnInit } from '@angular/core';
import { JobConfig } from './api.resources';
import { JobsService } from './jobs.service';
import { MdDialog } from '@angular/material';
import { JobConfigAddDialogComponent } from './job-config-add-dialog.component';

@Component({
  selector: 'app-job-configs',
  templateUrl: './job-configs.component.html',
  styleUrls: ['./job-configs.component.css']
})

export class JobConfigsComponent implements OnInit {
  jobConfigs: JobConfig[];

  constructor(
      private readonly jobsService: JobsService,
      public dialog: MdDialog
  ) { }

  ngOnInit() {
    this.updateJobConfigs();
  }

  private updateJobConfigs(): void {
    this.jobsService.getJobConfigs().subscribe(data => {
      this.jobConfigs = data;
    });
  }

  private getKeys(jsonObject: Object): String[] {
    return Object.keys(jsonObject);
  }

  private openAddJobConfigDialog(): void {
    const jobConfigDialogReference = this.dialog.open(JobConfigAddDialogComponent, {
      width: '500px'
    });

    jobConfigDialogReference.afterClosed().subscribe(configSuccessfullyPosted => {
      if (configSuccessfullyPosted === true) {
        this.updateJobConfigs();
      }
    });
  }
}
