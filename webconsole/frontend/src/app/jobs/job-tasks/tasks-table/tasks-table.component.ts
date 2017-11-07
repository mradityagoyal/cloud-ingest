import 'rxjs/add/observable/of';

import { HttpErrorResponse } from '@angular/common/http';
import {
  AfterViewInit,
  Component,
  ElementRef,
  Input,
  OnInit,
  QueryList,
  Renderer2,
  ViewChild,
  ViewChildren,
} from '@angular/core';
import { MatDialog, MatTable } from '@angular/material';

import { ErrorDialogComponent } from '../../../util/error-dialog/error-dialog.component';
import { ErrorDialogContent } from '../../../util/error-dialog/error-dialog.resources';
import { HttpErrorResponseFormatter } from '../../../util/error.resources';
import { Task, TASK_TYPE_TO_STRING_MAP, DEFAULT_BACKEND_PAGESIZE } from '../../jobs.resources';
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
export class TasksTableComponent implements OnInit, AfterViewInit {
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

  ngAfterViewInit() {
    // Listen to when the tasks table loads.
    this.tasksTableList.changes.subscribe(() => {
      if (this.tasksTableList.first) {
        this.addLoadMoreButtonToTable(this.tasksTableList.first);
      }
    });
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
    this.jobsService.getTasksOfStatus(
      this.jobConfigId, this.jobRunId, this.status, lastTask.LastModificationTime)
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


