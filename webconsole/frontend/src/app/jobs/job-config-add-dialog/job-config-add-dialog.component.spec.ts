import { TestBed, async } from '@angular/core/testing';
import { JobConfigAddDialogComponent } from './job-config-add-dialog.component';
import { JobsService } from '../jobs.service';
import { MdDialogRef } from '@angular/material';
import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { FormsModule } from '@angular/forms';
import { JobConfigFormModel } from './job-config-add-dialog.resources';
import { Observable } from 'rxjs/Observable';
import { JobConfig } from '../jobs.resources';
import 'rxjs/add/observable/of';

class JobsServiceStub {
  public postJobConfig = jasmine.createSpy('postJobConfig');
}

class MdDialogRefStub {
  public close = jasmine.createSpy('close');
}

let jobsServiceStub: JobsServiceStub;
let mdDialogRefStub: MdDialogRefStub;
let fakeJobConfigModel: JobConfigFormModel;


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
    mdDialogRefStub = new MdDialogRefStub();
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
        {provide: MdDialogRef, useValue: mdDialogRefStub},
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
    expect(mdDialogRefStub.close.calls.count()).toEqual(1);
    expect(mdDialogRefStub.close.calls.first().args[0]).toEqual(true);
  }));
});
