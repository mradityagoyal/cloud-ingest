import { async, fakeAsync, TestBed, tick } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { throwError as observableThrowError, of } from 'rxjs';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { AuthService } from '../../auth/auth.service';
import { AuthServiceStub, FAKE_HTTP_ERROR, MatDialogRefStub } from '../../util/common.test-util';
import { TransferJob } from '../jobs.resources';
import { JobsService } from '../jobs.service';
import { FAKE_TRANSFER_JOB_RESPONSE, JobsServiceStub } from '../jobs.test-util';
import { JobConfigAddDialogComponent } from './job-config-add-dialog.component';



let jobsServiceStub: JobsServiceStub;
let matDialogRefStub: MatDialogRefStub;
let fakeJobModel: TransferJob;
let authServiceStub: AuthServiceStub;
const EMPTY_MODEL = new TransferJob();


describe('JobConfigAddDialogComponent', () => {

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    matDialogRefStub = new MatDialogRefStub();
    authServiceStub = new AuthServiceStub();
    authServiceStub.grantBucketPermissionsIfNotExist.and.returnValue(Promise.resolve(true));
    authServiceStub.grantPubsubTopicPermissionsIfNotExists.and.returnValue(Promise.resolve(true));
    jobsServiceStub.postJob.and.returnValue(of(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0]));
    fakeJobModel = new TransferJob();
    TestBed.configureTestingModule({
      declarations: [
        JobConfigAddDialogComponent
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub},
        {provide: AuthService, useValue: authServiceStub},
        {provide: MatDialogRef, useValue: matDialogRefStub},
        {provide: MAT_DIALOG_DATA, useValue: EMPTY_MODEL}
      ],
      imports: [
        FormsModule,
        BrowserAnimationsModule,
        AngularMaterialImporterModule
      ],
    }).compileComponents();
  }));

  it('should create the job config add dialog component', async(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component).toBeTruthy();
  }));

  it('should initialize the component with expected values', async(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component.submittingForm).toBe(false);
    expect(component.model).toEqual(EMPTY_MODEL);
  }));

  it('onSubmit should call jobsService post job config', fakeAsync(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobModel;
    component.onSubmit();
    tick(50);
    expect(authServiceStub.grantBucketPermissionsIfNotExist.calls.count()).toEqual(1);
    expect(jobsServiceStub.postJob.calls.count()).toEqual(1);
    expect(jobsServiceStub.postJob.calls.first().args[0]).toEqual(fakeJobModel);
  }));

  it('onSubmit should close the dialog with "true" argument', fakeAsync(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobModel;
    component.onSubmit();
    tick(50);
    expect(authServiceStub.grantBucketPermissionsIfNotExist.calls.count()).toEqual(1);
    expect(matDialogRefStub.close.calls.count()).toEqual(1);
    expect(matDialogRefStub.close.calls.first().args[0]).toEqual(true);
  }));

  it('onSubmit should append a trailing slash', fakeAsync(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobModel;
    fakeJobModel.transferSpec.gcsDataSink.objectPrefix = 'fakePrefix';
    component.onSubmit();
    tick(50);
    expect(matDialogRefStub.close.calls.count()).toEqual(1);
    expect(jobsServiceStub.postJob.calls.first().args[0].transferSpec.gcsDataSink.objectPrefix).toEqual('fakePrefix/');
  }));

  it('onSubmit should leave the field blank', fakeAsync(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobModel;
    fakeJobModel.transferSpec.gcsDataSink.objectPrefix = '';
    component.onSubmit();
    tick(50);
    expect(matDialogRefStub.close.calls.count()).toEqual(1);
    expect(jobsServiceStub.postJob.calls.first().args[0].transferSpec.gcsDataSink.objectPrefix).toEqual('');
  }));

  it('should show error on submit error', async(() => {
    jobsServiceStub.postJob.and.returnValue(observableThrowError(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobModel;

    component.onSubmit();
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.textContent).toContain('FakeError');
    });
  }));

  it('should show error on grant permissions to bucket error', fakeAsync(() => {
    authServiceStub.grantBucketPermissionsIfNotExist.and.returnValue(Promise.reject(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobModel;

    component.onSubmit();
    tick(50);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.textContent).toContain('error');
    });
  }));

  it('should show error on grant permissions to project error', fakeAsync(() => {
    authServiceStub.grantPubsubTopicPermissionsIfNotExists.and.returnValue(Promise.reject(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobModel;

    component.onSubmit();
    tick(50);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.textContent).toContain('error');
    });
  }));

});
