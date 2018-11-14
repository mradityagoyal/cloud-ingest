import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit } from '@angular/core';
import { Response } from '@angular/http';
import { MatDialog } from '@angular/material';
import { Observable, of } from 'rxjs';
import { interval } from 'rxjs';
import { takeWhile } from 'rxjs/operators';

import { HttpErrorResponseFormatter } from '../util/error.resources';
import { AgentDataSource, Agent, AgentResponse } from './pulse.resources';
import { PulseService } from './pulse.service';

const UPDATE_ACTIVE_AGENTS_POLLING_INTERVAL_MILLISECONDS = 6000;

@Component({
  selector: 'app-pulse',
  templateUrl: './pulse.component.html',
  styleUrls: ['./pulse.component.css']
})
export class PulseComponent implements OnInit {
  agent: Agent;
  agentResponse: AgentResponse;
  showLoadingSpinner = true;
  errorMessage: string;
  errorTitle: string;
  displayErrorMessage = false;

  /**
   * If the number of requested agents is more than 0 this boolean is set to true,
   * otherwise, false.
   */
  hasAgents = false;

  agentsDisplayedColumns = ['agentID', 'pulseReceived'];

  activeAgentDataSource: AgentDataSource<Agent>;

  constructor(
    private readonly pulseService: PulseService,
    public dialog: MatDialog,
  ) { }

  ngOnInit() {
    this.showLoadingSpinner = true;
    this.initialAgentLoad();
  }

  initialAgentLoad(): void {
    this.pulseService.getAgents().subscribe(
    (response: AgentResponse) => {
      if (!response.agents) {
        this.hasAgents = false;
      } else {
        this.hasAgents = true;
        this.activeAgentDataSource = new AgentDataSource(response.agents);
      }
      this.showLoadingSpinner = false;
    }, (error: HttpErrorResponse) => {
      this.errorTitle = HttpErrorResponseFormatter.getTitle(error);
      this.errorMessage = HttpErrorResponseFormatter.getMessage(error);
    this.displayErrorMessage = true;
    this.showLoadingSpinner = false;
    });
  }

  updateAgents(): void {
    this.pulseService.getAgents().subscribe(
    (response: AgentResponse) => {
      if (!response.agents) {
        this.hasAgents = false;
      } else {
        this.hasAgents = true;
        this.activeAgentDataSource = new AgentDataSource(response.agents);
      }
      this.showLoadingSpinner = false;
    },
    (error: HttpErrorResponse) => {
      this.errorTitle = HttpErrorResponseFormatter.getTitle(error);
      this.errorMessage = HttpErrorResponseFormatter.getMessage(error);
      this.displayErrorMessage = true;
      this.showLoadingSpinner = false;
    });
  }
}
