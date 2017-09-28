import { TestBed, async, fakeAsync, tick, discardPeriodicTasks } from '@angular/core/testing';
import { InfrastructureService, INFRA_STATUS } from './infrastructure.service';
import { InfrastructureStatus, PubsubStatus} from './infrastructure.resources';
import { InfrastructureComponent } from './infrastructure.component';
import { AngularMaterialImporterModule } from '../angular-material-importer/angular-material-importer.module';
import { InfrastructureStatusItemComponent } from './infrastructure-status-item/infrastructure-status-item.component';
import { Observable } from 'rxjs/Observable';
import { IntervalObservable } from 'rxjs/observable/IntervalObservable';
import { HttpErrorResponse } from '@angular/common/http';
import { ActivatedRoute } from '@angular/router';
import { MdSnackBar } from '@angular/material';
import 'rxjs/add/observable/throw';
import 'rxjs/add/observable/of';
import 'rxjs/add/observable/never';


class InfrastructureServiceStub {
  public getInfrastructureStatus = jasmine.createSpy('getInfrastructureStatus');
  public postCreateInfrastructure = jasmine.createSpy('postCreateInfrastructure');
  public postTearDownInfrastructure = jasmine.createSpy('postTearDownInfrastructure');
}

class ActivatedRouteStub {
  snapshot = {
    queryParams: {
      project: 'fakeProjectId'
    }
  };
}

class MdSnackBarStub {
  open = jasmine.createSpy('open');
}

const FAKE_PUBSUB_STATUS_RUNNING: PubsubStatus = {
 list:  INFRA_STATUS.RUNNING,
 listProgress: INFRA_STATUS.RUNNING,
 uploadGCS: INFRA_STATUS.RUNNING,
 uploadGCSProgress: INFRA_STATUS.RUNNING,
 loadBigQuery : INFRA_STATUS.RUNNING,
 loadBigQueryProgress : INFRA_STATUS.RUNNING
};

const FAKE_INFRA_STATUS_RUNNING: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_RUNNING,
  dcpStatus: INFRA_STATUS.RUNNING,
  cloudFunctionsStatus: INFRA_STATUS.RUNNING
};

const FAKE_PUBSUB_STATUS_NOT_FOUND: PubsubStatus = {
 list:  INFRA_STATUS.NOT_FOUND,
 listProgress: INFRA_STATUS.NOT_FOUND,
 uploadGCS: INFRA_STATUS.NOT_FOUND,
 uploadGCSProgress: INFRA_STATUS.NOT_FOUND,
 loadBigQuery : INFRA_STATUS.NOT_FOUND,
 loadBigQueryProgress : INFRA_STATUS.NOT_FOUND
};

const FAKE_INFRA_STATUS_NOT_FOUND: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.NOT_FOUND,
  pubsubStatus: FAKE_PUBSUB_STATUS_NOT_FOUND,
  dcpStatus: INFRA_STATUS.NOT_FOUND,
  cloudFunctionsStatus: INFRA_STATUS.NOT_FOUND
};

const FAKE_PUBSUB_STATUS_UNKNOWN: PubsubStatus = {
 list:  INFRA_STATUS.RUNNING,
 listProgress: INFRA_STATUS.RUNNING,
 uploadGCS: INFRA_STATUS.RUNNING,
 uploadGCSProgress: INFRA_STATUS.RUNNING,
 loadBigQuery : INFRA_STATUS.RUNNING,
 loadBigQueryProgress : INFRA_STATUS.UNKNOWN
};

const FAKE_INFRA_STATUS_UNKNOWN: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_UNKNOWN,
  dcpStatus: INFRA_STATUS.UNKNOWN,
  cloudFunctionsStatus: INFRA_STATUS.RUNNING
};

const FAKE_PUBSUB_STATUS_FAILED: PubsubStatus = {
 list:  INFRA_STATUS.RUNNING,
 listProgress: INFRA_STATUS.FAILED,
 uploadGCS: INFRA_STATUS.RUNNING,
 uploadGCSProgress: INFRA_STATUS.RUNNING,
 loadBigQuery : INFRA_STATUS.RUNNING,
 loadBigQueryProgress : INFRA_STATUS.RUNNING
};

const FAKE_INFRA_STATUS_FAILED: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_UNKNOWN,
  dcpStatus: INFRA_STATUS.FAILED,
  cloudFunctionsStatus: INFRA_STATUS.RUNNING
};

const FAKE_PUBSUB_STATUS_DEPLOYING: PubsubStatus = {
 list:  INFRA_STATUS.DEPLOYING,
 listProgress: INFRA_STATUS.NOT_FOUND,
 uploadGCS: INFRA_STATUS.DEPLOYING,
 uploadGCSProgress: INFRA_STATUS.DEPLOYING,
 loadBigQuery : INFRA_STATUS.DEPLOYING,
 loadBigQueryProgress : INFRA_STATUS.NOT_FOUND
};

const FAKE_INFRA_STATUS_DEPLOYING: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.NOT_FOUND,
  pubsubStatus: FAKE_PUBSUB_STATUS_DEPLOYING,
  dcpStatus: INFRA_STATUS.NOT_FOUND,
  cloudFunctionsStatus: INFRA_STATUS.DEPLOYING
};

