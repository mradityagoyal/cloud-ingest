import { FAKE_TASKS } from './jobs.test-util';
import { JobsService } from './jobs.service';
import { TestBed, async, inject } from '@angular/core/testing';
import { environment } from '../../environments/environment';
import { HttpClient } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { Task, TASK_STATUS } from './jobs.resources';
import { TaskFailureType } from '../proto/tasks.js';
import 'rxjs/add/observable/of';

let activatedRouteStub: ActivatedRoute;

const FAKE_JOBCONFIG1 = 'fakeJobConfigId1';
const FAKE_JOBRUN1 = 'fakeJobRunId1';

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
    jobsService.getTasksOfStatus(FAKE_JOBCONFIG1, FAKE_JOBRUN1, TASK_STATUS.SUCCESS).subscribe(
      (response) => {
        actualTasks = response;
      },
      (error) => {
        // should not be called
      });
    httpMock.expectOne(`${environment.apiUrl}/projects/fakeProjectId/tasks/fakeJobConfigId1/fakeJobRunId1/status/${TASK_STATUS.SUCCESS}`)
        .flush(FAKE_TASKS);
    expect(actualTasks).toEqual(FAKE_TASKS);
    httpMock.verify();
  }));

});

