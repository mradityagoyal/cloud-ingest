import 'rxjs/add/observable/never';
import 'rxjs/add/observable/of';
import 'rxjs/add/observable/throw';

import { async, discardPeriodicTasks, fakeAsync, TestBed, tick } from '@angular/core/testing';
import { MatDialog, MatSnackBar } from '@angular/material';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { IntervalObservable } from 'rxjs/observable/IntervalObservable';

import { AngularMaterialImporterModule } from '../angular-material-importer/angular-material-importer.module';
import { ActivatedRouteStub, FAKE_HTTP_ERROR, MatDialogStub, MatSnackBarStub } from '../util/common.test-util';
import { ErrorDialogComponent } from '../util/error-dialog/error-dialog.component';
import { HttpErrorResponseFormatter } from '../util/error.resources';
import { InfrastructureStatusItemComponent } from './infrastructure-status-item/infrastructure-status-item.component';
import { InfrastructureComponent } from './infrastructure.component';
import { InfrastructureService } from './infrastructure.service';
import {
  FAKE_INFRA_STATUS_DELETING,
  FAKE_INFRA_STATUS_DEPLOYING,
  FAKE_INFRA_STATUS_FAILED,
  FAKE_INFRA_STATUS_NOT_DETERMINED,
  FAKE_INFRA_STATUS_NOT_FOUND,
  FAKE_INFRA_STATUS_RUNNING,
  FAKE_INFRA_STATUS_UNKNOWN,
  InfrastructureServiceStub,
} from './infrastructure.test-util';

let intervalObservableCreateSpy: any;
let windowConfirmSpy: any;

let infrastructureServiceStub: InfrastructureServiceStub;
let matSnackBarStub: MatSnackBarStub;
let activatedRouteStub: ActivatedRouteStub;
let matDialogStub: MatDialogStub;

