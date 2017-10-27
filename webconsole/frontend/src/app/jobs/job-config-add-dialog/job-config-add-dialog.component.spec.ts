import 'rxjs/add/observable/of';

import { async, TestBed } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { MatDialogRef } from '@angular/material';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { Observable } from 'rxjs/Observable';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { JobConfigRequest } from '../jobs.resources';
import { JobsService } from '../jobs.service';
import { JobConfigAddDialogComponent } from './job-config-add-dialog.component';

class JobsServiceStub {
  public postJobConfig = jasmine.createSpy('postJobConfig');
}

class MatDialogRefStub {
  public close = jasmine.createSpy('close');
}

let jobsServiceStub: JobsServiceStub;
let matDialogRefStub: MatDialogRefStub;

let fakeJobConfigModel: JobConfigRequest;

const FAKE_HTTP_ERROR = {error : {error: 'FakeError', message: 'Fake Error Message.'}};

const FAKE_JOB_CONFIG: JobConfigRequest = new JobConfigRequest(
  'fakeConfigId', 'fakeBucket', 'fakeFileSystemDir', 'fakeBigqueryDataset', 'fakeBigqueryTable');

const EMPTY_MODEL: JobConfigRequest = new JobConfigRequest('', '', '', '', '');

describe('JobConfigAddDialogComponent', () => {

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    matDialogRefStub = new MatDialogRefStub();
    jobsServiceStub.postJobConfig.and.returnValue(Observable.of(FAKE_JOB_CONFIG));
    fakeJobConfigModel = FAKE_JOB_CONFIG;
    TestBed.configureTestingModule({
      declarations: [
        JobConfigAddDialogComponent
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub},
        {provide: MatDialogRef, useValue: matDialogRefStub},
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

  it('onSubmit should call jobsService post job config', async(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobConfigModel;
    component.onSubmit();
    expect(jobsServiceStub.postJobConfig.calls.count()).toEqual(1);
    expect(jobsServiceStub.postJobConfig.calls.first().args[0]).toEqual(fakeJobConfigModel);
  }));

  it('onSubmit should close the dialog with "true" argument', async(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobConfigModel;
    component.onSubmit();
    expect(matDialogRefStub.close.calls.count()).toEqual(1);
    expect(matDialogRefStub.close.calls.first().args[0]).toEqual(true);
  }));

  it('should show error on submit error', async(() => {
    jobsServiceStub.postJobConfig.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobConfigModel;

    component.onSubmit();
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.textContent).toContain('FakeError');
    });
  }));

});
