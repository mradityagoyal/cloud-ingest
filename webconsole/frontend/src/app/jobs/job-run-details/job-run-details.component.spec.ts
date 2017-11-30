import { async, ComponentFixture, discardPeriodicTasks, fakeAsync, TestBed, tick } from '@angular/core/testing';
import { MatDialog } from '@angular/material';
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { RouterTestingModule } from '@angular/router/testing';
import { Observable } from 'rxjs/Observable';
import { IntervalObservable } from 'rxjs/observable/IntervalObservable';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { JobStatusPipe } from '../job-status/job-status.pipe';
import { JobsService } from '../jobs.service';
import { FAKE_HTTP_ERROR, FAKE_JOB_RUNS } from '../jobs.test-util';
import { JobRunDetailsComponent } from './job-run-details.component';

class JobsServiceStub {
  public getJobRun = jasmine.createSpy('getJobRun');
}

class MatDialogStub {
  public open = jasmine.createSpy('open');
}

let jobsServiceStub: JobsServiceStub;
let matDialogStub: MatDialogStub;
let intervalObservableCreateSpy: any;

describe('JobRunDetailsComponent', () => {
  let component: JobRunDetailsComponent;
  let fixture: ComponentFixture<JobRunDetailsComponent>;

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    matDialogStub = new MatDialogStub();
    jobsServiceStub.getJobRun.and.returnValue(Observable.of(FAKE_JOB_RUNS[0]));
    // Disable polling for most tests.
    intervalObservableCreateSpy = spyOn(IntervalObservable, 'create').and.returnValue(Observable.never());
    TestBed.configureTestingModule({
      declarations: [
        JobRunDetailsComponent,
        JobStatusPipe
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub},
        {provide: MatDialog, useValue: matDialogStub}
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
      expect(compiled.textContent).toContain(FAKE_HTTP_ERROR.error.error);
      expect(compiled.textContent).toContain(FAKE_HTTP_ERROR.error.message);
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

  it('should display the job config information', async(() => {
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.textContent).toContain('fakeJobConfigId0');
      expect(compiled.textContent).toContain('fakeSrcDir1');
      expect(compiled.textContent).toContain('fakeBucket1');
    });
  }));

  it('should open the mat dialog stub with the error dialog', fakeAsync((done) => {
    // Load successfully on first call, but throw on second call.
    jobsServiceStub.getJobRun.and.returnValues(Observable.of(FAKE_JOB_RUNS[0]), Observable.throw(FAKE_HTTP_ERROR));
    intervalObservableCreateSpy.and.callThrough();
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    tick(10000); // Tick for long enough until the app makes a polling call.
    expect(matDialogStub.open).toHaveBeenCalled();
    expect(matDialogStub.open.calls.mostRecent().args[0]).toBe(ErrorDialogComponent);
    discardPeriodicTasks();
  }));

});
