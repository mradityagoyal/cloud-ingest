import { TaskFailureType, TaskStatus } from '../../../proto/tasks';
import { ErrorDialogComponent } from '../../../util/error-dialog/error-dialog.component';
import 'rxjs/add/observable/never';
import 'rxjs/add/observable/of';

import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialog } from '@angular/material';
import { Observable } from 'rxjs/Observable';

import { AngularMaterialImporterModule } from '../../../angular-material-importer/angular-material-importer.module';
import { TASK_TYPE_TO_STRING_MAP } from '../../jobs.resources';
import { JobsService } from '../../jobs.service';
import { EMPTY_TASK_ARRAY, FAKE_HTTP_ERROR, FAKE_TASKS, FAKE_TASKS2 } from '../../jobs.test-util';
import { TasksTableComponent } from './tasks-table.component';

class JobsServiceStub {
  public getTasksOfStatus = jasmine.createSpy('getTasksOfStatus');
  public getTasksOfFailureType = jasmine.createSpy('getTasksOfFailureType');
}

class MatDialogStub {
  public open = jasmine.createSpy('open');
}

let jobsServiceStub: JobsServiceStub;
let matDialogStub: MatDialogStub;

describe('TasksTableComponent', () => {
  let component: TasksTableComponent;
  let fixture: ComponentFixture<TasksTableComponent>;

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    jobsServiceStub.getTasksOfStatus.and.returnValue(Observable.of(FAKE_TASKS));
    jobsServiceStub.getTasksOfFailureType.and.returnValue(Observable.of(FAKE_TASKS));
    matDialogStub = new MatDialogStub();
    TestBed.configureTestingModule({
      declarations: [ TasksTableComponent ],
      imports: [
        AngularMaterialImporterModule
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub},
        {provide: MatDialog, useValue: matDialogStub}
      ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.status = TaskStatus.Type.SUCCESS;
    fixture.detectChanges();
  });

  it('should be created', () => {
    expect(component).toBeTruthy();
  });

  it('should show the task information', () => {
    const parentElement = fixture.debugElement.nativeElement;
    expect(parentElement.textContent).toContain('fakeTaskId1');
    expect(parentElement.textContent).toContain(TASK_TYPE_TO_STRING_MAP[1]);
    expect(parentElement.textContent).toContain('Sep 7, 2016');
    expect(parentElement.textContent).toContain('Oct 7, 2017');

    expect(parentElement.textContent).toContain('fakeTaskId2');
    expect(parentElement.textContent).toContain(TASK_TYPE_TO_STRING_MAP[2]);
    expect(parentElement.textContent).toContain('Oct 7, 2014');
    expect(parentElement.textContent).toContain('Oct 7, 2015');
  });

  it('should show the failure message', () => {
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.failureType = TaskFailureType.Type.UNKNOWN;
    component.isFailureTable = true;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const parentElement = fixture.debugElement.nativeElement;
      expect(parentElement.textContent).toContain('Fake failure message 1');
      expect(parentElement.textContent).toContain('Fake failure message 2');
    });
  });

  it('should not show the failure message', () => {
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.status = 0;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const parentElement = fixture.debugElement.nativeElement;
      expect(parentElement.textContent).not.toContain('Fake failure message 1');
      expect(parentElement.textContent).not.toContain('Fake failure message 2');
    });
  });

  it('should show a loading spinner', () => {
    jobsServiceStub.getTasksOfStatus.and.returnValue(Observable.never());
    // Start over again.
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.status = 0;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const element = fixture.debugElement.nativeElement;
      expect(element.querySelector('mat-spinner')).not.toBeNull();
    });
  });

  it('should show an error message and title', () => {
    jobsServiceStub.getTasksOfStatus.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    // Start over again.
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.status = 0;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const parentElement = fixture.debugElement.nativeElement;
      const errorElement = parentElement.querySelector('.ingest-error-message');
      expect(errorElement).not.toBeNull();
      expect(errorElement.textContent).toContain('FakeError');
      expect(errorElement.textContent).toContain('Fake Error Message.');
    });
  });

  it('should show a load more button', async(() => {
    fixture.whenStable().then(() => {
      const parentElement = fixture.debugElement.nativeElement;
      const loadMoreDiv = parentElement.querySelector('.ingest-load-more-button');
      expect(loadMoreDiv).not.toBeNull();
    });
  }));

  it('should load more tasks when load more button is clicked', async(() => {
    jobsServiceStub.getTasksOfStatus.and.returnValues(Observable.of(FAKE_TASKS), Observable.of(FAKE_TASKS2));
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.status = 0;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const element = fixture.debugElement.nativeElement;
      expect(element.textContent).not.toContain('fakeTaskId3');
      const loadMoreButton = element.querySelector('.ingest-load-more-button');
      loadMoreButton.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        expect(element.textContent).toContain('fakeTaskId3');
      });
    });
  }));

  it('should show a spinner while more tasks are loaded', async(() => {
    jobsServiceStub.getTasksOfStatus.and.returnValues(Observable.of(FAKE_TASKS), Observable.never());
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.status = 0;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const element = fixture.debugElement.nativeElement;
      expect(element.querySelector('mat-spinner')).toBeNull();
      const loadMoreButton = element.querySelector('.ingest-load-more-button');
      loadMoreButton.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        expect(element.querySelector('mat-spinner')).not.toBeNull();
      });
    });
  }));

  it('should open an error dialog when there is an error', async(() => {
    jobsServiceStub.getTasksOfStatus.and.returnValues(Observable.of(FAKE_TASKS), Observable.throw(FAKE_HTTP_ERROR));
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.status = 0;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const element = fixture.debugElement.nativeElement;
      const loadMoreButton = element.querySelector('.ingest-load-more-button');
      loadMoreButton.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        expect(matDialogStub.open).toHaveBeenCalled();
        expect(matDialogStub.open.calls.first().args[0]).toBe(ErrorDialogComponent);
      });
    });
  }));

  it('displays a no more tasks message if there are no more tasks to load', async(() => {
    jobsServiceStub.getTasksOfStatus.and.returnValues(Observable.of(FAKE_TASKS), Observable.of(EMPTY_TASK_ARRAY));
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.status = 0;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const element = fixture.debugElement.nativeElement;
      const loadMoreButton = element.querySelector('.ingest-load-more-button');
      loadMoreButton.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        expect(element.querySelector('.ingest-no-more-tasks')).not.toBeNull();
      });
    });
  }));
});
