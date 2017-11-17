import 'rxjs/add/observable/of';

import { HttpClient } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { inject, TestBed } from '@angular/core/testing';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';

import { environment } from '../../environments/environment';
import { INFRA_STATUS, InfrastructureStatus, PubsubStatus } from './infrastructure.resources';
import { InfrastructureService } from './infrastructure.service';
import {
  FAKE_INFRA_STATUS_RUNNING,
  FAKE_INFRA_STATUS_NOT_FOUND,
  FAKE_INFRA_STATUS_UNKNOWN,
  FAKE_INFRA_STATUS_FAILED,
  FAKE_INFRA_STATUS_DEPLOYING,
  FAKE_INFRA_STATUS_DELETING,
  FAKE_INFRA_STATUS_NOT_DETERMINED
} from './infrastructure.test-util';

let activatedRouteStub: ActivatedRoute;

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

