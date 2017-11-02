import 'rxjs/add/observable/of';

import { HttpClient } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { inject, TestBed } from '@angular/core/testing';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';

import { environment } from '../../environments/environment';
import { INFRA_STATUS, InfrastructureStatus, PubsubStatus } from './infrastructure.resources';
import { InfrastructureService } from './infrastructure.service';

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
  dcpStatus: INFRA_STATUS.RUNNING
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
  dcpStatus: INFRA_STATUS.NOT_FOUND
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
  dcpStatus: INFRA_STATUS.UNKNOWN
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
  dcpStatus: INFRA_STATUS.FAILED
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
  dcpStatus: INFRA_STATUS.NOT_FOUND
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
  dcpStatus: INFRA_STATUS.RUNNING
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

});

