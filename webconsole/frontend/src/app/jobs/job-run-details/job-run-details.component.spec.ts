import { async, ComponentFixture, discardPeriodicTasks, fakeAsync, TestBed, tick } from '@angular/core/testing';
import { MatDialog } from '@angular/material';
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { RouterTestingModule } from '@angular/router/testing';
import { Observable } from 'rxjs/Observable';
import { IntervalObservable } from 'rxjs/observable/IntervalObservable';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { FAKE_HTTP_ERROR, MatDialogStub } from '../../util/common.test-util';
import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { JobStatusPipe } from '../job-status/job-status.pipe';
import { JobsService } from '../jobs.service';
import { FAKE_TRANSFER_JOB_RESPONSE, JobsServiceStub } from '../jobs.test-util';
import { JobRunDetailsComponent } from './job-run-details.component';

let jobsServiceStub: JobsServiceStub;
let matDialogStub: MatDialogStub;
let intervalObservableCreateSpy: any;

describe('JobRunDetailsComponent', () => {
  let component: JobRunDetailsComponent;
  let fixture: ComponentFixture<JobRunDetailsComponent>;

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    matDialogStub = new MatDialogStub();
    jobsServiceStub.getJob.and.returnValue(Observable.of(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0]));
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
    jobsServiceStub.getJob.and.returnValue(Observable.never());
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
    jobsServiceStub.getJob.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
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

  it('should get the job every ten seconds', fakeAsync((done) => {
    intervalObservableCreateSpy.and.callThrough(); // enable polling
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    // It should get the job runs four times: one initial loading plus 3 polling calls.
    tick(30000);
    expect(jobsServiceStub.getJob.calls.count()).toEqual(4);
    discardPeriodicTasks();
  }));

  it('should display the job information', async(() => {
    fixture = TestBed.createComponent(JobRunDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0].name);
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0].transferSpec.onPremFiler.directoryPath);
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0].transferSpec.gcsDataSink.bucketName);
    });
  }));

  it('should open the mat dialog stub with the error dialog', fakeAsync((done) => {
    // Load successfully on first call, but throw on second call.
    jobsServiceStub.getJob.and.returnValues(Observable.of(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0]), Observable.throw(FAKE_HTTP_ERROR));
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
