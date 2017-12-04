import 'rxjs/add/observable/of';

import { HttpErrorResponse } from '@angular/common/http';
import {
  AfterViewInit,
  Component,
  ElementRef,
  EventEmitter,
  Input,
  OnInit,
  Output,
  QueryList,
  Renderer2,
  ViewChild,
  ViewChildren,
} from '@angular/core';
import { MatDialog, MatTable } from '@angular/material';
import { Observable } from 'rxjs/Rx';

import { ErrorDialogComponent } from '../../../util/error-dialog/error-dialog.component';
import { ErrorDialogContent } from '../../../util/error-dialog/error-dialog.resources';
import { HttpErrorResponseFormatter } from '../../../util/error.resources';
import { DEFAULT_BACKEND_PAGESIZE, SimpleDataSource, Task, TASK_TYPE_TO_STRING_MAP } from '../../jobs.resources';
import { JobsService } from '../../jobs.service';

@Component({
  selector: 'app-tasks-table',
  templateUrl: './tasks-table.component.html',
  styleUrls: ['./tasks-table.component.css']
})
/**
 * Displays a table with the given task status.
 */
export class TasksTableComponent implements OnInit, AfterViewInit {
  @Input() public status: number;
  @Input() public jobRunId: string;
  @Input() public jobConfigId: string;

  /**
   * Whether this table is a failure table. This affects how the component gets the tasks to
   * display.
   */
  @Input() public isFailureTable: boolean;
  @Input() public failureType: number;

  /**
   * This custom event emits when the load is finished. It emits the number of tasks that it gets
   * on the initial load.
   */
  @Output() onLoadFinished = new EventEmitter<number>(true);

  TASK_TYPE_TO_STRING_MAP = TASK_TYPE_TO_STRING_MAP;

  tasks: Task[];
  showTasksLoading = true;

  showError = false;
  errorTitle: string;
  errorMessage: string;

  dataSource: SimpleDataSource<Task>;
  displayedColumns = ['taskType', 'creationTime', 'lastModificationTime', 'taskId'];

  // Whether the app should show that more tasks are loading after clicking on load more.
  showMoreTasksLoading = false;

  // Whether there are more tasks to load.
  noMoreTasks = false;

  // Should contain only one element because there is only one table.
  @ViewChildren(MatTable, { read: ElementRef }) tasksTableList: QueryList<ElementRef>;

  // The "load more" element that is placed at the bottom of the table.
  @ViewChild('loadMore') loadMore: ElementRef;

  noTasksToShow = false;
  noMoreTasksToLoad = false;

  constructor(private readonly jobsService: JobsService,
              private renderer: Renderer2,
              private dialog: MatDialog) { }

  ngOnInit() {
    if (this.jobConfigId == null || this.jobRunId == null) {
      throw new Error('A job config id and job run id must be specified.');
    }
    if (this.isFailureTable === true) {
      if (this.failureType == null || this.status != null) {
        throw new Error('A failure table should contain a failure type input and no status'
          + 'input.');
      }
      this.displayedColumns.push('failureMessage');
    } else {
      if (this.status == null) {
        throw new Error('Must specify a task status.');
      }
    }
    this.loadInitialTasks();
  }

  private loadInitialTasks() {
    this.getInitialLoadTasksObservable()
    .subscribe(
      (response: Task[]) => {
        this.tasks = response;
        this.showTasksLoading = false;
        this.onLoadFinished.emit(response.length);
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

  ngAfterViewInit() {
    // Listen to when the tasks table loads.
    this.tasksTableList.changes.subscribe(() => {
      if (this.tasksTableList.first) {
        this.addLoadMoreButtonToTable(this.tasksTableList.first);
      }
    });
  }

  private getInitialLoadTasksObservable(): Observable<Task[]> {
    if (this.isFailureTable === true) {
      return this.jobsService.getTasksOfFailureType(this.jobConfigId, this.failureType);
    } else {
      return this.jobsService.getTasksOfStatus(this.jobConfigId, this.status);
    }
  }

  private getLoadMoreTasksObservable(): Observable<Task[]> {
    const lastTask = this.tasks[this.tasks.length - 1];
    if (this.isFailureTable === true) {
      return this.jobsService.getTasksOfFailureType(
        this.jobConfigId, this.failureType, lastTask.LastModificationTime);
    } else {
      return this.jobsService.getTasksOfStatus(
        this.jobConfigId, this.status, lastTask.LastModificationTime);
    }
  }


  /**
   * Adds the "load more" button to the bottom of the table if there are 25 tasks or more.
   */
  private addLoadMoreButtonToTable(tasksTable: ElementRef) {
    // Only load if the maximum page size was reached.
    if (this.tasks.length >= DEFAULT_BACKEND_PAGESIZE) {
      this.renderer.appendChild(tasksTable.nativeElement, this.loadMore.nativeElement);
    }
  }

  private loadMoreClick() {
    this.showMoreTasksLoading = true;
    const lastTask = this.tasks[this.tasks.length - 1];
    this.getLoadMoreTasksObservable()
    .subscribe(
    (response: Task[]) => {
      if (response.length === 0) {
        this.noMoreTasksToLoad = true;
      } else {
        this.tasks = this.tasks.concat(response);
        this.dataSource = new SimpleDataSource(this.tasks);
      }
      this.showMoreTasksLoading = false;
    },
    (errorResponse: HttpErrorResponse) => {
      this.showMoreTasksLoading = false;
      const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
      const errorMessage = HttpErrorResponseFormatter.getMessage(errorResponse);
      const errorContent: ErrorDialogContent = {
        errorTitle: errorTitle,
        errorMessage: errorMessage
      };
      this.dialog.open(ErrorDialogComponent, {
        data: errorContent
      });
    });
  }


}


