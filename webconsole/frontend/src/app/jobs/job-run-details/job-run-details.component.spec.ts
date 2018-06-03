import { async, ComponentFixture, discardPeriodicTasks, fakeAsync, inject, TestBed, tick } from '@angular/core/testing';
import { MatDialog } from '@angular/material';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { RouterTestingModule } from '@angular/router/testing';
import { never, of, throwError as observableThrowError } from 'rxjs';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { FAKE_HTTP_ERROR, MatDialogStub } from '../../util/common.test-util';
import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { JobsService } from '../jobs.service';
import { FAKE_TRANSFER_JOB_RESPONSE, JobsServiceStub } from '../jobs.test-util';
import { ENABLE_POLLING, JobRunDetailsComponent } from './job-run-details.component';


let jobsServiceStub: JobsServiceStub;
let matDialogStub: MatDialogStub;

describe('JobRunDetailsComponent', () => {
  let component: JobRunDetailsComponent;
  let fixture: ComponentFixture<JobRunDetailsComponent>;

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    matDialogStub = new MatDialogStub();
    jobsServiceStub.getJob.and.returnValue(of(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0]));

    TestBed.configureTestingModule({
      declarations: [
        JobRunDetailsComponent,
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub},
        {provide: MatDialog, useValue: matDialogStub},
        {provide: ENABLE_POLLING, useValue: false},
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
    jobsServiceStub.getJob.and.returnValue(never());
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
    jobsServiceStub.getJob.and.returnValue(observableThrowError(FAKE_HTTP_ERROR));
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
    inject([ENABLE_POLLING], (enablePolling: true) => {
      fixture = TestBed.createComponent(JobRunDetailsComponent);
      component = fixture.componentInstance;
      fixture.detectChanges();
      // It should get the job runs four times: one initial loading plus 3 polling calls.
      tick(30000);
      expect(jobsServiceStub.getJob.calls.count()).toEqual(4);
      discardPeriodicTasks();
    });
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
    inject([ENABLE_POLLING], (enablePolling: true) => {
      // Load successfully on first call, but throw on second call.
      jobsServiceStub.getJob.and.returnValues(of(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0]), observableThrowError(FAKE_HTTP_ERROR));
      fixture = TestBed.createComponent(JobRunDetailsComponent);
      component = fixture.componentInstance;
      fixture.detectChanges();
      tick(10000); // Tick for long enough until the app makes a polling call.
      expect(matDialogStub.open).toHaveBeenCalled();
      expect(matDialogStub.open.calls.mostRecent().args[0]).toBe(ErrorDialogComponent);
      discardPeriodicTasks();
    });
  }));

});
