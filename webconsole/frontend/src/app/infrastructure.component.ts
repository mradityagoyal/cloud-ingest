import { Component, OnInit } from '@angular/core';
import { InfrastructureStatus } from './api.resources';
import { InfrastructureService, INFRA_STATUS } from './infrastructure.service';
import { HttpErrorResponse } from '@angular/common/http';

@Component({
  selector: 'app-infrastructure',
  templateUrl: './infrastructure.component.html',
})

export class InfrastructureComponent implements OnInit {

  infrastructureStatus: InfrastructureStatus;
  showUpdateInfrastructureError = false;
  showUpdateInfrastructureLoading = false;
  showInfrastructureNotFound = false;
  showInfrastructureStatusOk = false;
  showInfrastructureDeploying = false;
  showInfrastructureDeleting = false;
  showInfrastructureFailed = false;
  showInfrastructureUnknown = false;
  showCouldNotDetermineInfrastructure = false;
  updateInfrastructureError: string;
  updateInfrastructureErrorMessage: string;

  constructor(private readonly infrastructureService: InfrastructureService) { }

  ngOnInit() {
    // TODO(b/65736612): Set an interval timer to update the infrastructure status.
    this.updateInfrastructureStatus();
  }

  /**
   * Updates the infrastructure status.
   */
  private updateInfrastructureStatus() {
    this.showUpdateInfrastructureLoading = true;
    this.infrastructureService.getInfrastructureStatus().subscribe(
      (response: InfrastructureStatus) => {
        this.infrastructureStatus = response;
        this.updateInfrastructureStatusMessage(response);
        this.showUpdateInfrastructureError = false;
        this.showUpdateInfrastructureLoading = false;
      },
      (errorResponse: HttpErrorResponse) => {
        this.updateInfrastructureError = errorResponse.error;
        this.updateInfrastructureErrorMessage = errorResponse.message;
        this.showUpdateInfrastructureError = true;
        this.showUpdateInfrastructureLoading = false;
      }
    );
  }

  private updateInfrastructureStatusMessage(response) {
    this.showInfrastructureStatusOk = this.showInfrastructureNotFound =
    this.showInfrastructureDeploying = this.showInfrastructureDeleting =
    this.showInfrastructureFailed = this.showInfrastructureUnknown =
    this.showCouldNotDetermineInfrastructure = false;

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
}
