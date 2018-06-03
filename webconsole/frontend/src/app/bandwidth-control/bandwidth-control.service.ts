import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs';
import { map, switchMap } from 'rxjs/operators';

import { environment } from '../../environments/environment';
import { MaxBandwidthResponse } from './bandwidth-control.resources';

const POST_HEADERS = {
    headers: new HttpHeaders().set('Content-Type', 'application/json')
};

@Injectable()
export class BandwidthControlService {
  private project: Observable<string>;

  constructor(private http: HttpClient, private route: ActivatedRoute) {
    this.project = route.queryParams.pipe(map(p => p.project));
  }

  getProjectMaxBandwidth(): Observable<MaxBandwidthResponse> {
    return this.project.pipe(switchMap(projectId => {
      return this.http.get<MaxBandwidthResponse>(
          `${environment.apiUrl}/v1/projects`,
          {params: {projectId: projectId}});
    }));
  }

  postProjectMaxBandwidth(hasMaxBandwidth: boolean, bandwidth: number): Observable<MaxBandwidthResponse> {
    return this.project.pipe(switchMap(projectId => {
        return this.http.post<MaxBandwidthResponse>(
            `${environment.apiUrl}/v1/projects`,
            {
              projectId: projectId,
              hasMaxBandwidth: hasMaxBandwidth,
              bandwidth: bandwidth,
            }, POST_HEADERS);
    }));
  }
}
