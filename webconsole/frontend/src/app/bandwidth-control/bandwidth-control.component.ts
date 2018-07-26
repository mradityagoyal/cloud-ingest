import { HttpErrorResponse } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { BandwidthControlService } from './bandwidth-control.service';
import { MaxBandwidthResponse } from './bandwidth-control.resources';

export class BandwidthControl {
  public maxBandwidth: number;
  public maxEnabled: boolean;

  constructor(enabled: boolean, bandwidth: number) {
    this.maxEnabled = enabled;
    this.maxBandwidth = bandwidth;
  }

  copy(): BandwidthControl {
    return new BandwidthControl(this.maxEnabled, this.maxBandwidth);
  }
}

@Component({
  selector: 'app-bandwidth-control',
  templateUrl: './bandwidth-control.component.html',
  styleUrls: [
    './bandwidth-control.component.css',
    '../app.component.css',
  ]
})

export class BandwidthControlComponent implements OnInit {
   @ViewChild('maxBandwidthInput') maxBandwidthInput: ElementRef;
  bandwidthControl: BandwidthControl;
  initialControl: BandwidthControl;
  loading: boolean;
  showLoadingError: boolean;
  showSettingError: boolean;

  constructor(private readonly bwService: BandwidthControlService) { }

  ngOnInit() {
    this.getInitialSettings();
    this.initialControl = new BandwidthControl(false, null);
    this.bandwidthControl = new BandwidthControl(false, null);
  }

  applyResponse(r: MaxBandwidthResponse) {
    const bandwidth = r.hasMaxBandwidth ? Number(r.bandwidth) >> 20 : null;
    this.initialControl = new BandwidthControl(r.hasMaxBandwidth, bandwidth);
    this.bandwidthControl = this.initialControl.copy();
    this.loading = false;
  }

  getInitialSettings() {
    this.loading = true;
    this.bwService.getProjectMaxBandwidth().subscribe(
      (response: MaxBandwidthResponse) => {
        this.showLoadingError = false;
        this.applyResponse(response);
      },
      (error: HttpErrorResponse) => {
        this.showLoadingError = true;
      }
    );
  }

  onSubmit(enabled: boolean, bandwidth: number) {
    this.loading = true;
    if (bandwidth < 0) {
      bandwidth = 0;
    }
    this.bwService.postProjectMaxBandwidth(enabled, bandwidth << 20).subscribe(
      (response: MaxBandwidthResponse) => {
        this.showSettingError = false;
        this.applyResponse(response);
      },
      (error: HttpErrorResponse) => {
        this.showSettingError = true;
      }
    );
  }

  canSetCap() {
    return !this.loading &&
        this.bandwidthControl.maxBandwidth != null &&
        this.initialControl.maxBandwidth !== this.bandwidthControl.maxBandwidth;
  }
}
