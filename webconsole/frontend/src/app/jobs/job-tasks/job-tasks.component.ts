import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { Task, TASK_STATUS } from '../jobs.resources';

@Component({
  selector: 'app-job-tasks',
  templateUrl: './job-tasks.component.html',
  styleUrls: ['./job-tasks.component.css']
})
export class JobTasksComponent implements OnInit {
  TASK_STATUS = TASK_STATUS;
  jobConfigId: string;
  jobRunId: string;

  constructor(private route: ActivatedRoute) {
    this.jobConfigId = route.snapshot.paramMap.get('configId');
    this.jobRunId = this.route.snapshot.paramMap.get('runId');
  }

  ngOnInit() {}

}
