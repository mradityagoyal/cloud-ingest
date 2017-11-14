import 'rxjs/add/observable/never';
import 'rxjs/add/observable/of';
import 'rxjs/add/observable/throw';
import 'rxjs/add/operator/delay';

import { async, TestBed } from '@angular/core/testing';
import { MatDialog } from '@angular/material';
import { RouterTestingModule } from '@angular/router/testing';
import { Observable } from 'rxjs/Observable';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { JobConfigAddDialogComponent } from '../job-config-add-dialog/job-config-add-dialog.component';
import { JobConfigResponse } from '../jobs.resources';
import { JobsService } from '../jobs.service';
import { FAKE_JOB_CONFIGS } from '../jobs.test-util';
import { JobConfigsComponent } from './job-configs.component';

class JobsServiceStub {
  public getJobConfigs = jasmine.createSpy('getJobConfigs');
}

class MatDialogStub {
  public open = jasmine.createSpy('open');
}

class MatDialogRefStub {
  public afterClosed = jasmine.createSpy('afterClosed');
}

const EMPTY_JOB_CONFIG_ARR: JobConfigResponse[] = [];

const FAKE_HTTP_ERROR = {error: 'fakeErrorText', message: 'Fake error message.'};

let jobsServiceStub: JobsServiceStub;
let mdDialogStub: MatDialogStub;
let mdDialogRefStub: MatDialogRefStub;

describe('JobConfigsComponent', () => {

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    mdDialogStub = new MatDialogStub();
    mdDialogRefStub = new MatDialogRefStub();
    jobsServiceStub.getJobConfigs.and.returnValue(Observable.of(FAKE_JOB_CONFIGS));
    mdDialogStub.open.and.returnValue(mdDialogRefStub);
    mdDialogRefStub.afterClosed.and.returnValue(Observable.of(false));

    TestBed.configureTestingModule({
      declarations: [
        JobConfigsComponent
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub},
        {provide: MatDialog, useValue: mdDialogStub}
      ],
      imports: [
        AngularMaterialImporterModule,
        RouterTestingModule
      ],
    }).compileComponents();
  }));

  it('should create the job runs component', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component).toBeTruthy();
  }));

  it('should initialize the component with show loading spinner as false', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component.showLoadingSpinner = false);
  }));

  it('should initialize the component with display error message as false', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component.displayErrorMessage = false);
  }));

  it('should show a loading spinner while job configs are loading', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    component.showLoadingSpinner = true;
    jobsServiceStub.getJobConfigs.and.returnValue(Observable.never());
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('mat-spinner');
      expect(element).not.toBeNull();
    });
  }));

  it('should not show a loading spinner when job configs return', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    component.showLoadingSpinner = false;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('mat-spinner');
      expect(element).toBeNull();
    });
  }));

  it('should contain the job config information', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.textContent).toContain('fakeJobConfigId1');
      expect(compiled.textContent).toContain('fakeSrcDir1');
      expect(compiled.textContent).toContain('fakeBucket1');

      expect(compiled.textContent).toContain('fakeJobConfigId2');
      expect(compiled.textContent).toContain('fakeSrcDir2');
      expect(compiled.textContent).toContain('fakeBucket2');

      expect(compiled.textContent).toContain('fakeJobConfigId3');
      expect(compiled.textContent).toContain('fakeSrcDir3');
      expect(compiled.textContent).toContain('fakeBucket3');
    });
  }));

  it('should contain an add job configuration button', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    component.showLoadingSpinner = false;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-add-job-config');
      expect(element).not.toBeNull();
    });
  }));

  it('should open an add job config dialog when clicked', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    component.showLoadingSpinner = false;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-add-job-config');
      element.click();
      expect(mdDialogStub.open).toHaveBeenCalled();
      expect(expect(mdDialogStub.open.calls.first().args[0]).toBe(JobConfigAddDialogComponent));
    });
  }));

  it('should display an error message if getJobConfigs returns an error', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    jobsServiceStub.getJobConfigs.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-error-message');
      expect(element).not.toBeNull();
    });
  }));

it('should retrieve the title and text from the HttpErrorResponseFormatter', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    jobsServiceStub.getJobConfigs.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    spyOn(HttpErrorResponseFormatter, 'getTitle').and.callFake(function(httpError) {
      expect(httpError).toBe(FAKE_HTTP_ERROR);
      return 'fakeFormattedTitle';
    });
    spyOn(HttpErrorResponseFormatter, 'getMessage').and.callFake(function(httpError) {
      expect(httpError).toBe(FAKE_HTTP_ERROR);
      return 'fakeFormattedMessage';
    });
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-error-message');
      expect(element.textContent).toContain('fakeFormattedTitle');
      expect(element.textContent).toContain('fakeFormattedMessage');
    });
  }));

  it('should open the add job config dialog if there are no job configurations', async(() => {
    jobsServiceStub.getJobConfigs.and.returnValue(Observable.of(EMPTY_JOB_CONFIG_ARR));
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      expect(mdDialogStub.open).toHaveBeenCalled();
      expect(expect(mdDialogStub.open.calls.first().args[0]).toBe(JobConfigAddDialogComponent));
    });
  }));
});
