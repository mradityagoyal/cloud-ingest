import { Component, EventEmitter, Input, Output } from '@angular/core';

import { TaskFailureType } from '../../../proto/tasks.js';
import { DEFAULT_BACKEND_PAGESIZE } from '../../jobs.resources';
import { JobsService } from '../../jobs.service';

@Component({
  selector: 'app-failures-table',
  templateUrl: './failures-table.component.html'
})
export class FailuresTableComponent {
  @Input() public failureType: TaskFailureType.Type;
  @Input() public jobRunId: string;
  @Input() public jobConfigId: string;
  @Input() public failureTypeName: string;

  // Emits when the child tasks table component finished loading the tasks.
  @Output() onLoadFinished = new EventEmitter<TaskFailureType.Type>(true);
  // Emits if the child tasks table comonent shows no tasks.
  @Output() onNoTasks = new EventEmitter<TaskFailureType.Type>(true);

  /**
   * Whether this particular component has tasks or not. Controls whether content should be shown
   * or not.
   */
  hasTasks = false;

  // The number of failed tasks.
  numFailures = 0;

  // Re-define this variable here so that it can be used in the template.
  DEFAULT_BACKEND_PAGESIZE = DEFAULT_BACKEND_PAGESIZE;

  constructor(private readonly jobsService: JobsService) { }

  /**
   * Handles the onLoadFinished custom event triggered by the tasks table.
   *
   * @param numTasksLoaded The number of tasks that the tasks table loaded.
   */
  private handleOnLoadFinished(numTasksLoaded: number) {
    this.numFailures = numTasksLoaded;
    if (numTasksLoaded === 0) {
      this.onNoTasks.emit(this.failureType);
    } else {
      this.hasTasks = true;
    }
    this.onLoadFinished.emit(this.failureType);
  }

}

