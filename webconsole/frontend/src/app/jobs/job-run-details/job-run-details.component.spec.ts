import { async, ComponentFixture, TestBed, fakeAsync, tick, discardPeriodicTasks } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { By } from '@angular/platform-browser';
import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { JobsService } from '../jobs.service';
import { IntervalObservable } from 'rxjs/observable/IntervalObservable';
import { JobRun } from '../jobs.resources';
import { JobStatusPipe } from '../job-status/job-status.pipe';
import { Observable } from 'rxjs/Observable';

import { JobRunDetailsComponent } from './job-run-details.component';

class JobsServiceStub {
  public getJobRun = jasmine.createSpy('getJobRun');
}

const FAKE_JOB_RUNS: JobRun[] = [
  {
    JobConfigId: 'fakeJobConfigId0',
    JobRunId: 'fakeJobRunId0',
    JobCreationTime: '1504833274371000000',
    Status: 0,
    Counters: {
      totalTasks: 0,
      tasksCompleted: 0,
      tasksFailed: 0,

      totalTasksList: 0,
      tasksCompletedList: 0,
      tasksFailedList: 0,

      totalTasksCopy: 0,
      tasksCompletedCopy: 0,
      tasksFailedCopy: 0,

      totalTasksLoad: 0,
      tasksCompletedLoad: 0,
      tasksFailedLoad: 0,

      listFilesFound: 0,
      listBytesFound: 0,
      bytesCopied: 0
    }
  },
  {
    JobConfigId: 'fakeJobConfigId1',
    JobRunId: 'fakeJobRunId1',
    JobCreationTime: '1504833274371000000',
    Status: 1,
    Counters: {
      totalTasks: 1,
      tasksCompleted: 0,
      tasksFailed: 0,

      totalTasksList: 1,
      tasksCompletedList: 0,
      tasksFailedList: 0,

      totalTasksCopy: 0,
      tasksCompletedCopy: 0,
      tasksFailedCopy: 0,

      totalTasksLoad: 0,
      tasksCompletedLoad: 0,
      tasksFailedLoad: 0,

      listFilesFound: 0,
      listBytesFound: 0,
      bytesCopied: 0
    }
  },
  {
    JobConfigId: 'fakeJobConfigId2',
    JobRunId: 'fakeJobRunId2',
    JobCreationTime: '1504833274371000000',
    Status: 2,
    Counters: {
      totalTasks: 5,
      tasksCompleted: 4,
      tasksFailed: 1,

      totalTasksList: 1,
      tasksCompletedList: 1,
      tasksFailedList: 0,

      totalTasksCopy: 4,
      tasksCompletedCopy: 3,
      tasksFailedCopy: 1,

      totalTasksLoad: 0,
      tasksCompletedLoad: 0,
      tasksFailedLoad: 0,

      listFilesFound: 4,
      listBytesFound: 11223344,
      bytesCopied: 11220000
    }
  },
  {
    JobConfigId: 'fakeJobConfigId3',
    JobRunId: 'fakeJobRunId3',
    JobCreationTime: '1504833274371000000',
    Status: 3,
    Counters: {
      totalTasks: 9,
      tasksCompleted: 9,
      tasksFailed: 0,

      totalTasksList: 1,
      tasksCompletedList: 1,
      tasksFailedList: 0,

      totalTasksCopy: 4,
      tasksCompletedCopy: 4,
      tasksFailedCopy: 0,

      totalTasksLoad: 4,
      tasksCompletedLoad: 4,
      tasksFailedLoad: 0,

      listFilesFound: 4,
      listBytesFound: 11223344,
      bytesCopied: 11223344
    }
  }
];

const FAKE_HTTP_ERROR = {
  error: {
    error: 'Forbidden',
    message: 'You are not allowed to access this resource'
  },
  message: 'Fake error message.',
  statusText: 'FORBIDDEN'
};

const BADLY_FORMATED_ERROR = {
  error: 'I am not json like expected',
  message: 'Fake error message.',
  statusText: 'I\'M A TEAPOT'
};

let jobsServiceStub: JobsServiceStub;
let intervalObservableCreateSpy: any;

