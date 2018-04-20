import 'rxjs/add/operator/switchMap';
import { catchError, map, tap } from 'rxjs/operators';

import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';

import { environment } from '../../environments/environment';
import { BandwidthControl, MaxBandwidthResponse, MaxBandwidthRequest } from './bandwidth-control.resources';

const POST_HEADERS = {
    headers: new HttpHeaders().set('Content-Type', 'application/json')
};

@Injectable()
export class BandwidthControlService {
  private project: Observable<string>;

  constructor(private http: HttpClient, private route: ActivatedRoute) {
    this.project = route.queryParams.map(p => p.project);
  }

  getProjectMaxBandwidth(): Observable<MaxBandwidthResponse> {
    return this.project.switchMap(projectId => {
      return this.http.get<MaxBandwidthResponse>(
          `${environment.apiUrl}/v1/projects`,
          {params: {projectId: projectId}});
    });
  }

  postProjectMaxBandwidth(hasMaxBandwidth: boolean, bandwidth: number): Observable<MaxBandwidthResponse> {
    return this.project.switchMap(projectId => {
        return this.http.post<MaxBandwidthResponse>(
            `${environment.apiUrl}/v1/projects`,
            {
              projectId: projectId,
              hasMaxBandwidth: hasMaxBandwidth,
              bandwidth: bandwidth,
            }, POST_HEADERS);
    });
  }
}
