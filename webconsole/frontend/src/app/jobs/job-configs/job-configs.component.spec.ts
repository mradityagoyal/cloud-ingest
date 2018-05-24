import 'rxjs/add/observable/never';
import 'rxjs/add/observable/of';
import 'rxjs/add/observable/throw';
import 'rxjs/add/operator/delay';

import { async, TestBed } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { MatDialog } from '@angular/material';
import { RouterTestingModule } from '@angular/router/testing';
import { Observable } from 'rxjs/Observable';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { FAKE_HTTP_ERROR, MatDialogRefStub, MatDialogStub } from '../../util/common.test-util';
import { ErrorDialogComponent } from '../../util/error-dialog/error-dialog.component';
import { HttpErrorResponseFormatter } from '../../util/error.resources';
import { JobConfigAddDialogComponent } from '../job-config-add-dialog/job-config-add-dialog.component';
import { JobsService } from '../jobs.service';
import { FAKE_TRANSFER_JOB_RESPONSE, JobsServiceStub, EMPTY_TRANSFER_JOB_RESPONSE } from '../jobs.test-util';
import { JobConfigsComponent } from './job-configs.component';

let jobsServiceStub: JobsServiceStub;
let matDialogStub: MatDialogStub;
let matDialogRefStub: MatDialogRefStub;

