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

export interface MaxBandwidthRequest {
  projectId: string;
  hasMaxBandwidth: boolean;
  bandwidth: number;
}

export interface MaxBandwidthResponse {
  projectId: string;
  hasMaxBandwidth: boolean;
  bandwidth: number;
}
