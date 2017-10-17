import { DataSource } from '@angular/cdk/collections';
import { Observable } from 'rxjs/Observable';

import { Task } from '../jobs.resources';

/**
 * A simple data source that will just return the tasks passed into the constructor.
 *
 * TODO(b/67581174): The webconsole should paginate the tasks and this data source should return
 * the next page.
 */
export class SimpleDataSource extends DataSource<Task> {
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