describe('InfrastructureComponent', () => {

  beforeEach(async(() => {
    infrastructureServiceStub = new InfrastructureServiceStub();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));
    infrastructureServiceStub.postCreateInfrastructure.and.returnValue(Observable.of({}));
    infrastructureServiceStub.postTearDownInfrastructure.and.returnValue(Observable.of({}));
    // Disable polling for most tests
    intervalObservableCreateSpy = spyOn(IntervalObservable, 'create').and.returnValue(Observable.never());

    windowConfirmSpy = spyOn(window, 'confirm').and.returnValue(true);
    matSnackBarStub = new MatSnackBarStub();
    activatedRouteStub = new ActivatedRouteStub();
    matDialogStub = new MatDialogStub();

    TestBed.configureTestingModule({
      declarations: [
        InfrastructureComponent,
        InfrastructureStatusItemComponent
      ],
      providers: [
        {provide: InfrastructureService, useValue: infrastructureServiceStub},
        {provide: MatSnackBar, useValue: matSnackBarStub},
        {provide: ActivatedRoute, useValue: activatedRouteStub},
        {provide: MatDialog, useValue: matDialogStub}
      ],
      imports: [
        BrowserAnimationsModule,
        AngularMaterialImporterModule
      ],
    }).compileComponents();
  }));

  it('should create the infrastructure component', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component).toBeTruthy();
  }));

  it('should initialize the infrastructure component with the expected initial values', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component.showLoadInfrastructureError).toEqual(false);
    expect(component.showInfrastructureStatusLoading).toEqual(false);
    expect(component.showInfrastructureNotFound).toEqual(false);
    expect(component.showInfrastructureStatusOk).toEqual(false);
    expect(component.showInfrastructureDeploying).toEqual(false);
    expect(component.showInfrastructureDeleting).toEqual(false);
    expect(component.showInfrastructureFailed).toEqual(false);
    expect(component.showInfrastructureUnknown).toEqual(false);
    expect(component.showCouldNotDetermineInfrastructure).toEqual(false);
    expect(component.createInfrastructureDisabled).toEqual(false);
    expect(component.tearDownDisabled).toEqual(false);
  }));

  it('should show a loading spinner while infrastructure status is loading', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.never());
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('mat-spinner');
      expect(element).not.toBeNull();
    });
  }));

  it('should contain an mat-list', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('mat-list');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the error message div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-error-message');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the error and error message formatted by HttpErrorFormatter', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
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

  it('should display the error message div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-error-message');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the infrastructure ok div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-infrastructure-ok');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the infrastructure not found div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_FOUND));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-infrastructure-not-found');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the infrastructure deploying div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DEPLOYING));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-infrastructure-deploying');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the infrastructure deleting div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DELETING));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-infrastructure-deleting');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the infrastructure failed div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_FAILED));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-infrastructure-failed');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the infrastructure unknown div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_UNKNOWN));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-infrastructure-unknown');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the infrastructure not determined div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_DETERMINED));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-infrastructure-not-determined');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the infrastructure not determined div', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_DETERMINED));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-infrastructure-not-determined');
      expect(element).not.toBeNull();
    });
  }));

  it('should display the create infrastructure and tear down infrastructure buttons', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.querySelector('.ingest-create-infrastructure')).not.toBeNull();
      expect(compiled.querySelector('.ingest-tear-down-infrastructure')).not.toBeNull();
    });
  }));

  it('should call the post create infrastructure method on service', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_FOUND));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-create-infrastructure');
      element.click();
      expect(infrastructureServiceStub.postCreateInfrastructure).toHaveBeenCalled();
    });
  }));

  it('should call the tear down infrastructure method on service', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-tear-down-infrastructure');
      element.click();
      expect(infrastructureServiceStub.postTearDownInfrastructure).toHaveBeenCalled();
    });
  }));

  /**
   * TODO(b/65848519): Instead of spying on IntervalTimer, should use TestScheduler to test
   *     polling.
   */
  it('should get the infrastructure status four times when the infrastructure is deploying', async(() => {
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DEPLOYING));
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    // It should poll the infrastructure three times
    intervalObservableCreateSpy.and.returnValue(Observable.of(1, 2, 3));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      // There should have been four calls: one initial getInfrastructureStatus + 3 polling calls
      expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(4);
    });
  }));

  it('should get the infrastructure status four times when the infrastructure is DELETING', async(() => {
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DELETING));
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    // It should poll the infrastructure three times
    intervalObservableCreateSpy.and.returnValue(Observable.of(1, 2, 3));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      // There should have been four calls: one initial getInfrastructureStatus + 3 polling calls
      expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(4);
    });
  }));

  it('should get the infrastructure once if the infrastructure is not deploying or deleting', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    // It should poll the infrastructure three times
    intervalObservableCreateSpy.and.returnValue(Observable.of(1, 2, 3));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      // There should just be one call, the initial call and no polling calls.
      expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(1);
    });
  }));

  it('should open a snackbar with formatted error title when polling the infrastructure status',
    async(() => {
    const infrastructureStatusObservable = Observable.create((observer) => {
          observer.next(FAKE_INFRA_STATUS_DEPLOYING);
          observer.error(FAKE_HTTP_ERROR);
        });
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(infrastructureStatusObservable);
    spyOn(HttpErrorResponseFormatter, 'getTitle').and.callFake(function(httpError) {
      expect(httpError).toBe(FAKE_HTTP_ERROR);
      return 'fakeFormattedTitle';
    });
    const fixture = TestBed.createComponent(InfrastructureComponent);
    intervalObservableCreateSpy.and.returnValue(Observable.of(1));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      // There should have been two calls, one get infrastructure and one for polling
      expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(2);
      expect(matSnackBarStub.open).toHaveBeenCalled();
      expect(matSnackBarStub.open.calls.first().args[0]).toMatch('fakeFormattedTitle');
    });
  }));

  it('should show the infrastructure as deploying after clicking on infrastructure create', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_FOUND));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DEPLOYING));
      const compiled = fixture.debugElement.nativeElement;
      let element = compiled.querySelector('.ingest-create-infrastructure');
      element.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        fixture.detectChanges();
        element = compiled.querySelector('.ingest-infrastructure-deploying');
        expect(element).not.toBeNull();
      });
    });
  }));

  it('should show the infrastructure as tearing down after clicking on infrastructure tear down', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DELETING));
      const compiled = fixture.debugElement.nativeElement;
      let element = compiled.querySelector('.ingest-tear-down-infrastructure');
      element.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        fixture.detectChanges();
        element = compiled.querySelector('.ingest-infrastructure-deleting');
        expect(element).not.toBeNull();
      });
    });
  }));

  it('should start and stop polling when the infrastructure is deploying or deleting', fakeAsync(() => {
    intervalObservableCreateSpy.and.callThrough(); // Enable polling for this test.
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_FOUND));
    const fixture = TestBed.createComponent(InfrastructureComponent);

    // First load the initial state as 'not found'.
    tick(500);
    fixture.detectChanges();
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(1); // initial call
    const compiled = fixture.debugElement.nativeElement;

    // Then, click on the create infrastructure button. The app should poll the status.
    infrastructureServiceStub.getInfrastructureStatus.calls.reset();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DEPLOYING));
    const createInfrastructureButton = compiled.querySelector('.ingest-create-infrastructure');
    createInfrastructureButton.click();
    tick(30000);
    fixture.detectChanges();
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(3);

    // The app should not poll when it is not deployed.
    infrastructureServiceStub.getInfrastructureStatus.calls.reset();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));
    tick(20000);
    fixture.detectChanges();
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(1);

    // Click on the teardown infrastructure button. The app should poll the status again.
    infrastructureServiceStub.getInfrastructureStatus.calls.reset();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DELETING));
    const tearDownInfrastructureButton = compiled.querySelector('.ingest-tear-down-infrastructure');
    tearDownInfrastructureButton.click();
    tick(30000);
    fixture.detectChanges();
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(3);

    // The app should not poll when the infrastructure status is 'not found' again.
    infrastructureServiceStub.getInfrastructureStatus.calls.reset();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_FOUND));
    tick(20000);
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(1);
    discardPeriodicTasks(); // end of fakeAsync test.
  }));

  it('should not poll if the server has not responded to create infrastructure', fakeAsync(() => {
    intervalObservableCreateSpy.and.callThrough(); // Enable polling for this test.
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_FOUND));
    infrastructureServiceStub.postCreateInfrastructure(Observable.never()); // Never reply.

    // First load the initial state as "not found".
    const fixture = TestBed.createComponent(InfrastructureComponent);
    tick(500);
    fixture.detectChanges();
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(1); // initial call
    // Then, click on the create infrastructure button.
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    const createInfrastructureButton = compiled.querySelector('.ingest-create-infrastructure');
    createInfrastructureButton.click();
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      // The create infrastucture button finished it's logic. Now let time pass.
      infrastructureServiceStub.getInfrastructureStatus.calls.reset();
      tick(90000);
      expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(0);
    });
    discardPeriodicTasks(); // end of fakeAsync test.
  }));

  it('should not poll if the server has not responded to tear down infrastructure', fakeAsync(() => {
    intervalObservableCreateSpy.and.callThrough(); // Enable polling for this test.
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));
    infrastructureServiceStub.postTearDownInfrastructure(Observable.never()); // Never reply.

    // First load the initial state as "running".
    const fixture = TestBed.createComponent(InfrastructureComponent);
    tick(500);
    fixture.detectChanges();
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(1); // initial call
    // Then, click on the tear down infrastructure button.
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    const createInfrastructureButton = compiled.querySelector('.ingest-tear-down-infrastructure');
    createInfrastructureButton.click();
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      // The tear down infrastucture button finished it's logic. Now let time pass.
      infrastructureServiceStub.getInfrastructureStatus.calls.reset();
      tick(90000);
      expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(0);
    });
    discardPeriodicTasks(); // end of fakeAsync test.
  }));

  it('should disable create button when infrastructure is deploying', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DEPLOYING));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const createInfrastructureButton = compiled.querySelector('.ingest-create-infrastructure');
      expect(createInfrastructureButton.hasAttribute('disabled')).toBe(true);
    });
  }));

  it('should disable the create button when infrastructure is tearing down', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DELETING));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const createInfrastructureButton = compiled.querySelector('.ingest-create-infrastructure');
      expect(createInfrastructureButton.hasAttribute('disabled')).toBe(true);
    });
  }));

  it('should disable the tear down button when infrastructure is not found', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_FOUND));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const tearDownInfrastructureButton = compiled.querySelector('.ingest-tear-down-infrastructure');
      expect(tearDownInfrastructureButton.hasAttribute('disabled')).toBe(true);
    });
  }));

  it('should not tear down the infrastructure when the user does not confirm', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    windowConfirmSpy.and.returnValue(false);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const tearDownInfrastructureButton = compiled.querySelector('.ingest-tear-down-infrastructure');
      tearDownInfrastructureButton.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        // Should not show as deleting.
        const element = compiled.querySelector('.ingest-infrastructure-deleting');
        expect(element).toBeNull();
      });
    });
  }));

  it('should display a dialog with the error when teardown infrastructure gives error', async(() => {
    infrastructureServiceStub.postTearDownInfrastructure.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(InfrastructureComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const tearDownInfrastructureButton = compiled.querySelector('.ingest-tear-down-infrastructure');
      tearDownInfrastructureButton.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        fixture.detectChanges();
        expect(matDialogStub.open).toHaveBeenCalled();
        expect(matDialogStub.open.calls.mostRecent().args[0]).toBe(ErrorDialogComponent);
        expect(matDialogStub.open.calls.mostRecent().args[1].data.errorTitle).toBe('FakeError');
        expect(matDialogStub.open.calls.mostRecent().args[1].data.errorMessage).toBe('Fake Error Message.');
      });
    });
  }));

  it('should display a dialog with the error when create infrastructure gives error', async(() => {
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_FOUND));
    infrastructureServiceStub.postCreateInfrastructure.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(InfrastructureComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const createInfrastructureButton = compiled.querySelector('.ingest-create-infrastructure');
      createInfrastructureButton.click();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        fixture.detectChanges();
        expect(matDialogStub.open).toHaveBeenCalled();
        expect(matDialogStub.open.calls.mostRecent().args[0]).toBe(ErrorDialogComponent);
        expect(matDialogStub.open.calls.mostRecent().args[1].data.errorTitle).toBe('FakeError');
        expect(matDialogStub.open.calls.mostRecent().args[1].data.errorMessage).toBe('Fake Error Message.');
      });
    });
  }));


});
