import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Response } from '@angular/http';
import { ActivatedRoute } from '@angular/router';
import { combineLatest, Observable } from 'rxjs';
import { map, switchMap, take } from 'rxjs/operators';

import { environment } from '../../environments/environment';
import {
  Agent,
  AgentResponse,
} from './pulse.resources';

const POST_HEADERS = {
    headers: new HttpHeaders().set('Content-Type', 'application/json')
};

@Injectable()
export class PulseService {
  private project: Observable<string>;

  constructor(private http: HttpClient, private route: ActivatedRoute) {
    this.project = route.queryParams.pipe(map(p => p.project));
  }

  /**
   * Get a list of agents.
   */
  getAgents(): Observable<AgentResponse> {
    return this.project.pipe(switchMap(projectId => {
      // Query all Agents.
      return this.http.get<AgentResponse>(
        `${environment.apiUrl}/v1/projects/${projectId}/agents`);
     }));
  }
}