describe('JobConfigsComponent', () => {

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    matDialogStub = new MatDialogStub();
    matDialogRefStub = new MatDialogRefStub();
    jobsServiceStub.getJobs.and.returnValue(Observable.of(FAKE_TRANSFER_JOB_RESPONSE));
    matDialogStub.open.and.returnValue(matDialogRefStub);
    matDialogRefStub.afterClosed.and.returnValue(Observable.of(false));

    TestBed.configureTestingModule({
      declarations: [
        JobConfigsComponent
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub},
        {provide: MatDialog, useValue: matDialogStub}
      ],
      imports: [
        AngularMaterialImporterModule,
        RouterTestingModule,
        FormsModule
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
    jobsServiceStub.getJobs.and.returnValue(Observable.never());
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

  it('should contain the job information', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0].description);
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0].transferSpec.onPremFiler.directoryPath);
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[0].transferSpec.gcsDataSink.bucketName);

      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[1].description);
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[1].transferSpec.onPremFiler.directoryPath);
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[1].transferSpec.gcsDataSink.bucketName);

      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[2].description);
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[2].transferSpec.onPremFiler.directoryPath);
      expect(compiled.textContent).toContain(FAKE_TRANSFER_JOB_RESPONSE.transferJobs[2].transferSpec.gcsDataSink.bucketName);
    });
  }));

  it('should contain an add job button', async(() => {
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

  it('should open an add job dialog when clicked', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    component.showLoadingSpinner = false;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-add-job-config');
      element.click();
      expect(matDialogStub.open).toHaveBeenCalled();
      expect(expect(matDialogStub.open.calls.first().args[0]).toBe(JobConfigAddDialogComponent));
    });
  }));

  it('should display an error message if getJobs returns an error', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    jobsServiceStub.getJobs.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
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
    jobsServiceStub.getJobs.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
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

  it('should open the add job dialog if there are no jobs', async(() => {
    jobsServiceStub.getJobs.and.returnValue(Observable.of(EMPTY_TRANSFER_JOB_RESPONSE));
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      expect(matDialogStub.open).toHaveBeenCalled();
      expect(expect(matDialogStub.open.calls.first().args[0]).toBe(JobConfigAddDialogComponent));
    });
  }));

  it('should open the add job config dialog if there are no job configurations', async(() => {
    jobsServiceStub.getJobs.and.returnValue(Observable.of(EMPTY_TRANSFER_JOB_RESPONSE));
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      expect(matDialogStub.open).toHaveBeenCalled();
      expect(expect(matDialogStub.open.calls.first().args[0]).toBe(JobConfigAddDialogComponent));
    });
  }));

  it('should pause the checked jobs', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const compiled = fixture.debugElement.nativeElement;
      const checkbox1 = compiled.querySelector('#transferJobs\\/OPI1-input');
      const checkbox2 = compiled.querySelector('#transferJobs\\/OPI3-input');
      checkbox1.click();
      checkbox2.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        const pauseConfigButton = compiled.querySelector('.ingest-pause-job');
        pauseConfigButton.click();
        fixture.detectChanges();
        fixture.whenStable().then(() => {
          const component = fixture.debugElement.componentInstance;
          expect(jobsServiceStub.pauseJobs).toHaveBeenCalledWith(['transferJobs/OPI1', 'transferJobs/OPI3']);
        });
      });
    });
  }));

  it('should not pause jobs if none are checked', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    const compiled = fixture.debugElement.nativeElement;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const pauseConfigButton = compiled.querySelector('.ingest-pause-job');
      pauseConfigButton.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        expect(jobsServiceStub.pauseJobs).not.toHaveBeenCalled();
      });
    });
  }));

  it('should open an error dialog if there is an error pausing job configurations', async(() => {
    jobsServiceStub.pauseJobs.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const compiled = fixture.debugElement.nativeElement;
      const checkbox1 = compiled.querySelector('#transferJobs\\/OPI1-input');
      const checkbox2 = compiled.querySelector('#transferJobs\\/OPI3-input');
      checkbox1.click();
      checkbox2.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        const pauseConfigButton = compiled.querySelector('.ingest-pause-job');
        pauseConfigButton.click();
        fixture.detectChanges();
        fixture.whenStable().then(() => {
          expect(matDialogStub.open).toHaveBeenCalled();
          expect(matDialogStub.open.calls.first().args[0]).toBe(ErrorDialogComponent);
        });
      });
    });
  }));

  it('should resume the checked jobs', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const compiled = fixture.debugElement.nativeElement;
      const checkbox1 = compiled.querySelector('#transferJobs\\/OPI1-input');
      const checkbox2 = compiled.querySelector('#transferJobs\\/OPI3-input');
      checkbox1.click();
      checkbox2.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        const resumeConfigButton = compiled.querySelector('.ingest-resume-job');
        resumeConfigButton.click();
        fixture.detectChanges();
        fixture.whenStable().then(() => {
          const component = fixture.debugElement.componentInstance;
          expect(jobsServiceStub.resumeJobs).toHaveBeenCalledWith(['transferJobs/OPI1', 'transferJobs/OPI3']);
        });
      });
    });
  }));

  it('should not resume jobs if none are checked', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    const compiled = fixture.debugElement.nativeElement;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const resumeConfigButton = compiled.querySelector('.ingest-resume-job');
      resumeConfigButton.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        expect(jobsServiceStub.resumeJobs).not.toHaveBeenCalled();
      });
    });
  }));

  it('should open an error dialog if there is an error resuming job configurations', async(() => {
    jobsServiceStub.resumeJobs.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const compiled = fixture.debugElement.nativeElement;
      const checkbox1 = compiled.querySelector('#transferJobs\\/OPI1-input');
      const checkbox2 = compiled.querySelector('#transferJobs\\/OPI3-input');
      checkbox1.click();
      checkbox2.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        const resumeConfigButton = compiled.querySelector('.ingest-resume-job');
        resumeConfigButton.click();
        fixture.detectChanges();
        fixture.whenStable().then(() => {
          expect(matDialogStub.open).toHaveBeenCalled();
          expect(matDialogStub.open.calls.first().args[0]).toBe(ErrorDialogComponent);
        });
      });
    });
  }));

  it('should allow to delete a paused job ', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const compiled = fixture.debugElement.nativeElement;
      const checkbox2 = compiled.querySelector('#transferJobs\\/OPI3-input');
      checkbox2.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        const deleteJobButton = compiled.querySelector('.ingest-delete-job');
        deleteJobButton.click();
        fixture.detectChanges();
        fixture.whenStable().then(() => {
          const component = fixture.debugElement.componentInstance;
          expect(jobsServiceStub.deleteJobs).toHaveBeenCalledWith(['transferJobs/OPI3']);
        });
      });
    });
  }));


  it('should not delete a job in progress', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const compiled = fixture.debugElement.nativeElement;
      const checkbox1 = compiled.querySelector('#transferJobs\\/OPI1-input');
      const checkbox2 = compiled.querySelector('#transferJobs\\/OPI2-input');
      checkbox1.click();
      checkbox2.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        const deleteJobButton = compiled.querySelector('.ingest-delete-job');
        deleteJobButton.click();
        fixture.detectChanges();
        fixture.whenStable().then(() => {
          const component = fixture.debugElement.componentInstance;
          expect(jobsServiceStub.deleteJobs).not.toHaveBeenCalled();
        });
      });
    });
  }));

  it('should open an error dialog if there is an error deleting a job', async(() => {
    jobsServiceStub.deleteJobs.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      const compiled = fixture.debugElement.nativeElement;
      const checkbox = compiled.querySelector('#transferJobs\\/OPI3-input');
      checkbox.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        const deleteJobButton = compiled.querySelector('.ingest-delete-job');
        deleteJobButton.click();
        fixture.detectChanges();
        fixture.whenStable().then(() => {
          expect(matDialogStub.open).toHaveBeenCalled();
          expect(matDialogStub.open.calls.first().args[0]).toBe(ErrorDialogComponent);
        });
      });
    });
  }));

});
