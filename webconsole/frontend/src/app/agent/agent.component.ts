import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { environment } from '../../environments/environment';

@Component({
  selector: 'app-agent',
  templateUrl: './agent.component.html'
})
export class AgentComponent implements OnInit {
  // Location of the public agent release we are exposing.
  AGENT_RELEASE_PREFIX = 'https://storage.googleapis.com/cloud-ingest-pub/agent/current';
  PUBSUB_PREFIX = environment.pubSubPrefix;

  projectId: string;

  constructor(private readonly route: ActivatedRoute) {
    this.projectId = route.snapshot.queryParams.project;
  }

  ngOnInit() {}

}
