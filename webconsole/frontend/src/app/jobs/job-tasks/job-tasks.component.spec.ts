import 'rxjs/add/observable/never';
import 'rxjs/add/observable/of';

import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ActivatedRoute } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { Task } from 'protractor/built/taskScheduler';
import { Observable } from 'rxjs/Observable';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { TaskFailureType, TaskStatus } from '../../proto/tasks.js';
import { JobsService } from '../jobs.service';
import { FAKE_TASKS } from '../jobs.test-util';
import { FailuresTableComponent } from './failures-table/failures-table.component';
import { JobTasksComponent } from './job-tasks.component';
import { TasksTableComponent } from './tasks-table/tasks-table.component';

class ActivatedRouteStub {
  snapshot = {
    paramMap: {
      get: jasmine.createSpy('get')
    }
  };
}

class JobsServiceStub {
  getTasksOfStatus = jasmine.createSpy('getTasksOfStatus');
  getTasksOfFailureType = jasmine.createSpy('getTasksOfFailureType');
}

const FAKE_PARAMS = {
  configId : 'fakeJobConfigId',
  runId: 'fakeRunId'
};

let activatedRouteStub: ActivatedRouteStub;
let jobsServiceStub: JobsServiceStub;

describe('JobTasksComponent', () => {
  let component: JobTasksComponent;
  let fixture: ComponentFixture<JobTasksComponent>;

  beforeEach(async(() => {
    activatedRouteStub = new ActivatedRouteStub();
    jobsServiceStub = new JobsServiceStub();

    activatedRouteStub.snapshot.paramMap.get.and.returnValue(FAKE_PARAMS);
    jobsServiceStub.getTasksOfStatus.and.returnValue(Observable.of(FAKE_TASKS));
    jobsServiceStub.getTasksOfFailureType.and.returnValue(Observable.of(FAKE_TASKS));
    TestBed.configureTestingModule({
      declarations: [ JobTasksComponent, TasksTableComponent, FailuresTableComponent ],
      providers: [
        {provide: ActivatedRoute, useValue: activatedRouteStub},
        {provide: JobsService, useValue: jobsServiceStub}
      ],
      imports: [
        BrowserAnimationsModule,
        AngularMaterialImporterModule,
        RouterTestingModule
      ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(JobTasksComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should be created', () => {
    expect(component).toBeTruthy();
  });

  it('should contain back navigation link', () => {
    const compiled = fixture.debugElement.nativeElement;
    const backLink = compiled.querySelector('.ingest-job-tasks-back-link');
    expect(backLink).toBeTruthy();
  });

  it('should contain one tab per task status', () => {
    const compiled = fixture.debugElement.nativeElement;
    const mdTabs = compiled.querySelectorAll('mat-tab-body');
    expect(mdTabs.length).toEqual(Object.getOwnPropertyNames(TaskStatus.Type).length);
  });

  it('should contain the tasks table component', () => {
    const compiled = fixture.debugElement.nativeElement;
    const taskTabs = compiled.querySelectorAll('.mat-tab-label');
    taskTabs[1].click(); // Change to the "succeeded tasks" tab
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const taskTables = compiled.querySelector('app-tasks-table');
      expect(taskTables).toBeTruthy();
    });
  });

  it('should contain one failures table component per task failure type', () => {
    const compiled = fixture.debugElement.nativeElement;
    const failuresTables = compiled.querySelectorAll('app-failures-table');
    expect(failuresTables.length).toEqual(Object.keys(TaskFailureType.Type).length);
  });

  it('should show a loading spinner when the failure types are loading', () => {
    jobsServiceStub.getTasksOfFailureType.and.returnValue(Observable.never());
    fixture = TestBed.createComponent(JobTasksComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    const loadingSpinner = compiled.querySelector('mat-spinner');
    expect(loadingSpinner).toBeTruthy();
  });

  it('should show a no failed tasks message if there are no tasks', () => {
    const emptyTasks: Task[] = [];
    jobsServiceStub.getTasksOfFailureType.and.returnValue(Observable.of(emptyTasks));
    fixture = TestBed.createComponent(JobTasksComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const compiled = fixture.debugElement.nativeElement;
      const failuresMessage = compiled.querySelector('.ingest-no-failures');
      expect(failuresMessage).toBeTruthy();
    });
  });

});
