import { FAKE_HTTP_ERROR, FAKE_TASKS } from '../../jobs.test-util';
import 'rxjs/add/observable/never';
import 'rxjs/add/observable/of';

import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { Observable } from 'rxjs/Observable';

import { AngularMaterialImporterModule } from '../../../angular-material-importer/angular-material-importer.module';
import { Task, TASK_STATUS, TASK_TYPE_TO_STRING_MAP } from '../../jobs.resources';
import { JobsService } from '../../jobs.service';
import { TasksTableComponent } from './tasks-table.component';

class JobsServiceStub {
  public getTasksOfStatus = jasmine.createSpy('getTasksOfStatus');
}

let jobsServiceStub: JobsServiceStub;

describe('TasksTableComponent', () => {
  let component: TasksTableComponent;
  let fixture: ComponentFixture<TasksTableComponent>;

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    jobsServiceStub.getTasksOfStatus.and.returnValue(Observable.of(FAKE_TASKS));
    TestBed.configureTestingModule({
      declarations: [ TasksTableComponent ],
      imports: [
        AngularMaterialImporterModule
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub}
      ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(TasksTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.status = 0;
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
    component.status = 0;
    component.showFailureMessage = true;
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
    component.showFailureMessage = false;
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
});
