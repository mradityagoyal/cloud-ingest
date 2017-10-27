import { Component, OnInit } from '@angular/core';
import { InfrastructureStatus } from './infrastructure.resources';
import { InfrastructureService, INFRA_STATUS } from './infrastructure.service';
import { HttpErrorResponse } from '@angular/common/http';
import { MatSnackBar } from '@angular/material';
import { ActivatedRoute } from '@angular/router';
import { IntervalObservable } from 'rxjs/observable/IntervalObservable';
import { HttpErrorResponseFormatter } from '../util/error.resources';

const UPDATE_STATUS_POLLING_INTERVAL_MILISECONDS = 3000;

@Component({
  selector: 'app-infrastructure',
  templateUrl: './infrastructure.component.html',
  styleUrls: ['./infrastructure.component.css'],
})

export class InfrastructureComponent implements OnInit {

  infrastructureStatus: InfrastructureStatus;
  showLoadInfrastructureError = false;
  showInfrastructureStatusLoading = false;
  showInfrastructureNotFound = false;
  showInfrastructureStatusOk = false;
  showInfrastructureDeploying = false;
  showInfrastructureDeleting = false;
  showInfrastructureFailed = false;
  showInfrastructureUnknown = false;
  showCouldNotDetermineInfrastructure = false;
  createInfrastructureDisabled = false;
  tearDownDisabled = false;
  loadInfrastructureErrorTitle: string;
  loadInfrastructureErrorMessage: string;
  projectId: string;

  constructor(private readonly infrastructureService: InfrastructureService,
              private readonly snackBar: MatSnackBar,
              private readonly route: ActivatedRoute) {
    this.projectId = route.snapshot.queryParams.project;
  }

  ngOnInit() {
    this.loadInfrastructureStatus();
    IntervalObservable.create(UPDATE_STATUS_POLLING_INTERVAL_MILISECONDS)
      .subscribe(() => {
        if (this.showInfrastructureDeploying || this.showInfrastructureDeleting) {
          this.pollInfrastructureStatus();
        }
      });
  }

  /**
   * Loads the infrastructure status initially.
   */
  private loadInfrastructureStatus() {
    this.showInfrastructureStatusLoading = true;
    this.infrastructureService.getInfrastructureStatus().subscribe(
      (response: InfrastructureStatus) => {
        this.infrastructureStatus = response;
        this.updateInfrastructureStatusMessage(response);
        this.updateCreateTearDownButtons(response);
        this.showLoadInfrastructureError = false;
        this.showInfrastructureStatusLoading = false;
      },
      (errorResponse: HttpErrorResponse) => {
        this.loadInfrastructureErrorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
        this.loadInfrastructureErrorMessage = HttpErrorResponseFormatter.getMessage(errorResponse);
        this.showLoadInfrastructureError = true;
        this.showInfrastructureStatusLoading = false;
      }
    );
  }

  /**
   * Polls the infrastructure status in intervals.
   */
  private pollInfrastructureStatus() {
    this.infrastructureService.getInfrastructureStatus().subscribe(
      (response: InfrastructureStatus) => {
        this.infrastructureStatus = response;
        this.updateInfrastructureStatusMessage(response);
        this.updateCreateTearDownButtons(response);
      },
      (errorResponse: HttpErrorResponse) => {
        const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
        console.error(errorTitle + '\m' + HttpErrorResponseFormatter.getMessage(errorResponse));
        this.snackBar.open(`There was an error polling the infrastructure status: ${errorTitle}`, 'Dismiss');
      }
    );
  }

  private updateInfrastructureStatusMessage(response) {
    this.resetMessageVariables();

    if (InfrastructureService.isInfrastructureOk(response)) {
      this.showInfrastructureStatusOk = true;
    } else if (InfrastructureService.isInfrastructureNotFound(response)) {
      this.showInfrastructureNotFound = true;
    } else if (InfrastructureService.isInfrastructureDeploying(response)) {
      this.showInfrastructureDeploying = true;
    } else if (InfrastructureService.isInfrastructureDeleting(response)) {
      this.showInfrastructureDeleting = true;
    } else if (InfrastructureService.isInfrastructureFailed(response)) {
      this.showInfrastructureFailed = true;
    } else if (InfrastructureService.isInfrastructureUnknown(response)) {
      this.showInfrastructureUnknown = true;
    } else {
      this.showCouldNotDetermineInfrastructure = true;
    }
  }

  private resetMessageVariables() {
    this.showInfrastructureStatusOk = this.showInfrastructureNotFound =
    this.showInfrastructureDeploying = this.showInfrastructureDeleting =
    this.showInfrastructureFailed = this.showInfrastructureUnknown =
    this.showCouldNotDetermineInfrastructure = false;
  }

  private showInfrastructureDeployingMessage() {
    this.resetMessageVariables();
    this.showInfrastructureDeploying = true;
  }

  private showInfrastructureDeletingMessage() {
    this.resetMessageVariables();
    this.showInfrastructureDeleting = true;
  }

  private updateCreateTearDownButtons(response) {
    if (InfrastructureService.isInfrastructureNotFound(response)) {
      this.createInfrastructureDisabled = false;
      this.tearDownDisabled = true;
    } else {
      this.createInfrastructureDisabled = true;
      this.tearDownDisabled = false;
    }
  }

  /**
   * TODO(b/65954031): Revise the mechanism of getting the infrastructure status right after
   *     requesting the backend to create the infrastructure.
   */
  createInfrastructure() {
    this.showInfrastructureDeployingMessage();
    this.createInfrastructureDisabled = true;
    this.infrastructureService.postCreateInfrastructure().subscribe(
      (response: {}) => {
        this.pollInfrastructureStatus();
      },
      (errorResponse: HttpErrorResponse) => {
        const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
        console.error(errorTitle + '\n' + HttpErrorResponseFormatter.getMessage(errorResponse));
        this.snackBar.open(`There was an error in the create infrastructure request: ${errorTitle}`, 'Dismiss');
        this.pollInfrastructureStatus();
      }
    );
  }

  tearDownInfrastructure() {
    if (!confirm('Are you sure you want to tear down the ' +
    'infrastructure? This will remove any existing jobs.')) {
      return;
    }
    this.showInfrastructureDeletingMessage();
    this.infrastructureService.postTearDownInfrastructure().subscribe(
      (response: {}) => {
        this.pollInfrastructureStatus();
      },
      (errorResponse: HttpErrorResponse) => {
        const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
        console.error(errorTitle + '\n' + HttpErrorResponseFormatter.getMessage(errorResponse));
        this.snackBar.open(`There was an error in the tear down infrastructure request: ${errorTitle}`, 'Dismiss');
        this.pollInfrastructureStatus();
      }
    );
  }
}