const FAKE_PUBSUB_STATUS_DELETING: PubsubStatus = {
 list:  INFRA_STATUS.DELETING,
 listProgress: INFRA_STATUS.RUNNING,
 uploadGCS: INFRA_STATUS.DELETING,
 uploadGCSProgress: INFRA_STATUS.DELETING,
 loadBigQuery : INFRA_STATUS.DELETING,
 loadBigQueryProgress : INFRA_STATUS.RUNNING
};

const FAKE_INFRA_STATUS_DELETING: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_DELETING,
  dcpStatus: INFRA_STATUS.RUNNING,
  cloudFunctionsStatus: INFRA_STATUS.DELETING
};

const FAKE_PUBSUB_STATUS_NOT_DETERMINED: PubsubStatus = {
 list:  INFRA_STATUS.DEPLOYING,
 listProgress: INFRA_STATUS.RUNNING,
 uploadGCS: INFRA_STATUS.RUNNING,
 uploadGCSProgress: INFRA_STATUS.RUNNING,
 loadBigQuery : INFRA_STATUS.RUNNING,
 loadBigQueryProgress : INFRA_STATUS.DELETING
};

const FAKE_INFRA_STATUS_NOT_DETERMINED: InfrastructureStatus = {
  spannerStatus: INFRA_STATUS.RUNNING,
  pubsubStatus: FAKE_PUBSUB_STATUS_NOT_DETERMINED,
  dcpStatus: INFRA_STATUS.DELETING,
  cloudFunctionsStatus: INFRA_STATUS.DELETING
};

const FAKE_HTTP_ERROR = {error: 'FakeError', message: 'Fake Error Message.'};

let intervalObservableCreateSpy: any;

let infrastructureServiceStub: InfrastructureServiceStub;
let mdSnackBarStub: MdSnackBarStub;
let activatedRouteStub: ActivatedRouteStub;

describe('InfrastructureComponent', () => {

  beforeEach(async(() => {
    infrastructureServiceStub = new InfrastructureServiceStub();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));
    infrastructureServiceStub.postCreateInfrastructure.and.returnValue(Observable.of({}));
    infrastructureServiceStub.postTearDownInfrastructure.and.returnValue(Observable.of({}));
    // Disable polling for most tests
    intervalObservableCreateSpy = spyOn(IntervalObservable, 'create').and.returnValue(Observable.never());
    mdSnackBarStub = new MdSnackBarStub();
    activatedRouteStub = new ActivatedRouteStub();

    TestBed.configureTestingModule({
      declarations: [
        InfrastructureComponent,
        InfrastructureStatusItemComponent
      ],
      providers: [
        {provide: InfrastructureService, useValue: infrastructureServiceStub},
        {provide: MdSnackBar, useValue: mdSnackBarStub},
        {provide: ActivatedRoute, useValue: activatedRouteStub}
      ],
      imports: [
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
      const element = compiled.querySelector('md-spinner');
      expect(element).not.toBeNull();
    });
  }));

  it('should contain an md-list', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('md-list');
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

  it('should display the error and error message', async(() => {
    const fixture = TestBed.createComponent(InfrastructureComponent);
    const component = fixture.debugElement.componentInstance;
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.throw(FAKE_HTTP_ERROR));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-error-message');
      expect(element.textContent).toContain('FakeError');
      expect(element.textContent).toContain('Fake Error Message.');
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

  it('should open a snackbar with the error when polling the infrastructure status', async(() => {
    const infrastructureStatusObservable = Observable.create((observer) => {
          observer.next(FAKE_INFRA_STATUS_DEPLOYING);
          observer.error(FAKE_HTTP_ERROR);
        });
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(infrastructureStatusObservable);
    const fixture = TestBed.createComponent(InfrastructureComponent);
    intervalObservableCreateSpy.and.returnValue(Observable.of(1));
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      // There should have been two calls, one get infrastructure and one for polling
      expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(2);
      expect(mdSnackBarStub.open).toHaveBeenCalled();
      expect(mdSnackBarStub.open.calls.first().args[0]).toMatch('Fake Error Message.');
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
    tick(9000);
    fixture.detectChanges();
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(4);

    // The app should not poll when it is not deployed.
    infrastructureServiceStub.getInfrastructureStatus.calls.reset();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));
    tick(6000);
    fixture.detectChanges();
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(1);

    // Click on the teardown infrastructure button. The app should poll the status again.
    infrastructureServiceStub.getInfrastructureStatus.calls.reset();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_DELETING));
    const tearDownInfrastructureButton = compiled.querySelector('.ingest-tear-down-infrastructure');
    tearDownInfrastructureButton.click();
    tick(9000);
    fixture.detectChanges();
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(4);

    // The app should not poll when the infrastructure status is 'not found' again.
    infrastructureServiceStub.getInfrastructureStatus.calls.reset();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_NOT_FOUND));
    tick(6000);
    expect(infrastructureServiceStub.getInfrastructureStatus.calls.count()).toEqual(1);
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

});
