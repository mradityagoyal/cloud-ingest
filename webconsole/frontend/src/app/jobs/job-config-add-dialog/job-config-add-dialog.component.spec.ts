import 'rxjs/add/observable/of';

import { async, TestBed } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { MatDialogRef, MatSnackBar } from '@angular/material';
import { Observable } from 'rxjs/Observable';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { JobConfig } from '../jobs.resources';
import { JobsService } from '../jobs.service';
import { JobConfigAddDialogComponent } from './job-config-add-dialog.component';
import { JobConfigFormModel } from './job-config-add-dialog.resources';

class JobsServiceStub {
  public postJobConfig = jasmine.createSpy('postJobConfig');
}

class MatDialogRefStub {
  public close = jasmine.createSpy('close');
}

class MatSnackBarStub {
  public open = jasmine.createSpy('open');
}

let jobsServiceStub: JobsServiceStub;
let matDialogRefStub: MatDialogRefStub;
let matSnackbarStub: MatSnackBarStub;
let fakeJobConfigModel: JobConfigFormModel;

const FAKE_HTTP_ERROR = {error : {error: 'FakeError', message: 'Fake Error Message.'}};

const FAKE_JOB_CONFIG: JobConfig = {
  JobConfigId : 'fake-config-2',
  JobSpec : '{ "on_prem" : "fake_spec", "gcs_dest" : "fake_spec"}',
};
const EMPTY_MODEL = new JobConfigFormModel(
    /** jobConfigId **/ '',
    /** gcsBucket **/ '',
    /** fileSystemDirectory **/ '',
    /** bigqueryDataset **/ '',
    /** bigqueryTable **/ '');

describe('JobConfigAddDialogComponent', () => {

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    matDialogRefStub = new MatDialogRefStub();
    matSnackbarStub = new MatSnackBarStub();
    jobsServiceStub.postJobConfig.and.returnValue(Observable.of(FAKE_JOB_CONFIG));
    fakeJobConfigModel = new JobConfigFormModel(
                        /**jobConfigId**/ 'fakeJobConfigId',
                       /**gcsBucket**/ 'fakeGcsBucket',
                       /**fileSystemDirectory**/
                           'fake/file/system/dir',
                       /**bigqueryDataset**/ 'fakeBigqueryDataset',
                       /**bigqueryTable**/ 'fakeBigqueryTable');
    TestBed.configureTestingModule({
      declarations: [
        JobConfigAddDialogComponent
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub},
        {provide: MatDialogRef, useValue: matDialogRefStub},
        {provide: MatSnackBar, useValue: matSnackbarStub},
      ],
      imports: [
        FormsModule,
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
    expect(component.bigQueryTransferChecked).toBe(false);
    expect(component.formError).toBe(false);
    expect(component.appError).toBe(false);
    expect(component.model).toEqual(EMPTY_MODEL);
  }));

  it('onSubmit should call jobsService post job config', async(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobConfigModel;
    component.onSubmit();
    expect(jobsServiceStub.postJobConfig.calls.count()).toEqual(1);
    expect(jobsServiceStub.postJobConfig.calls.first().args[0]).toEqual(fakeJobConfigModel.toApiJobConfig());
  }));

  it('onSubmit should close the dialog with "true" argument', async(() => {
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobConfigModel;
    component.onSubmit();
    expect(matDialogRefStub.close.calls.count()).toEqual(1);
    expect(matDialogRefStub.close.calls.first().args[0]).toEqual(true);
  }));

  it('onSubmit should open a snackbar on error', async(() => {
    jobsServiceStub.postJobConfig.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(JobConfigAddDialogComponent);
    const component = fixture.debugElement.componentInstance;
    component.model = fakeJobConfigModel;
    component.onSubmit();
    expect(matSnackbarStub.open.calls.count()).toEqual(1);
    expect(matSnackbarStub.open.calls.first().args[0]).toContain('FakeError');
  }));
});