describe('JobRunDetailsComponent', () => {
  let component: JobRunDetailsComponent;
  let fixture: ComponentFixture<JobRunDetailsComponent>;

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    jobsServiceStub.getJobRun.and.returnValue(Observable.of(FAKE_JOB_RUNS[0]));
    // Disable polling for most tests.
    intervalObservableCreateSpy = spyOn(IntervalObservable, 'create').and.returnValue(Observable.never());
    TestBed.configureTestingModule({
      declarations: [
        JobRunDetailsComponent,
        JobStatusPipe
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub}
      ],
      imports: [
        NoopAnimationsModule,
        AngularMaterialImporterModule,
        RouterTestingModule
      ]
    })
    .compileComponents();
  }));

  it('should be created', () => {
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should show a loading spinner when job details are loading', async(() => {
    jobsServiceStub.getJobRun.and.returnValue(Observable.never());
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('mat-spinner');
      expect(element).not.toBeNull();
    });
  }));

  it('should not show a loading spinner when job details have loaded', async(() => {
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('mat-spinner');
      expect(element).toBeNull();
    });
  }));

  it('should show an error message when there is an error', async(() => {
    jobsServiceStub.getJobRun.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-error-message');
      expect(element).not.toBeNull();
      const titleElement = compiled.querySelector('.ingest-error-title');
      expect(titleElement.innerText).toContain(FAKE_HTTP_ERROR.error.error);
      const msgElement = compiled.querySelector('.ingest-error-details');
      expect(msgElement.innerText).toContain(FAKE_HTTP_ERROR.error.message);
    });
  }));

  it('should show an error message even when the error is badly formatted',
        async(() => {
    jobsServiceStub.getJobRun.and.returnValue(Observable.throw(BADLY_FORMATED_ERROR));
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-error-message');
      expect(element).not.toBeNull();
      const titleElement = compiled.querySelector('.ingest-error-title');
      expect(titleElement.innerText).toContain(BADLY_FORMATED_ERROR.statusText);
      const msgElement = compiled.querySelector('.ingest-error-details');
      expect(msgElement.innerText).toContain(BADLY_FORMATED_ERROR.message);
    });
  }));

  it('should display a dl list', async(() => {
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('dl');
      expect(element).not.toBeNull();
    });
  }));

  it('should display a #job-progress-tabs', async(() => {
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('#job-progress-tabs');
      expect(element).not.toBeNull();
    });
  }));

  it('should display a overall tab when only overall progress is available', async(() => {
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('#job-progress-tabs');
      expect(element).not.toBeNull();
      const tabs = element.querySelectorAll('.mat-tab-label');
      expect(tabs.length).toEqual(1);
      expect(tabs[0].textContent).toEqual('Overall Progress');
    });
  }));

  it('should display a overall and list tabs when progress info is available', async(() => {
    jobsServiceStub.getJobRun.and.returnValue(Observable.of(FAKE_JOB_RUNS[1]));
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('#job-progress-tabs');
      expect(element).not.toBeNull();
      const tabs = element.querySelectorAll('.mat-tab-label');
      expect(tabs.length).toEqual(2);
      expect(tabs[0].textContent).toEqual('Overall Progress');
      expect(tabs[1].textContent).toEqual('Listing Progress');
    });
  }));

  it('should display a overall, list and uploadGCS tabs when progress is available',
        async(() => {
    jobsServiceStub.getJobRun.and.returnValue(Observable.of(FAKE_JOB_RUNS[2]));
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('#job-progress-tabs');
      expect(element).not.toBeNull();
      const tabs = element.querySelectorAll('.mat-tab-label');
      expect(tabs.length).toEqual(3);
      expect(tabs[0].textContent).toEqual('Overall Progress');
      expect(tabs[1].textContent).toEqual('Listing Progress');
      expect(tabs[2].textContent).toEqual('Upload to GCS Progress');
    });
  }));

  it('should display a overall, list, uploadGCS and loadBQ tabs when progress is available',
        async(() => {
    jobsServiceStub.getJobRun.and.returnValue(Observable.of(FAKE_JOB_RUNS[3]));
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('#job-progress-tabs');
      expect(element).not.toBeNull();
      const tabs = element.querySelectorAll('.mat-tab-label');
      expect(tabs.length).toEqual(4);
      expect(tabs[0].textContent).toEqual('Overall Progress');
      expect(tabs[1].textContent).toEqual('Listing Progress');
      expect(tabs[2].textContent).toEqual('Upload to GCS Progress');
      expect(tabs[3].textContent).toEqual('Load into BigQuery Progress');
    });
  }));

  it('should show progress information in the overall progress tab', async(() => {
    const jobRun = FAKE_JOB_RUNS[0];
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const tabGroup = compiled.querySelector('#job-progress-tabs');
      expect(tabGroup).not.toBeNull();
      const tabs = tabGroup.querySelectorAll('.mat-tab-label');
      expect(tabs.length).toEqual(1);
      expect(tabs[0].textContent).toEqual('Overall Progress');
      const tabContents = compiled.querySelectorAll('.mat-tab-body-content');
      expect(tabContents).not.toBeNull();
      const infoList = tabContents[0].querySelector('dl');
      expect(infoList).not.toBeNull();
      const children = infoList.children;
      expect(children.length).toEqual(6);
      expect(children[0].innerText).toEqual('Total Tasks');
      expect(children[1].innerText).toEqual(String(jobRun.Counters.totalTasks));
      expect(children[2].innerText).toEqual('Tasks Completed');
      expect(children[3].innerText).toEqual(String(jobRun.Counters.tasksCompleted));
      expect(children[4].innerText).toEqual('Tasks Failed');
      expect(children[5].innerText).toEqual(String(jobRun.Counters.tasksFailed));
    });
  }));

  it('should show progress information in the listing progress tab', async(() => {
    const jobRun = FAKE_JOB_RUNS[1];
    jobsServiceStub.getJobRun.and.returnValue(Observable.of(jobRun));
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const tabGroup = compiled.querySelector('#job-progress-tabs');
      expect(tabGroup).not.toBeNull();
      const tabs = fixture.debugElement.queryAll(By.css('.mat-tab-label'));
      expect(tabs.length).toEqual(2);
      expect(tabs[1].nativeElement.textContent).toEqual('Listing Progress');
      tabs[1].nativeElement.click();
      fixture.detectChanges();
      const tabContents = compiled.querySelectorAll('.mat-tab-body-content');
      expect(tabContents).not.toBeNull();
      const infoList = tabContents[1].querySelector('dl');
      expect(infoList).not.toBeNull();
      const children = infoList.children;
      expect(children.length).toEqual(10);
      expect(children[0].innerText).toEqual('List Tasks');
      expect(children[1].innerText).toEqual(String(jobRun.Counters.totalTasksList));
      expect(children[2].innerText).toEqual('List Tasks Completed');
      expect(children[3].innerText).toEqual(String(jobRun.Counters.tasksCompletedList));
      expect(children[4].innerText).toEqual('List Tasks Failed');
      expect(children[5].innerText).toEqual(String(jobRun.Counters.tasksFailedList));
      expect(children[6].innerText).toEqual('Files Found');
      expect(children[7].innerText).toEqual(String(jobRun.Counters.listFilesFound));
      expect(children[8].innerText).toEqual('Bytes Found');
      expect(children[9].innerText).toEqual(String(jobRun.Counters.listBytesFound));
    });
  }));

  it('should show progress information in the upload gcs progress tab', async(() => {
    const jobRun = FAKE_JOB_RUNS[2];
    jobsServiceStub.getJobRun.and.returnValue(Observable.of(jobRun));
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const tabGroup = compiled.querySelector('#job-progress-tabs');
      expect(tabGroup).not.toBeNull();
      const tabs = fixture.debugElement.queryAll(By.css('.mat-tab-label'));
      expect(tabs.length).toEqual(3);
      expect(tabs[2].nativeElement.textContent).toEqual('Upload to GCS Progress');
      tabs[2].nativeElement.click();
      fixture.detectChanges();
      const tabContents = compiled.querySelectorAll('.mat-tab-body-content');
      expect(tabContents).not.toBeNull();
      const infoList = tabContents[2].querySelector('dl');
      expect(infoList).not.toBeNull();
      const children = infoList.children;
      expect(children.length).toEqual(8);
      expect(children[0].innerText).toEqual('Total Files');
      expect(children[1].innerText).toEqual(String(jobRun.Counters.totalTasksCopy));
      expect(children[2].innerText).toEqual('Files Completed');
      expect(children[3].innerText).toEqual(String(jobRun.Counters.tasksCompletedCopy));
      expect(children[4].innerText).toEqual('Files Failed');
      expect(children[5].innerText).toEqual(String(jobRun.Counters.tasksFailedCopy));
      expect(children[6].innerText).toEqual('Bytes Copied');
      expect(children[7].innerText).toEqual(String(jobRun.Counters.bytesCopied));
    });
  }));

  it('should show progress information in the load into BQ progress tab', async(() => {
    const jobRun = FAKE_JOB_RUNS[3];
    jobsServiceStub.getJobRun.and.returnValue(Observable.of(jobRun));
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const tabGroup = compiled.querySelector('#job-progress-tabs');
      expect(tabGroup).not.toBeNull();
      const tabs = fixture.debugElement.queryAll(By.css('.mat-tab-label'));
      expect(tabs.length).toEqual(4);
      expect(tabs[3].nativeElement.textContent).toEqual('Load into BigQuery Progress');
      tabs[3].nativeElement.click();
      fixture.detectChanges();
      const tabContents = compiled.querySelectorAll('.mat-tab-body-content');
      expect(tabContents).not.toBeNull();
      const infoList = tabContents[3].querySelector('dl');
      expect(infoList).not.toBeNull();
      const children = infoList.children;
      expect(children.length).toEqual(6);
      expect(children[0].innerText).toEqual('Total Objects');
      expect(children[1].innerText).toEqual(String(jobRun.Counters.totalTasksLoad));
      expect(children[2].innerText).toEqual('Objects Completed');
      expect(children[3].innerText).toEqual(String(jobRun.Counters.tasksCompletedLoad));
      expect(children[4].innerText).toEqual('Objects Failed');
      expect(children[5].innerText).toEqual(String(jobRun.Counters.tasksFailedLoad));
    });
  }));

  it('should get the job run every ten seconds', fakeAsync((done) => {
    intervalObservableCreateSpy.and.callThrough(); // enable polling
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    // It should get the job runs four times: one initial loading plus 3 polling calls.
    tick(30000);
    expect(jobsServiceStub.getJobRun.calls.count()).toEqual(4);
    discardPeriodicTasks();
  }));

});
