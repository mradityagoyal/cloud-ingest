import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { JobTasksComponent } from './job-tasks.component';
import { JobsService } from '../jobs.service';
import { Task, TASK_STATUS } from '../jobs.resources';
import { TasksTableComponent } from './tasks-table/tasks-table.component';
import { ActivatedRoute, ActivatedRouteSnapshot } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { Observable } from 'rxjs/Observable';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import 'rxjs/add/observable/of';

const FAKE_TASKS: Task[] = [
  {
    JobConfigId: 'fakeJobConfigId1',
    JobRunId: 'fakeJobRunId1',
    TaskId: 'fakeTaskId1',
    TaskSpec: '{ fakeField: "fakeTaskSpec1" }',
    TaskType: 1,
    Status: TASK_STATUS.SUCCESS,
    CreationTime: 1,
    WorkerId: 'fakeWorkerId1',
    LastModificationTime: 1,
    FailureMessage: 'Fake failure message 1'
  },
  {
    JobConfigId: 'fakeJobConfigId1',
    JobRunId: 'fakeJobRunId1',
    TaskId: 'fakeTaskId3',
    TaskSpec: '{ fakeField: "fakeTaskSpec2" }',
    TaskType: 2,
    Status: TASK_STATUS.SUCCESS,
    CreationTime: 2,
    WorkerId: 'fakeWorkerId2',
    LastModificationTime: 2,
    FailureMessage: 'Fake failure message 2'
  }
];

class ActivatedRouteStub {
  snapshot = {
    paramMap: {
      get: jasmine.createSpy('get')
    }
  };
}

class JobsServiceStub {
  getTasksOfStatus = jasmine.createSpy('getTasksOfStatus');
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
    TestBed.configureTestingModule({
      declarations: [ JobTasksComponent, TasksTableComponent ],
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
    expect(mdTabs.length).toEqual(Object.getOwnPropertyNames(TASK_STATUS).length);
  });

  it('should contain the tasks table component', () => {
    const compiled = fixture.debugElement.nativeElement;
    const taskTables = compiled.querySelector('app-tasks-table');
    expect(taskTables).toBeTruthy();
  });

});
