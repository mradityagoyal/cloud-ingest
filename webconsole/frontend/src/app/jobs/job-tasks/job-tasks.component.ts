import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { TaskFailureType, TaskType } from '../../proto/tasks.js';
import { Task, TASK_STATUS, FAILURE_TYPE_TO_STRING_MAP } from '../jobs.resources';

@Component({
  selector: 'app-job-tasks',
  templateUrl: './job-tasks.component.html',
  styleUrls: ['./job-tasks.component.css']
})
export class JobTasksComponent implements OnInit {
  TASK_STATUS = TASK_STATUS;
  FAILURE_TYPE_TO_STRING_MAP = FAILURE_TYPE_TO_STRING_MAP;
  TaskFailureType = TaskFailureType;
  taskFailureTypes = Object.keys(TaskFailureType.Type);
  jobConfigId: string;
  jobRunId: string;

  showFailuresLoading = true;
  showNoFailures = false;
  // How many failure types have loaded successfully.
  numFailureTypesLoaded = 0;
  // How many failure types are empty / no failures.
  numFailureTypesEmpty = 0;

  constructor(private route: ActivatedRoute) {
    this.jobConfigId = route.snapshot.paramMap.get('configId');
    this.jobRunId = this.route.snapshot.paramMap.get('runId');
  }

  onFailureTypeLoadFinished(failureType: TaskFailureType.Type) {
    this.numFailureTypesLoaded++;
    if (this.numFailureTypesLoaded >= Object.keys(TaskFailureType.Type).length) {
      this.showFailuresLoading = false;
    }
  }

  onNoTasksForFailureType(failureType: TaskFailureType.Type) {
    this.numFailureTypesEmpty++;
    if (this.numFailureTypesEmpty >= Object.keys(TaskFailureType.Type).length) {
      this.showNoFailures = true;
    }
  }

  ngOnInit() {}

}
