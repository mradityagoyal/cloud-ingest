import { InfrastructureService, INFRA_STATUS } from './infrastructure.service';
import { TestBed, async, inject } from '@angular/core/testing';
import { environment } from '../../environments/environment';
import { HttpClient } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { InfrastructureStatus, PubsubStatus } from './infrastructure.resources';
import 'rxjs/add/observable/of';

let activatedRouteStub: ActivatedRoute;

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
 uploadGCS: INFRA_STATUS.FAILED,
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

describe('InfrastructureService', () => {
  beforeEach(() => {

    activatedRouteStub = new ActivatedRoute();
    activatedRouteStub.queryParams = Observable.of({project: 'fakeProjectId'});

    TestBed.configureTestingModule({
      providers: [
        {provide: ActivatedRoute, useValue: activatedRouteStub},
      ],
      imports: [HttpClientTestingModule],
     });
  });

  it('should request for the get infrastructure url once',
    inject([HttpClient, HttpTestingController], (http: HttpClient, httpMock: HttpTestingController) => {
    const infrastructureService = new InfrastructureService(http, activatedRouteStub);
    let actualInfrastructureStatus: InfrastructureStatus;
    infrastructureService.getInfrastructureStatus().subscribe(
      (response) => {
        actualInfrastructureStatus = response;
      },
      (error) => {
        // should not be called
      });
    httpMock.expectOne(`${environment.apiUrl}/projects/fakeProjectId/infrastructure-status`).flush(FAKE_INFRA_STATUS_RUNNING);
    expect(actualInfrastructureStatus).toEqual(FAKE_INFRA_STATUS_RUNNING);
    httpMock.verify();
  }));

  it('should post to the create infrastructure url once',
    inject([HttpClient, HttpTestingController], (http: HttpClient, httpMock: HttpTestingController) => {
    const infrastructureService = new InfrastructureService(http, activatedRouteStub);
    let actualResponse: any;
    infrastructureService.postCreateInfrastructure().subscribe(
      (response) => {
        actualResponse = response;
      },
      (error) => {
        // should not be called
      });
    httpMock.expectOne(`${environment.apiUrl}/projects/fakeProjectId/create-infrastructure`).flush({});
    expect(actualResponse).toEqual({});
    httpMock.verify();
  }));

  it('should post to the tear down infrastructure url once',
    inject([HttpClient, HttpTestingController], (http: HttpClient, httpMock: HttpTestingController) => {
    const infrastructureService = new InfrastructureService(http, activatedRouteStub);
    let actualResponse: any;
    infrastructureService.postTearDownInfrastructure().subscribe(
      (response) => {
        actualResponse = response;
      },
      (error) => {
        // should not be called
      });
    httpMock.expectOne(`${environment.apiUrl}/projects/fakeProjectId/tear-down-infrastructure`).flush({});
    expect(actualResponse).toEqual({});
    httpMock.verify();
  }));

  it('isInfrastructureOk should return true', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureOk(FAKE_INFRA_STATUS_RUNNING)).toEqual(true);
  }));

  it('isInfrastructureOk should return false', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureOk(FAKE_INFRA_STATUS_NOT_FOUND)).toEqual(false);
    expect(InfrastructureService.isInfrastructureOk(FAKE_INFRA_STATUS_UNKNOWN)).toEqual(false);
    expect(InfrastructureService.isInfrastructureOk(FAKE_INFRA_STATUS_FAILED)).toEqual(false);
    expect(InfrastructureService.isInfrastructureOk(FAKE_INFRA_STATUS_DEPLOYING)).toEqual(false);
    expect(InfrastructureService.isInfrastructureOk(FAKE_INFRA_STATUS_DELETING)).toEqual(false);
  }));

  it('isInfrastructureNotFound should return true ', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureNotFound(FAKE_INFRA_STATUS_NOT_FOUND)).toEqual(true);
  }));

  it('isInfrastructureNotFound should return false ', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureNotFound(FAKE_INFRA_STATUS_RUNNING)).toEqual(false);
    expect(InfrastructureService.isInfrastructureNotFound(FAKE_INFRA_STATUS_UNKNOWN)).toEqual(false);
    expect(InfrastructureService.isInfrastructureNotFound(FAKE_INFRA_STATUS_FAILED)).toEqual(false);
    expect(InfrastructureService.isInfrastructureNotFound(FAKE_INFRA_STATUS_DEPLOYING)).toEqual(false);
    expect(InfrastructureService.isInfrastructureNotFound(FAKE_INFRA_STATUS_DELETING)).toEqual(false);
  }));

  it('isInfrastructureDeploying should return true ', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureDeploying(FAKE_INFRA_STATUS_DEPLOYING)).toEqual(true);
  }));

  it('isInfrastructureDeploying should return false ', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureDeploying(FAKE_INFRA_STATUS_RUNNING)).toEqual(false);
    expect(InfrastructureService.isInfrastructureDeploying(FAKE_INFRA_STATUS_UNKNOWN)).toEqual(false);
    expect(InfrastructureService.isInfrastructureDeploying(FAKE_INFRA_STATUS_FAILED)).toEqual(false);
    expect(InfrastructureService.isInfrastructureDeploying(FAKE_INFRA_STATUS_NOT_FOUND)).toEqual(false);
    expect(InfrastructureService.isInfrastructureDeploying(FAKE_INFRA_STATUS_DELETING)).toEqual(false);
  }));

  it('isInfrastructureDeleting should return true ', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureDeleting(FAKE_INFRA_STATUS_DELETING)).toEqual(true);
  }));

  it('isInfrastructureDeleting should return false ', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureDeleting(FAKE_INFRA_STATUS_RUNNING)).toEqual(false);
    expect(InfrastructureService.isInfrastructureDeleting(FAKE_INFRA_STATUS_UNKNOWN)).toEqual(false);
    expect(InfrastructureService.isInfrastructureDeleting(FAKE_INFRA_STATUS_FAILED)).toEqual(false);
    expect(InfrastructureService.isInfrastructureDeleting(FAKE_INFRA_STATUS_NOT_FOUND)).toEqual(false);
    expect(InfrastructureService.isInfrastructureDeleting(FAKE_INFRA_STATUS_DEPLOYING)).toEqual(false);
  }));

  it('isInfrastructureFailed should return true', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureFailed(FAKE_INFRA_STATUS_FAILED)).toEqual(true);
  }));

  it('isInfrastructureFailed should return false', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureFailed(FAKE_INFRA_STATUS_NOT_FOUND)).toEqual(false);
    expect(InfrastructureService.isInfrastructureFailed(FAKE_INFRA_STATUS_RUNNING)).toEqual(false);
    expect(InfrastructureService.isInfrastructureFailed(FAKE_INFRA_STATUS_DEPLOYING)).toEqual(false);
    expect(InfrastructureService.isInfrastructureFailed(FAKE_INFRA_STATUS_DELETING)).toEqual(false);
  }));

  it('isInfrastructureUnknown should return true', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureUnknown(FAKE_INFRA_STATUS_UNKNOWN)).toEqual(true);
  }));

  it('isInfrastructureUnknown should return false', inject([HttpClient, HttpTestingController],
    (http: HttpClient, httpMock: HttpTestingController) => {
    expect(InfrastructureService.isInfrastructureUnknown(FAKE_INFRA_STATUS_NOT_FOUND)).toEqual(false);
    expect(InfrastructureService.isInfrastructureUnknown(FAKE_INFRA_STATUS_RUNNING)).toEqual(false);
    expect(InfrastructureService.isInfrastructureUnknown(FAKE_INFRA_STATUS_DEPLOYING)).toEqual(false);
    expect(InfrastructureService.isInfrastructureUnknown(FAKE_INFRA_STATUS_DELETING)).toEqual(false);
  }));

});

