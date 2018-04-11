import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';

class BandwidthControl {
  public maxBandwidth: number;
  public maxEnabled: boolean;

  constructor(enabled: boolean, bandwidth: number) {
    this.maxEnabled = enabled
    this.maxBandwidth = bandwidth
  }

  copy():BandwidthControl {
    return new BandwidthControl(this.maxEnabled, this.maxBandwidth)
  }
}

@Component({
  selector: 'app-bandwidth-control',
  templateUrl: './bandwidth-control.component.html',
  styleUrls: ['./bandwidth-control.component.css']
})

export class BandwidthControlComponent implements OnInit {
   @ViewChild('maxBandwidthInput') maxBandwidthInput: ElementRef;
  bandwidthControl: BandwidthControl;
  initialControl: BandwidthControl;
  writingValues: boolean;
  readonly enabled = "Enabled";
  readonly disabled = "Disabled";

  constructor() { }

  ngOnInit() {
    this.initialControl = this.getInitialSettings()
    this.bandwidthControl = this.initialControl.copy()
  }

  getInitialSettings() {
    // TODO get current bandwidth settings for the project.
    return new BandwidthControl(false, null)
  }

  async writeValues(enabled: boolean, bandwidth: number) {
    // TODO send new bandwidth settings via transfer service API and update the
    // current value displayed to the user when the call returns successfully.
    console.log(this.bandwidthControl);
    this.writingValues = true
    await sleep(2000);
    this.writingValues = false
    this.initialControl = new BandwidthControl(enabled, bandwidth)
    this.bandwidthControl = this.initialControl.copy()
  }

  canSetCap() {
    return !this.writingValues &&
        this.bandwidthControl.maxBandwidth != null &&
        this.initialControl.maxBandwidth != this.bandwidthControl.maxBandwidth
  }
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}
