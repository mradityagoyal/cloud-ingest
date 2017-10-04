import { Component, Input, OnInit } from '@angular/core';
import { DataSource } from '@angular/cdk/collections';
import { JobsService } from '../../jobs.service';
import { Task, TASK_TYPE } from '../../jobs.resources';
import { Observable } from 'rxjs/Observable';
import { HttpErrorResponse } from '@angular/common/http';
import { HttpErrorResponseFormatter } from '../../../util/error.resources';
import 'rxjs/add/observable/of';

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

  TASK_TYPE = TASK_TYPE;

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

/**
 * A simple data source that will just return the tasks passed into the constructor.
 *
 * TODO(b/67581174): The webconsole should paginate the tasks and this data source should return
 * the next page.
 */
class SimpleDataSource extends DataSource<Task> {
  tasks: Task[];

  constructor(tasks: Task[]) {
    super();
    this.tasks = tasks;
  }

  connect(): Observable<Task[]> {
    return Observable.of(this.tasks);
  }

  disconnect() {}
}
