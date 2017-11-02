import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnDestroy, OnInit } from '@angular/core';
import { MatDialog, MatSnackBar } from '@angular/material';
import { ActivatedRoute } from '@angular/router';
import { IntervalObservable } from 'rxjs/observable/IntervalObservable';

import { ErrorDialogComponent } from '../util/error-dialog/error-dialog.component';
import { ErrorDialogContent } from '../util/error-dialog/error-dialog.resources';
import { HttpErrorResponseFormatter } from '../util/error.resources';
import { INFRA_STATUS, InfrastructureStatus } from './infrastructure.resources';
import { InfrastructureService } from './infrastructure.service';

const UPDATE_STATUS_POLLING_INTERVAL_MILISECONDS = 10000;

@Component({
  selector: 'app-infrastructure',
  templateUrl: './infrastructure.component.html',
  styleUrls: ['./infrastructure.component.css'],
})

export class InfrastructureComponent implements OnInit, OnDestroy {

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
  overallInfrastructureStatus: string;
  overallPubSubStatus: string;

  /**
   * Whether this component is alive or not. Used to determine if it should keep polling the
   * infrastructure status.
   */
  alive = false;

  /**
   * Whether the app is waiting for create infrastructure or teardown infrastructure server
   * response.
   */
  isWaitingForCreateOrTeardownResponse = false;

  constructor(private readonly infrastructureService: InfrastructureService,
              private readonly snackBar: MatSnackBar,
              private readonly route: ActivatedRoute,
              private readonly dialog: MatDialog) {
    this.projectId = route.snapshot.queryParams.project;
  }

  ngOnInit() {
    this.alive = true;
    this.loadInfrastructureStatus();
    IntervalObservable.create(UPDATE_STATUS_POLLING_INTERVAL_MILISECONDS)
      .takeWhile(() => this.alive)
      .subscribe(() => {
        /**
         * Only poll when the infrastructure is deploying or deleting, and don't poll if the app
         * is waiting for a create infrastructure or tear down infrastructure response. Also don't
         * poll if the infrastructure status is loading initially.
         */
        if ((this.showInfrastructureDeploying || this.showInfrastructureDeleting) &&
          (!this.isWaitingForCreateOrTeardownResponse) &&
          (!this.showInfrastructureStatusLoading)) {
          this.pollInfrastructureStatus();
        }
      });
  }

  ngOnDestroy() {
    this.alive = false;
  }

  /**
   * Updates the overall pubsub status variable with a new response.
   *
   * @param response The infrastructure status response
   */
  private updateOverallPubSubStatus(response: InfrastructureStatus) {
    const statusList = [response.pubsubStatus.list, response.pubsubStatus.listProgress,
      response.pubsubStatus.uploadGCS, response.pubsubStatus.uploadGCSProgress,
      response.pubsubStatus.loadBigQuery, response.pubsubStatus.loadBigQueryProgress];
    this.overallPubSubStatus = InfrastructureService.getOverallStatus(statusList);
  }

  /**
   * Loads the infrastructure status initially.
   */
  private loadInfrastructureStatus() {
    this.showInfrastructureStatusLoading = true;
    this.infrastructureService.getInfrastructureStatus().subscribe(
      (response: InfrastructureStatus) => {
        this.infrastructureStatus = response;
        this.updateInfrastructureStatusVariables(response);
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
        this.updateInfrastructureStatusVariables(response);
      },
      (errorResponse: HttpErrorResponse) => {
        const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
        console.error(errorTitle + '\m' + HttpErrorResponseFormatter.getMessage(errorResponse));
        this.snackBar.open(`There was an error polling the infrastructure status` +
          `(open browser console for details): ${errorTitle}`, 'Dismiss');
      }
    );
  }
  /**
   * Updates all of the related variables after getting a new infrastructure status.
   *
   * @param response A get infrastructure response.
   */
  private updateInfrastructureStatusVariables(response: InfrastructureStatus) {
    this.resetMessageVariables();
    this.updateOverallPubSubStatus(response);
    const statusList = [this.infrastructureStatus.spannerStatus, this.infrastructureStatus.dcpStatus,
      this.overallPubSubStatus];
    this.overallInfrastructureStatus = InfrastructureService.getOverallStatus(statusList);
    this.updateCreateTearDownButtons();

    switch (this.overallInfrastructureStatus) {
      case INFRA_STATUS.RUNNING:
        this.showInfrastructureStatusOk = true;
        break;
      case INFRA_STATUS.NOT_FOUND:
        this.showInfrastructureNotFound = true;
        break;
      case INFRA_STATUS.DEPLOYING:
        this.showInfrastructureDeploying = true;
        break;
      case INFRA_STATUS.DELETING:
        this.showInfrastructureDeleting = true;
        break;
      case INFRA_STATUS.FAILED:
        this.showInfrastructureFailed = true;
        break;
      case INFRA_STATUS.UNKNOWN:
        this.showInfrastructureUnknown = true;
        break;
      default:
        this.showCouldNotDetermineInfrastructure = true;
        break;
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

  /**
   * Updates the "disabled" state of the create and tear down buttons. Should be called updating
   * the overall infrastructure status.
   */
  private updateCreateTearDownButtons() {
    if (this.overallInfrastructureStatus === INFRA_STATUS.NOT_FOUND) {
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
    this.isWaitingForCreateOrTeardownResponse = true;
    this.showInfrastructureDeployingMessage();
    this.createInfrastructureDisabled = true;
    this.infrastructureService.postCreateInfrastructure().subscribe(
      (response: {}) => {
        this.isWaitingForCreateOrTeardownResponse = false;
      },
      (errorResponse: HttpErrorResponse) => {
        this.isWaitingForCreateOrTeardownResponse = false;
        const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
        const errorMessage = HttpErrorResponseFormatter.getMessage(errorResponse);
        const errorContent: ErrorDialogContent = {
          errorTitle: errorTitle,
          errorMessage: errorMessage
        };
        this.dialog.open(ErrorDialogComponent, {
          data: errorContent
        });
        this.pollInfrastructureStatus();
      }
    );
  }

  tearDownInfrastructure() {
    if (!confirm('Are you sure you want to tear down the ' +
    'infrastructure? This will remove any existing jobs.')) {
      return;
    }
    this.isWaitingForCreateOrTeardownResponse = true;
    this.showInfrastructureDeletingMessage();
    this.infrastructureService.postTearDownInfrastructure().subscribe(
      (response: {}) => {
        this.isWaitingForCreateOrTeardownResponse = false;
      },
      (errorResponse: HttpErrorResponse) => {
        this.isWaitingForCreateOrTeardownResponse = false;
        const errorTitle = HttpErrorResponseFormatter.getTitle(errorResponse);
        const errorMessage = HttpErrorResponseFormatter.getMessage(errorResponse);
        const errorContent: ErrorDialogContent = {
          errorTitle: errorTitle,
          errorMessage: errorMessage
        };
        this.dialog.open(ErrorDialogComponent, {
          data: errorContent
        });
        this.pollInfrastructureStatus();
      }
    );
  }
}
