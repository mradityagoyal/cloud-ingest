import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import {throwError as observableThrowError,  Observable, of, never } from 'rxjs';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { AngularMaterialImporterModule } from '../angular-material-importer/angular-material-importer.module';
import { FAKE_HTTP_ERROR } from '../util/common.test-util';
import { PulseService } from './pulse.service';
import { PulseComponent } from './pulse.component';

let serviceStub: PulseServiceStub;
const FAKE_AGENT = {
  agentId: 'fake-agent-id',
  lastPulseReceived: 'fake-time-stamp',
};

class PulseServiceStub {
  public getAgents = jasmine.createSpy('getAgents');
}

describe('PulseComponent', () => {

  beforeEach(async(() => {
    serviceStub = new PulseServiceStub();
    serviceStub.getAgents.and.returnValue(of(FAKE_AGENT));
    TestBed.configureTestingModule({
      declarations: [ PulseComponent ],
      providers: [
        {provide: PulseService, useValue: serviceStub}
      ],
      imports: [
        BrowserAnimationsModule,
        AngularMaterialImporterModule
      ],
    })
    .compileComponents();
  }));

  it('should create the pulse component', () => {
    const fixture = TestBed.createComponent(PulseComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component).toBeTruthy();
  });

  it('should show a loading spinner while the agents are loading', async(() => {
    const fixture = TestBed.createComponent(PulseComponent);
    const component = fixture.debugElement.componentInstance;
    component.showLoadingSpinner = true;
    serviceStub.getAgents.and.returnValue(never());
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const spinner = compiled.querySelector('mat-spinner');
      expect(spinner).not.toBeNull();
    });
  }));

  it('should not show a loading spinner when agents are running', async(() => {
    const fixture = TestBed.createComponent(PulseComponent);
    const component = fixture.debugElement.componentInstance;
    component.loading = false;
  }));
});
