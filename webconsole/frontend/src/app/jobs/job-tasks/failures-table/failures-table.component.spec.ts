import 'rxjs/add/observable/of';

import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { Observable } from 'rxjs/Observable';

import { AngularMaterialImporterModule } from '../../../angular-material-importer/angular-material-importer.module';
import { TaskFailureType } from '../../../proto/tasks';
import { TASK_TYPE_TO_STRING_MAP } from '../../jobs.resources';
import { JobsService } from '../../jobs.service';
import { FAKE_TASKS, JobsServiceStub } from '../../jobs.test-util';
import { TasksTableComponent } from '../tasks-table/tasks-table.component';
import { FailuresTableComponent } from './failures-table.component';

let jobsServiceStub: JobsServiceStub;

describe('FailuresTableComponent', () => {
  let component: FailuresTableComponent;
  let fixture: ComponentFixture<FailuresTableComponent>;

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    jobsServiceStub.getTasksOfFailureType.and.returnValue(Observable.of(FAKE_TASKS));
    TestBed.configureTestingModule({
      declarations: [ FailuresTableComponent, TasksTableComponent],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub}
      ],
      imports: [
        BrowserAnimationsModule,
        AngularMaterialImporterModule
      ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(FailuresTableComponent);
    component = fixture.componentInstance;
    component.jobRunId = 'fakeJobRunId';
    component.jobConfigId = 'fakeJobConfigId';
    component.failureType = TaskFailureType.Type.UNKNOWN;
    component.failureTypeName = 'Fake failure type name';
    fixture.detectChanges();
  });

  it('should be created', () => {
    expect(component).toBeTruthy();
  });

  it('should show the failure type name and number of tasks', async(() => {
    const compiled = fixture.debugElement.nativeElement;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      expect(compiled.textContent).toContain('Fake failure type name');
      expect(compiled.textContent).toContain('2 failures');
    });
  }));

  it('should show the task failure information', () => {
    const compiled = fixture.debugElement.nativeElement;
    expect(compiled.textContent).toContain('Fake failure message 1');
    expect(compiled.textContent).toContain('fakeTaskId1');
    expect(compiled.textContent).toContain('Sep 7, 2016');
    expect(compiled.textContent).toContain('Oct 7, 2017');
    expect(compiled.textContent).toContain(TASK_TYPE_TO_STRING_MAP[FAKE_TASKS[0].TaskType]);

    expect(compiled.textContent).toContain('Fake failure message 2');
    expect(compiled.textContent).toContain('fakeTaskId2');
    expect(compiled.textContent).toContain('Oct 7, 2014');
    expect(compiled.textContent).toContain('Oct 7, 2015');
    expect(compiled.textContent).toContain(TASK_TYPE_TO_STRING_MAP[FAKE_TASKS[1].TaskType]);
  });

  it('should include the tasks table component', async(() => {
    const compiled = fixture.debugElement.nativeElement;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const tasksTableComponent = compiled.querySelector('app-tasks-table');
      expect(tasksTableComponent).not.toBeNull();
    });
  }));

});
