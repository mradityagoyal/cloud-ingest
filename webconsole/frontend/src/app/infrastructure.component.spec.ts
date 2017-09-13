import { TestBed, async } from '@angular/core/testing';
import { InfrastructureService, INFRA_STATUS } from './infrastructure.service';
import { InfrastructureStatus, PubsubStatus} from './api.resources';
import { InfrastructureComponent } from './infrastructure.component';
import { AngularMaterialImporterModule } from './angular-material-importer.module';
import { InfrastructureStatusItemComponent } from './infrastructure-status-item.component';
import { Observable } from 'rxjs/Observable';
import { HttpErrorResponse } from '@angular/common/http';
import 'rxjs/add/observable/throw';
import 'rxjs/add/observable/of';
import 'rxjs/add/observable/never';


class InfrastructureServiceStub {
  public getInfrastructureStatus = jasmine.createSpy('getInfrastructureStatus');
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

let infrastructureServiceStub: InfrastructureServiceStub;

describe('InfrastructureComponent', () => {

  beforeEach(async(() => {
    infrastructureServiceStub = new InfrastructureServiceStub();
    infrastructureServiceStub.getInfrastructureStatus.and.returnValue(Observable.of(FAKE_INFRA_STATUS_RUNNING));

    TestBed.configureTestingModule({
      declarations: [
        InfrastructureComponent,
        InfrastructureStatusItemComponent
      ],
      providers: [
        {provide: InfrastructureService, useValue: infrastructureServiceStub},
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
    expect(component.showUpdateInfrastructureError).toEqual(false);
    expect(component.showUpdateInfrastructureLoading).toEqual(false);
    expect(component.showInfrastructureNotFound).toEqual(false);
    expect(component.showInfrastructureStatusOk).toEqual(false);
    expect(component.showInfrastructureDeploying).toEqual(false);
    expect(component.showInfrastructureDeleting).toEqual(false);
    expect(component.showInfrastructureFailed).toEqual(false);
    expect(component.showInfrastructureUnknown).toEqual(false);
    expect(component.showCouldNotDetermineInfrastructure).toEqual(false);
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

});
