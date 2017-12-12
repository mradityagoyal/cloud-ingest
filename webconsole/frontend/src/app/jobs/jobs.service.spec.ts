import { FAKE_TASKS } from './jobs.test-util';
import { JobsService } from './jobs.service';
import { TestBed, async, inject } from '@angular/core/testing';
import { environment } from '../../environments/environment';
import { HttpClient } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { Task } from './jobs.resources';
import { TaskFailureType, TaskStatus } from '../proto/tasks.js';
import 'rxjs/add/observable/of';

let activatedRouteStub: ActivatedRoute;

const FAKE_JOBCONFIG1 = 'fakeJobConfigId1';
const FAKE_JOBRUN1 = 'fakeJobRunId1';
const FAKE_LAST_MODIFIED_TIME = '1';

describe('JobsService', () => {
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

  it('should request for the get tasks of status url once',
  inject([HttpClient, HttpTestingController], (http: HttpClient, httpMock: HttpTestingController) => {
  const jobsService = new JobsService(http, activatedRouteStub);
  let actualTasks: Task[];
  jobsService.getTasksOfStatus(FAKE_JOBCONFIG1, TaskStatus.Type.SUCCESS).subscribe(
    (response) => {
      actualTasks = response;
    },
    (error) => {
      // should not be called
    });
  httpMock.expectOne(`${environment.apiUrl}/projects/fakeProjectId/tasks/fakeJobConfigId1/status/${TaskStatus.Type.SUCCESS}`)
      .flush(FAKE_TASKS);
  expect(actualTasks).toEqual(FAKE_TASKS);
  httpMock.verify();
}));

 it('should request for the get tasks of status url once with last modified',
    inject([HttpClient, HttpTestingController], (http: HttpClient, httpMock: HttpTestingController) => {
    const jobsService = new JobsService(http, activatedRouteStub);
    let actualTasks: Task[];
    jobsService.getTasksOfStatus(FAKE_JOBCONFIG1, TaskStatus.Type.SUCCESS, FAKE_LAST_MODIFIED_TIME).subscribe(
      (response) => {
        actualTasks = response;
      },
      (error) => {
        // should not be called
      });
    httpMock.expectOne(`${environment.apiUrl}/projects/fakeProjectId/tasks/fakeJobConfigId1/status/${TaskStatus.Type.SUCCESS}` +
                       `?lastModifiedBefore=${FAKE_LAST_MODIFIED_TIME}`)
        .flush(FAKE_TASKS);
    expect(actualTasks).toEqual(FAKE_TASKS);
    httpMock.verify();
  }));

});

