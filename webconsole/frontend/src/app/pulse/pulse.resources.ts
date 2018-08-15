import { DataSource } from '@angular/cdk/collections';
import { Observable, of } from 'rxjs';


export class GcsData {
  constructor() {
    this.bucketName = '';
    this.objectPrefix = '';
  }
  bucketName: string;
  objectPrefix: string;
}

export class Agent {
  agentId: string;
  lastPulseReceived: string;
}

export interface AgentResponse {
  agents: Agent[];
}

export class Agents {
  agents: Agent[];
}

export class AgentDataSource<A> extends DataSource<A> {
  items: A[];

  constructor(items: A[]) {
    super();
    this.items = items;
  }

  connect(): Observable<A[]> {
    return of(this.items);
  }

  disconnect() {}
}
