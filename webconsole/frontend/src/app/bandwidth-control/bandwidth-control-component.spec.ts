
import {throwError as observableThrowError,  Observable, of, never } from 'rxjs';


import { async, TestBed } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import { AngularMaterialImporterModule } from '../angular-material-importer/angular-material-importer.module';
import { FAKE_HTTP_ERROR } from '../util/common.test-util';
import { BandwidthControl, BandwidthControlComponent } from './bandwidth-control.component';
import { BandwidthControlService } from './bandwidth-control.service';



let serviceStub: BandwidthControlServiceStub;
const FAKE_RESPONSE = {
  projectId: 'fake-project',
  hasMaxBandwidth: true,
  bandwidth: 1234,
};

class BandwidthControlServiceStub {
  public getProjectMaxBandwidth = jasmine.createSpy('getProjectMaxBandwidth');
  public postProjectMaxBandwidth = jasmine.createSpy('postProjectMaxBandwidth');
}

describe('BandwidthControlComponent', () => {

  beforeEach(async(() => {
    serviceStub = new BandwidthControlServiceStub();
    serviceStub.getProjectMaxBandwidth.and.returnValue(of(FAKE_RESPONSE));
    TestBed.configureTestingModule({
      declarations: [
        BandwidthControlComponent
      ],
      providers: [
        {provide: BandwidthControlService, useValue: serviceStub}
      ],
      imports: [
        FormsModule,
        BrowserAnimationsModule,
        AngularMaterialImporterModule
      ],
    }).compileComponents();
  }));

  it('should create the bandwidth control component', async(() => {
    const fixture = TestBed.createComponent(BandwidthControlComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component).toBeTruthy();
  }));

  it('should show a loading spinner while max bandwidth is loading', async(() => {
    const fixture = TestBed.createComponent(BandwidthControlComponent);
    const component = fixture.debugElement.componentInstance;
    component.loading = true;
    serviceStub.getProjectMaxBandwidth.and.returnValue(never());
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const spinner = compiled.querySelector('mat-progress-spinner');
      expect(spinner).not.toBeNull();
    });
  }));

  it('should not show a loading spinner when max bandwidth is returned', async(() => {
    const fixture = TestBed.createComponent(BandwidthControlComponent);
    const component = fixture.debugElement.componentInstance;
    component.loading = false;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const spinner = compiled.querySelector('mat-progress-spinner');
      expect(spinner).toBeNull();
    });
  }));

  it('ngOnInit should call bandwidth control service get max bandwidth', async(() => {
    const fixture = TestBed.createComponent(BandwidthControlComponent);
    const component = fixture.debugElement.componentInstance;
    component.ngOnInit();
    expect(serviceStub.getProjectMaxBandwidth.calls.count()).toEqual(1);
  }));

  it('onSubmit should call bandwidth control service post max bandwidth', async(() => {
    const fixture = TestBed.createComponent(BandwidthControlComponent);
    const component = fixture.debugElement.componentInstance;
    serviceStub.postProjectMaxBandwidth.and.returnValue(of(FAKE_RESPONSE));
    const compiled = fixture.debugElement.nativeElement;
    const button = compiled.querySelector('.ingest-submit-button');

    component.bandwidthControl = new BandwidthControl(true, 123);
    button.click();
    expect(serviceStub.postProjectMaxBandwidth.calls.count()).toEqual(1);
    expect(serviceStub.postProjectMaxBandwidth.calls.first().args[0]).toEqual(true);
    expect(serviceStub.postProjectMaxBandwidth.calls.first().args[1]).toEqual(123 << 20);
  }));

  it('onSubmit should not send negative bandwidth', async(() => {
    const fixture = TestBed.createComponent(BandwidthControlComponent);
    const component = fixture.debugElement.componentInstance;
    serviceStub.postProjectMaxBandwidth.and.returnValue(Observable.of(FAKE_RESPONSE));
    const compiled = fixture.debugElement.nativeElement;
    const button = compiled.querySelector('.ingest-submit-button');

    component.bandwidthControl = new BandwidthControl(true, -123);
    button.click();
    expect(serviceStub.postProjectMaxBandwidth.calls.count()).toEqual(1);
    expect(serviceStub.postProjectMaxBandwidth.calls.first().args[0]).toEqual(true);
    expect(serviceStub.postProjectMaxBandwidth.calls.first().args[1]).toEqual(0);
  }));

  it('should show error on loading error', async(() => {
    serviceStub.getProjectMaxBandwidth.and.returnValue(observableThrowError(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(BandwidthControlComponent);
    const component = fixture.debugElement.componentInstance;

    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const errorMessage = compiled.querySelector('.ingest-error-message');
      expect(errorMessage).not.toBeNull();
      expect(errorMessage.textContent).toContain('loading');
    });
  }));

  it('should show error on submit error', async(() => {
    serviceStub.postProjectMaxBandwidth.and.returnValue(observableThrowError(FAKE_HTTP_ERROR));
    const fixture = TestBed.createComponent(BandwidthControlComponent);
    const component = fixture.debugElement.componentInstance;
    const compiled = fixture.debugElement.nativeElement;
    const button = compiled.querySelector('.ingest-submit-button');

    component.bandwidthControl = new BandwidthControl(true, 123);
    button.click();
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const errorMessage = compiled.querySelector('.ingest-error-message');
      expect(errorMessage).not.toBeNull();
      expect(errorMessage.textContent).toContain('setting');
    });
  }));
});
