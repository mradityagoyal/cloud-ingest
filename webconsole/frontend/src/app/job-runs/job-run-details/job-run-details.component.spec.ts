import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { By } from '@angular/platform-browser';
import { AngularMaterialImporterModule } from '../../angular-material-importer.module';
import { JobsService } from '../../jobs.service';
import { JobRun } from '../../api.resources';
import { JobStatusPipe } from '../job-status.pipe';
import { Observable } from 'rxjs/Observable';

import { JobRunDetailsComponent } from './job-run-details.component';

class JobsServiceStub {
  public getJobRun = jasmine.createSpy('getJobRun');
}

const FAKE_JOB_RUNS: JobRun[] = [
  {
    JobConfigId: 'fakeJobConfigId1',
    JobRunId: 'fakeJobRunId1',
    JobCreationTime: '1504833274371000000',
    Status: 0,
    Progress: {
      totalTasks: 0,
      tasksCompleted: 0,
      tasksFailed: 0
    }
  },
  {
    JobConfigId: 'fakeJobConfigId2',
    JobRunId: 'fakeJobRunId2',
    JobCreationTime: '1504833274371000000',
    Status: 1,
    Progress: {
      totalTasks: 1,
      tasksCompleted: 0,
      tasksFailed: 0,
      list: {
        totalLists: 1,
        listsCompleted: 0,
        listsFailed: 0
      }
    }
  },
  {
    JobConfigId: 'fakeJobConfigId3',
    JobRunId: 'fakeJobRunId3',
    JobCreationTime: '1504833274371000000',
    Status: 2,
    Progress: {
      totalTasks: 5,
      tasksCompleted: 4,
      tasksFailed: 1,
      list: {
        totalLists: 1,
        listsCompleted: 1,
        listsFailed: 0
      },
      uploadGCS: {
        totalFiles: 4,
        filesCompleted: 3,
        filesFailed: 1
      }
    },
  },
  {
    JobConfigId: 'fakeJobConfigId4',
    JobRunId: 'fakeJobRunId4',
    JobCreationTime: '1504833274371000000',
    Status: 3,
    Progress: {
      totalTasks: 9,
      tasksCompleted: 9,
      tasksFailed: 0,
      list: {
        totalLists: 1,
        listsCompleted: 1,
        listsFailed: 0
      },
      uploadGCS: {
        totalFiles: 4,
        filesCompleted: 4,
        filesFailed: 0
      },
      loadBigQuery: {
        totalObjects: 4,
        objectsCompleted: 4,
        objectsFailed: 0
      }
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

describe('JobRunDetailsComponent', () => {
  let component: JobRunDetailsComponent;
  let fixture: ComponentFixture<JobRunDetailsComponent>;

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    jobsServiceStub.getJobRun.and.returnValue(Observable.of(FAKE_JOB_RUNS[0]));
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
      const element = compiled.querySelector('md-spinner');
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
      const element = compiled.querySelector('md-spinner');
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
      expect(children[1].innerText).toEqual('' + jobRun.Progress.totalTasks);
      expect(children[2].innerText).toEqual('Tasks Completed');
      expect(children[3].innerText).toEqual('' + jobRun.Progress.tasksCompleted);
      expect(children[4].innerText).toEqual('Tasks Failed');
      expect(children[5].innerText).toEqual('' + jobRun.Progress.tasksFailed);
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
      expect(children.length).toEqual(6);
      expect(children[0].innerText).toEqual('List Tasks');
      expect(children[1].innerText).toEqual('' + jobRun.Progress.list.totalLists);
      expect(children[2].innerText).toEqual('List Tasks Completed');
      expect(children[3].innerText).toEqual('' + jobRun.Progress.list.listsCompleted);
      expect(children[4].innerText).toEqual('List Tasks Failed');
      expect(children[5].innerText).toEqual('' + jobRun.Progress.list.listsFailed);
    });
  }));

  it('should show progress information in the upload gcs progress tab', async(() => {
    const jobRun = FAKE_JOB_RUNS[2];
    const gcsProgress = jobRun.Progress.uploadGCS;
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
      expect(children.length).toEqual(6);
      expect(children[0].innerText).toEqual('Total Files');
      expect(children[1].innerText).toEqual('' + gcsProgress.totalFiles);
      expect(children[2].innerText).toEqual('Files Completed');
      expect(children[3].innerText).toEqual('' + gcsProgress.filesCompleted);
      expect(children[4].innerText).toEqual('Files Failed');
      expect(children[5].innerText).toEqual('' + gcsProgress.filesFailed);
    });
  }));

  it('should show progress information in the load into BQ progress tab', async(() => {
    const jobRun = FAKE_JOB_RUNS[3];
    const bqProgress = jobRun.Progress.loadBigQuery;
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
      expect(children[1].innerText).toEqual('' + bqProgress.totalObjects);
      expect(children[2].innerText).toEqual('Objects Completed');
      expect(children[3].innerText).toEqual('' + bqProgress.objectsCompleted);
      expect(children[4].innerText).toEqual('Objects Failed');
      expect(children[5].innerText).toEqual('' + bqProgress.objectsFailed);
    });
  }));

});
