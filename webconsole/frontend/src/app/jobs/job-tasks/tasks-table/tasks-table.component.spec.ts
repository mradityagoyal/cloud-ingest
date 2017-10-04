import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { AngularMaterialImporterModule } from '../../../angular-material-importer/angular-material-importer.module';
import { JobsService } from '../../jobs.service';
import { TasksTableComponent } from './tasks-table.component';
import { Task, TASK_STATUS, TASK_TYPE } from '../../jobs.resources';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/of';
import 'rxjs/add/observable/never';

class JobsServiceStub {
  public getTasksOfStatus = jasmine.createSpy('getTasksOfStatus');
}

let jobsServiceStub: JobsServiceStub;
const FAKE_TASKS1: Task[] = [
  {
    JobConfigId: 'fakeJobConfigId1',
    JobRunId: 'fakeJobRunId1',
    TaskId: 'fakeTaskId1',
    TaskSpec: '{ fakeField: "fakeTaskSpec1" }',
    TaskType: 1,
    Status: TASK_STATUS.SUCCESS,
    // September 7, 2016 12:00:00 PM
    CreationTime: 1473274800000000000,
    WorkerId: 'fakeWorkerId1',
    // October 7, 2017, 12:00:00 PM
    LastModificationTime: 1507402800000000000,
    FailureMessage: 'Fake failure message 1'
  },
  {
    JobConfigId: 'fakeJobConfigId1',
    JobRunId: 'fakeJobRunId1',
    TaskId: 'fakeTaskId2',
    TaskSpec: '{ fakeField: "fakeTaskSpec2" }',
    TaskType: 2,
    Status: TASK_STATUS.SUCCESS,
    // October 7, 2014 12:00:00 PM
    CreationTime: 1412708400000000000,
    WorkerId: 'fakeWorkerId2',
    // October 7, 2015 12:00:00 PM
    LastModificationTime: 1444244400000000000,
    FailureMessage: 'Fake failure message 2'
  }
];

const FAKE_HTTP_ERROR = {error : {error: 'FakeError', message: 'Fake Error Message.'}};

describe('TasksTableComponent', () => {
  let component: TasksTableComponent;
  let fixture: ComponentFixture<TasksTableComponent>;

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    jobsServiceStub.getTasksOfStatus.and.returnValue(Observable.of(FAKE_TASKS1));
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
    expect(parentElement.textContent).toContain(TASK_TYPE[1]);
    expect(parentElement.textContent).toContain('Sep 7, 2016');
    expect(parentElement.textContent).toContain('Oct 7, 2017');

    expect(parentElement.textContent).toContain('fakeTaskId2');
    expect(parentElement.textContent).toContain(TASK_TYPE[2]);
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
