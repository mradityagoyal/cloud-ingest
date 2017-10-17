import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core';
import { JobsService } from '../../jobs.service';
import { TaskFailureType, TaskType } from '../../../proto/tasks.js';
import { Task, TASK_TYPE_TO_STRING_MAP } from '../../jobs.resources';
import { DataSource } from '@angular/cdk/collections';
import { HttpErrorResponse } from '@angular/common/http';
import { SimpleDataSource } from '../job-tasks.resources';
import { HttpErrorResponseFormatter } from '../../../util/error.resources';

@Component({
  selector: 'app-failures-table',
  templateUrl: './failures-table.component.html'
})
export class FailuresTableComponent implements OnInit {
  @Input() public failureType: TaskFailureType.Type;
  @Input() public jobRunId: string;
  @Input() public jobConfigId: string;
  @Input() public failureTypeName: string;
  @Output() onLoadFinished = new EventEmitter<TaskFailureType.Type>(true);
  @Output() onNoTasks = new EventEmitter<TaskFailureType.Type>(true);

  TaskType = TaskType;
  TASK_TYPE_TO_STRING_MAP = TASK_TYPE_TO_STRING_MAP;

  tasks: Task[];

  showError = false;
  errorTitle: string;
  errorMessage: string;

  dataSource: SimpleDataSource;
  displayedColumns = ['failureMessage', 'taskId', 'taskType', 'creationTime', 'lastModificationTime'];

  constructor(private readonly jobsService: JobsService) { }

  ngOnInit() {
    this.jobsService.getTasksOfFailureType(this.jobConfigId, this.jobRunId, this.failureType)
    .subscribe(
      (response: Task[]) => {
        this.tasks = response;
        this.dataSource = new SimpleDataSource(this.tasks);
        this.onLoadFinished.emit(this.failureType);
        if (this.tasks.length === 0) {
          this.onNoTasks.emit(this.failureType);
        }
      }, (error: HttpErrorResponse) => {
        this.errorTitle = HttpErrorResponseFormatter.getTitle(error);
        this.errorMessage = HttpErrorResponseFormatter.getMessage(error);
        this.showError = true;
      }
    );
  }

}

