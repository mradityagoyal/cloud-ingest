import 'rxjs/add/observable/of';

import { HttpErrorResponse } from '@angular/common/http';
import { Component, Input, OnInit } from '@angular/core';

import { HttpErrorResponseFormatter } from '../../../util/error.resources';
import { Task, TASK_TYPE_TO_STRING_MAP } from '../../jobs.resources';
import { JobsService } from '../../jobs.service';
import { SimpleDataSource } from '../job-tasks.resources';

@Component({
  selector: 'app-tasks-table',
  templateUrl: './tasks-table.component.html',
  styleUrls: ['./tasks-table.component.css']
})
/**
 * Displays a table with the given task status.
 */
export class TasksTableComponent implements OnInit {
  @Input() public status: number;
  @Input() public jobRunId: string;
  @Input() public jobConfigId: string;
  @Input() public showFailureMessage: boolean;

  TASK_TYPE_TO_STRING_MAP = TASK_TYPE_TO_STRING_MAP;

  tasks: Task[];
  showTasksLoading = true;

  showError = false;
  errorTitle: string;
  errorMessage: string;

  dataSource: SimpleDataSource;
  displayedColumns = ['taskType', 'creationTime', 'lastModificationTime', 'taskId'];

  noTasksToShow = false;

  constructor(private readonly jobsService: JobsService) { }

  ngOnInit() {
    if (this.showFailureMessage === true) {
      this.displayedColumns.push('failureMessage');
    }
    this.jobsService.getTasksOfStatus(this.jobConfigId, this.jobRunId, this.status)
    .subscribe(
      (response: Task[]) => {
        this.tasks = response;
        this.showTasksLoading = false;
        if (response.length === 0) {
          this.noTasksToShow = true;
        } else {
          this.noTasksToShow = false;
        }
        this.dataSource = new SimpleDataSource(this.tasks);
      }, (error: HttpErrorResponse) => {
        this.errorTitle = HttpErrorResponseFormatter.getTitle(error);
        this.errorMessage = HttpErrorResponseFormatter.getMessage(error);
        this.showTasksLoading = false;
        this.showError = true;
      }
    );
  }
}


